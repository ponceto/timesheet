/*
 * calendar.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "maps"
    "slices"
)

// ---------------------------------------------------------------------------
// Calendar declaration
// ---------------------------------------------------------------------------

type Calendar struct {
    Revision    int64                 `json:"revision"`
    Settings    UserSettings          `json:"settings"`
    Days        map[Date]DayEntry     `json:"days"`
    Adjustments []Adjustment          `json:"adjustments"`
    Months      map[Month]MonthRecord `json:"months"`
    Ledger      []LedgerEntry         `json:"ledger"`
}

var (
    ErrStaleRevision      = errors.New("activity: the data changed in the meantime, reload the page")
    ErrCalendarTooLarge   = errors.New("activity: the data of this account exceed the allowed size")
    ErrConfigurationInUse = errors.New("activity: the configuration would invalidate existing data")
    ErrHistoryRewritten   = errors.New("activity: a validated month cannot be rewritten")
)

// ---------------------------------------------------------------------------
// Calendar implementation
// ---------------------------------------------------------------------------

func NewCalendar(start Date) Calendar {
    return Calendar{Settings: UserSettings{Start: start, Counters: map[string]CounterSettings{}}, Days: map[Date]DayEntry{}, Adjustments: []Adjustment{}, Months: map[Month]MonthRecord{}, Ledger: []LedgerEntry{}}
}

func (c Calendar) Clone() Calendar {
    clone := c
    if c.Settings.PublicHolidays != nil {
        zone := *c.Settings.PublicHolidays
        clone.Settings.PublicHolidays = &zone
    }
    clone.Settings.Counters = make(map[string]CounterSettings, len(c.Settings.Counters))
    for id, settings := range c.Settings.Counters {
        clone.Settings.Counters[id] = settings.Clone()
    }
    clone.Settings.Colors = nil
    if len(c.Settings.Colors) > 0 {
        clone.Settings.Colors = maps.Clone(c.Settings.Colors)
    }
    clone.Days = maps.Clone(c.Days)
    clone.Adjustments = slices.Clone(c.Adjustments)
    clone.Months = make(map[Month]MonthRecord, len(c.Months))
    for m, record := range c.Months {
        clone.Months[m] = record.Clone()
    }
    clone.Ledger = slices.Clone(c.Ledger)
    if clone.Days == nil {
        clone.Days = map[Date]DayEntry{}
    }
    if clone.Adjustments == nil {
        clone.Adjustments = []Adjustment{}
    }
    if clone.Ledger == nil {
        clone.Ledger = []LedgerEntry{}
    }
    return clone
}

func (c Calendar) IsValidated(m Month) bool {
    return c.Months[m].Status == MonthValidated
}

func (c Calendar) LastValidated() (Month, bool) {
    last, found := Month{}, false
    for m := c.Settings.Start.Month(); c.IsValidated(m); m = m.Next() {
        last, found = m, true
    }
    return last, found
}

func (c Calendar) FirstOpen() Month {
    if last, found := c.LastValidated(); found {
        return last.Next()
    }
    return c.Settings.Start.Month()
}

func (c Calendar) Tracks(d Date) bool {
    return !d.Before(c.Settings.Start)
}

func (c Calendar) AdjustedBefore(d Date) bool {
    for _, adjustment := range c.Adjustments {
        if adjustment.Date.Before(d) {
            return true
        }
    }
    return false
}

func (c Calendar) HasHistory() bool {
    _, found := c.LastValidated()
    return found
}

func (c Calendar) KeepsHistoryOf(previous Calendar) bool {
    if len(c.Ledger) < len(previous.Ledger) {
        return false
    }
    for i, entry := range previous.Ledger {
        if !entry.Same(c.Ledger[i]) {
            return false
        }
    }
    last, history := previous.LastValidated()
    if history && (c.Settings.Start != previous.Settings.Start) {
        return false
    }
    for _, entry := range c.Ledger[len(previous.Ledger):] {
        reopening := (entry.Kind == KindReversal) && history && (entry.Month == last) && !c.IsValidated(last)
        validation := (entry.Kind != KindReversal) && !previous.IsValidated(entry.Month) && c.IsValidated(entry.Month)
        if !reopening && !validation {
            return false
        }
    }
    for m, record := range previous.Months {
        if !previous.IsValidated(m) {
            continue
        }
        if (!c.IsValidated(m) && (m != last)) || !c.sameDays(previous, m) || !c.sameAdjustments(previous, m) {
            return false
        }
        if c.IsValidated(m) && !c.sameRecord(record, m) {
            return false
        }
    }
    return true
}

func (c Calendar) sameRecord(previous MonthRecord, m Month) bool {
    record := c.Months[m]
    if !record.ValidatedAt.Equal(previous.ValidatedAt) || !maps.Equal(record.Rules, previous.Rules) || (len(record.History) != len(previous.History)) {
        return false
    }
    for i, event := range previous.History {
        if !event.At.Equal(record.History[i].At) || (event.Action != record.History[i].Action) || (event.Reason != record.History[i].Reason) {
            return false
        }
    }
    if (record.Indicators == nil) || (previous.Indicators == nil) {
        return record.Indicators == previous.Indicators
    }
    return record.Indicators.Same(*previous.Indicators)
}

func (c Calendar) sameDays(previous Calendar, m Month) bool {
    for d, entry := range previous.Days {
        if (d.Month() == m) && (c.Days[d] != entry) {
            return false
        }
    }
    for d, entry := range c.Days {
        if (d.Month() == m) && (previous.Days[d] != entry) {
            return false
        }
    }
    return true
}

func (c Calendar) sameAdjustments(previous Calendar, m Month) bool {
    kept := map[Adjustment]int{}
    for _, adjustment := range previous.Adjustments {
        if adjustment.Date.Month() == m {
            kept[adjustment]++
        }
    }
    for _, adjustment := range c.Adjustments {
        if adjustment.Date.Month() == m {
            kept[adjustment]--
        }
    }
    for _, count := range kept {
        if count != 0 {
            return false
        }
    }
    return true
}

func (c Calendar) UsesActivity(id string) bool {
    for _, entry := range c.Days {
        if (entry.AM == id) || (entry.PM == id) {
            return true
        }
    }
    return false
}

func (c Calendar) UsesCounter(id string) bool {
    if _, set := c.Settings.Counters[id]; set {
        return true
    }
    for _, adjustment := range c.Adjustments {
        if adjustment.Counter == id {
            return true
        }
    }
    for _, entry := range c.Ledger {
        if entry.Counter == id {
            return true
        }
    }
    for _, record := range c.Months {
        if _, mentioned := record.Rules[id]; mentioned {
            return true
        }
    }
    return false
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
