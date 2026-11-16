/*
 * calendar_test.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "testing"
    "time"
)

// ---------------------------------------------------------------------------
// Calendar tests
// ---------------------------------------------------------------------------

func TestCalendarKeepsTheHistoryOfValidatedMonths(t *testing.T) {
    holidays := NewPublicHolidayCalendar(ZoneMetropole)
    config := authorConfiguration()
    calendar := authorCalendar(t, "2026-09")
    fillMonth(t, &calendar, month(t, "2026-09"), holidays)
    fillMonth(t, &calendar, month(t, "2026-10"), holidays)
    today := day(t, "2026-11-08")
    september, err := newClosing(config, calendar).Validate(month(t, "2026-09"), fixedNow(), today)
    if err != nil {
        t.Fatal(err)
    }
    validated, err := newClosing(config, september).Validate(month(t, "2026-10"), fixedNow(), today)
    if (err != nil) || !validated.KeepsHistoryOf(september) {
        t.Fatalf("a validation keeps the history: %v", err)
    }
    reopened, err := newClosing(config, validated).Reopen(month(t, "2026-10"), "correction", fixedNow())
    if (err != nil) || !reopened.KeepsHistoryOf(validated) {
        t.Fatalf("a reopening of the last validated month keeps the history: %v", err)
    }
    if !reopened.Clone().KeepsHistoryOf(reopened) {
        t.Fatal("an unchanged calendar keeps its history")
    }
    rewritten := validated.Clone()
    rewritten.Days[day(t, "2026-09-15")] = DayEntry{AM: "CP", PM: "CP"}
    removed := validated.Clone()
    delete(removed.Days, day(t, "2026-10-14"))
    truncated := validated.Clone()
    truncated.Ledger = truncated.Ledger[:len(truncated.Ledger) - 1]
    altered := validated.Clone()
    altered.Ledger[0].Amount = q(t, "99")
    adjusted := validated.Clone()
    adjusted.Adjustments = append(adjusted.Adjustments, Adjustment{ID: "a1", Date: day(t, "2026-09-30"), Counter: "CP", Amount: q(t, "1"), Reason: "ajout"})
    credited := validated.Clone()
    credited.Ledger = append(credited.Ledger, LedgerEntry{ID: "forged", Month: month(t, "2026-09"), Date: day(t, "2026-09-30"), Counter: "CP", Kind: KindCredit, Amount: q(t, "10"), Created: fixedNow()})
    first := validated.Ledger[0]
    cancelled := validated.Clone()
    cancelled.Ledger = append(cancelled.Ledger, LedgerEntry{ID: "undo", Month: first.Month, Date: first.Date, Counter: first.Counter, Kind: KindReversal, Amount: first.Amount.Neg(), Ref: first.ID, Created: fixedNow()})
    older := validated.Clone()
    record := older.Months[month(t, "2026-09")]
    record.Status = MonthOpen
    older.Months[month(t, "2026-09")] = record
    moved := validated.Clone()
    moved.Settings.Start = day(t, "2026-12-01")
    refrozen := validated.Clone()
    record = refrozen.Months[month(t, "2026-09")]
    record.Rules = map[string]Quantity{"CP": q(t, "3")}
    refrozen.Months[month(t, "2026-09")] = record
    redated := validated.Clone()
    record = redated.Months[month(t, "2026-10")]
    record.ValidatedAt = fixedNow().Add(time.Hour)
    redated.Months[month(t, "2026-10")] = record
    recounted := validated.Clone()
    record = recounted.Months[month(t, "2026-09")]
    indicators := *record.Indicators
    indicators.ByCode = map[string]Quantity{"TT": q(t, "99")}
    record.Indicators = &indicators
    recounted.Months[month(t, "2026-09")] = record
    forgotten := validated.Clone()
    record = forgotten.Months[month(t, "2026-09")]
    record.History = nil
    forgotten.Months[month(t, "2026-09")] = record
    for name, candidate := range map[string]Calendar{"frozen indicators changed": recounted, "history of a validated month erased": forgotten, "entry added to a validated month": credited, "reversal in a month that stays validated": cancelled, "moved start": moved, "frozen rules changed": refrozen, "validation date changed": redated, "rewritten day": rewritten, "removed day": removed, "truncated ledger": truncated, "altered ledger": altered, "adjustment in a validated month": adjusted, "older month reopened": older} {
        if candidate.KeepsHistoryOf(validated) {
            t.Errorf("%s: accepted", name)
        }
    }
    opened := validated.Clone()
    opened.Days[day(t, "2026-11-02")] = DayEntry{AM: "TT", PM: "TT"}
    opened.Adjustments = append(opened.Adjustments, Adjustment{ID: "a2", Date: day(t, "2026-11-02"), Counter: "CP", Amount: q(t, "1"), Reason: "ajout"})
    if !opened.KeepsHistoryOf(validated) {
        t.Fatal("a change in an open month keeps the history")
    }
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
