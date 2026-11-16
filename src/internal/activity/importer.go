/*
 * importer.go - Copyright (c) 2023-2026 - Olivier Poncet
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 2 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package activity

import (
    "errors"
    "slices"
    "strconv"
    "strings"
    "time"
)

// ---------------------------------------------------------------------------
// Importer declaration
// ---------------------------------------------------------------------------

type Importer struct {
    config      Configuration
    clock       Clock
    identifiers *IdentifierGenerator
    location    *time.Location
}

type adjustmentKey struct {
    date    Date
    counter string
    amount  Quantity
    reason  string
}

const maxImportedDays = 366

var ErrMonthNotImported = errors.New("this month was not imported")

// ---------------------------------------------------------------------------
// Importer implementation
// ---------------------------------------------------------------------------

func NewImporter(config Configuration, clock Clock, identifiers *IdentifierGenerator, location *time.Location) *Importer {
    return &Importer{config: config, clock: clock, identifiers: identifiers, location: location}
}

func (i *Importer) Run(calendar Calendar, imported ImportFile, options ImportOptions, today Date, horizon Month) (ImportReport, error) {
    next := calendar.Clone()
    report := ImportReport{Problems: i.apply(&next, imported.Settings), Warnings: []ImportProblem{}, Settings: imported.Settings != nil}
    if imported.Settings != nil {
        for _, given := range imported.Settings.Counters {
            if given != nil {
                report.Counters++
            }
        }
    }
    days := map[Date]DayEntry{}
    for d, entry := range imported.Days {
        entry.Comment = strings.TrimSpace(entry.Comment)
        days[d] = entry
        if !entry.Empty() && !d.Month().Before(options.From) && !d.Month().After(options.To) {
            report.Dates = append(report.Dates, d)
        }
    }
    slices.SortFunc(report.Dates, Date.Compare)
    adjustments := []ImportAdjustment{}
    for _, adjustment := range imported.Adjustments {
        adjustment.Reason = strings.TrimSpace(adjustment.Reason)
        if adjustment.Date.IsZero() || (!adjustment.Date.Month().Before(options.From) && !adjustment.Date.Month().After(options.To)) {
            adjustments = append(adjustments, adjustment)
        }
    }
    report.Adjustments = len(adjustments)
    if (len(report.Dates) == 0) && (len(adjustments) == 0) && (imported.Settings == nil) {
        report.Problems = append(report.Problems, ImportProblem{Kind: importNothing})
    }
    if (len(report.Dates) > 0) && !next.Tracks(report.Dates[0]) {
        if next.HasHistory() {
            report.Problems = append(report.Problems, ImportProblem{Kind: importStartBeforeValidated})
        } else {
            if report.Dates[0] != calendar.Settings.Start {
                report.Warnings = append(report.Warnings, ImportProblem{Date: report.Dates[0], Previous: calendar.Settings.Start, Kind: importStartMoved})
            }
            next.Settings.Start = report.Dates[0]
        }
    }
    if (next.Settings.Start != calendar.Settings.Start) && next.AdjustedBefore(next.Settings.Start) {
        report.Problems = append(report.Problems, ImportProblem{Date: next.Settings.Start, Kind: importStartAfterAdjustment})
    }
    book := NewBookkeeper(i.config, next)
    for _, d := range report.Dates {
        entry := days[d]
        existing := calendar.Days[d]
        switch {
            case d.Month().After(horizon):
                report.Problems = append(report.Problems, ImportProblem{Date: d, Kind: importBeyondHorizon})
                continue
            case next.IsValidated(d.Month()):
                report.Problems = append(report.Problems, ImportProblem{Date: d, Kind: importMonthValidated})
                continue
            case !existing.Empty() && !options.Replace:
                report.Problems = append(report.Problems, ImportProblem{Date: d, Kind: importEntryExists})
                continue
        }
        for _, problem := range book.CheckEntry(d, existing, entry) {
            report.Problems = append(report.Problems, ImportProblem{Date: d, Kind: importGridRefusal, Detail: problem})
        }
        next.Days[d] = entry
    }
    present := map[adjustmentKey]int{}
    for _, stored := range next.Adjustments {
        present[i.key(stored.Date, stored.Counter, stored.Amount, stored.Reason)]++
    }
    for _, adjustment := range adjustments {
        found := adjustment.Problems(i.config, next, horizon)
        report.Problems = append(report.Problems, found...)
        if len(found) > 0 {
            continue
        }
        rule, _ := i.config.Counter(adjustment.Counter)
        if key := i.key(adjustment.Date, adjustment.Counter, adjustment.Amount, adjustment.Reason); present[key] > 0 {
            present[key]--
            report.Warnings = append(report.Warnings, ImportProblem{Date: adjustment.Date, Kind: importAdjustmentPresent})
            continue
        }
        id, err := i.identifiers.Next()
        if err != nil {
            return report, err
        }
        next.Adjustments = append(next.Adjustments, adjustment.Adjustment(id, rule))
    }
    report.Calendar = next
    if (len(report.Problems) > 0) || (len(report.Dates) == 0) {
        return report, nil
    }
    report.Missing = i.missing(next, report.First(), report.Last(), today)
    if options.Validate {
        report.Calendar = i.validate(&report, next, today)
    }
    return report, nil
}

func (i *Importer) key(date Date, counter string, amount Quantity, reason string) adjustmentKey {
    return adjustmentKey{date: date, counter: counter, amount: amount.reduced(), reason: reason}
}

func (i *Importer) apply(calendar *Calendar, settings *ImportSettings) []ImportProblem {
    problems := []ImportProblem{}
    if settings == nil {
        return problems
    }
    locked := calendar.HasHistory()
    if settings.PublicHolidays != nil {
        if zone := *settings.PublicHolidays; zone.Valid() {
            calendar.Settings.PublicHolidays = &zone
        } else {
            problems = append(problems, ImportProblem{Kind: importPublicHolidaysUnknown, Detail: strconv.QuoteToASCII(string(zone))})
        }
    }
    if (settings.Start != nil) && (*settings.Start != calendar.Settings.Start) {
        switch {
            case locked:
                problems = append(problems, ImportProblem{Kind: importStartLocked})
            default:
                calendar.Settings.Start = *settings.Start
        }
    }
    if calendar.Settings.Counters == nil {
        calendar.Settings.Counters = map[string]CounterSettings{}
    }
    limit := NewQuantity(maxImportedDays, 1)
    ids := []string{}
    for id := range settings.Counters {
        ids = append(ids, id)
    }
    slices.Sort(ids)
    for _, id := range ids {
        given := settings.Counters[id]
        if given == nil {
            continue
        }
        rule, known := i.config.Counter(id)
        if !known || rule.Archived {
            problems = append(problems, ImportProblem{Kind: importCounterUnknown, Detail: strconv.QuoteToASCII(id)})
            continue
        }
        current := calendar.Settings.Counters[id]
        if given.Enabled != nil {
            current.Enabled = given.Enabled
        }
        if (given.Amount != nil) && ((given.Amount.Sign() < 0) || (given.Amount.Cmp(limit) > 0)) {
            problems = append(problems, ImportProblem{Kind: importCounterAmount, Detail: rule.Label})
        }
        current.Amount = given.Amount
        initial := current.Initial
        if given.Initial != nil {
            initial = *given.Initial
        }
        if initial.Abs().Cmp(limit) > 0 {
            problems = append(problems, ImportProblem{Kind: importCounterInitial, Detail: rule.Label})
        }
        if !initial.Equal(current.Initial) {
            if locked {
                problems = append(problems, ImportProblem{Kind: importInitialLocked, Detail: rule.Label})
            } else {
                current.Initial = initial
            }
        }
        calendar.Settings.Counters[id] = current
    }
    return problems
}

func (i *Importer) missing(calendar Calendar, first Month, last Month, today Date) []ImportMissing {
    book := NewBookkeeper(i.config, calendar)
    missing := []ImportMissing{}
    for m := first; !m.After(last); m = m.Next() {
        dates := []Date{}
        for d := m.First(); !d.After(m.Last()) && d.Before(today); d = d.AddDays(1) {
            if (book.DayType(d, Morning, today) == DayMissing) || (book.DayType(d, Afternoon, today) == DayMissing) {
                dates = append(dates, d)
            }
        }
        if len(dates) > 0 {
            missing = append(missing, ImportMissing{Month: m, Dates: dates})
        }
    }
    return missing
}

func (i *Importer) validate(report *ImportReport, calendar Calendar, today Date) Calendar {
    imported := map[Month]bool{}
    for _, d := range report.Dates {
        imported[d.Month()] = true
    }
    last := report.Last()
    for m := calendar.FirstOpen(); m.Before(today.Month()) && !m.After(last); m = m.Next() {
        if !imported[m] {
            report.Stopped, report.StopReason = m, ErrMonthNotImported
            break
        }
        closing := NewMonthClosing(NewBookkeeper(i.config, calendar), i.identifiers)
        end := m.Last().Time()
        at := time.Date(end.Year(), end.Month(), end.Day(), 23, 59, 59, 0, i.location).UTC()
        if history := calendar.Months[m].History; (len(history) > 0) && history[len(history) - 1].At.After(at) {
            at = i.clock.Now().UTC()
        }
        next, err := closing.Validate(m, at, today)
        if err != nil {
            report.Stopped, report.StopReason = m, err
            break
        }
        calendar = next
        report.Validated = append(report.Validated, m)
    }
    return calendar
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
