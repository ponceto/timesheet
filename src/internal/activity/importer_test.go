/*
 * importer_test.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "crypto/rand"
    "strings"
    "testing"
    "time"
)

// ---------------------------------------------------------------------------
// Importer tests
// ---------------------------------------------------------------------------

func TestExportedFileGivesBackTheSameBalances(t *testing.T) {
    config := authorConfiguration()
    holidays := NewPublicHolidayCalendar(ZoneMetropole)
    calendar := authorCalendar(t, "2026-01")
    for m := month(t, "2026-01"); !m.After(month(t, "2026-03")); m = m.Next() {
        fillMonth(t, &calendar, m, holidays)
    }
    amount := q(t, "8/12")
    calendar.Settings.Counters["RTT"] = CounterSettings{Enabled: calendar.Settings.Counters["RTT"].Enabled, Amount: &amount}
    alsace := ZoneAlsaceMoselle
    calendar.Settings.PublicHolidays = &alsace
    calendar.Adjustments = append(calendar.Adjustments, Adjustment{ID: "a1", Date: day(t, "2026-02-27"), Counter: "CP", Amount: q(t, "0.5"), Reason: "Alignement sur le bulletin de paie"})
    data, err := NewImportFileOf(calendar, config).Encode()
    if err != nil {
        t.Fatal(err)
    }
    text := string(data)
    position := 0
    for _, rule := range config.Counters {
        next := strings.Index(text, `"` + rule.ID + `": {`)
        if next < position {
            t.Fatalf("counter %s out of the order of the configuration", rule.ID)
        }
        position = next
    }
    for _, expected := range []string{`"CP": {
        "enabled": true,
        "amount": "2.08",
        "initial": "0"
      }`, `"RTT": {
        "enabled": false,
        "amount": "8/12",
        "initial": "0"
      }`, `"TPC": {
        "enabled": true,
        "amount": "7/12",
        "initial": "0"
      }`, `"public_holidays": "fr-alsace-moselle"`, `"comment": null`, `"adjustments": [`} {
        if !strings.Contains(text, expected) {
            t.Errorf("the export must write %s", expected)
        }
    }
    exported, err := NewImportFile(data)
    if (err != nil) || (len(exported.Days) != len(calendar.Days)) || (len(exported.Adjustments) != 1) || (exported.Settings.Counters["RTT"].Amount.String() != "8/12") {
        t.Fatalf("exported file: %v", err)
    }
    today := day(t, "2026-04-15")
    options := NewImportOptions()
    options.Validate = true
    importer := NewImporter(config, NewManualClock(time.Date(2026, time.April, 15, 10, 0, 0, 0, time.UTC)), NewIdentifierGenerator(rand.Reader), time.UTC)
    report, err := importer.Run(NewCalendar(day(t, "2026-04-01")), exported, options, today, month(t, "2028-04"))
    if (err != nil) || (len(report.Problems) != 0) || (len(report.Validated) != 3) || (len(report.Warnings) != 0) || (report.Calendar.Settings.Start != day(t, "2026-01-01")) || (report.Calendar.Settings.PublicHolidays == nil) || (*report.Calendar.Settings.PublicHolidays != alsace) {
        t.Fatalf("import of the export: %v %+v %v %+v", err, report.Problems, report.Validated, report.Warnings)
    }
    before := NewBookkeeper(config, calendar).Journal(month(t, "2026-03"))
    after := NewBookkeeper(config, report.Calendar).Journal(month(t, "2026-03"))
    for _, id := range []string{"CP", "RTT", "TPC", "TT"} {
        if !before.Balance(id, day(t, "2026-03-31")).Equal(after.Balance(id, day(t, "2026-03-31"))) {
            t.Errorf("%s: %s before, %s after", id, before.Balance(id, day(t, "2026-03-31")), after.Balance(id, day(t, "2026-03-31")))
        }
    }
}

func TestImportProblemsSpeakEnglishAndFrench(t *testing.T) {
    cases := []struct {
        problem ImportProblem
        text    string
        label   string
    }{
        {ImportProblem{Date: NewDate(2029, time.January, 3), Kind: importBeyondHorizon}, "2029-01-03: beyond the planning horizon", "03/01/2029 : au-delà de l'horizon de planification"},
        {ImportProblem{Date: NewDate(2026, time.November, 3), Kind: importAdjustmentCounter, Detail: `"NOPE"`}, `adjustment of 2026-11-03: counter "NOPE" unknown or archived`, `Ajustement du 03/11/2026 : compteur "NOPE" inconnu ou archivé`},
        {ImportProblem{Kind: importCounterAmount, Detail: "Congés payés"}, "Congés payés: invalid accrual amount (between 0 and 366)", "Congés payés : montant d'acquisition invalide (entre 0 et 366)"},
        {ImportProblem{Kind: importNothing}, "no day to import", "Le fichier ne contient aucun jour à importer"},
        {ImportProblem{Date: NewDate(2026, time.January, 1), Previous: NewDate(2026, time.April, 1), Kind: importStartMoved}, "tracking start moved from 2026-04-01 to 2026-01-01", "Date de début avancée du 01/04/2026 au 01/01/2026"},
    }
    for _, c := range cases {
        if (c.problem.Text() != c.text) || (c.problem.Label() != c.label) {
            t.Errorf("%q / %q", c.problem.Text(), c.problem.Label())
        }
    }
}

func TestImportedPublicHolidaysAreChecked(t *testing.T) {
    importer := NewImporter(authorConfiguration(), NewManualClock(time.Date(2026, time.April, 15, 10, 0, 0, 0, time.UTC)), NewIdentifierGenerator(rand.Reader), time.UTC)
    calendar := authorCalendar(t, "2026-01")
    guyane := ZoneGuyane
    calendar.Settings.PublicHolidays = &guyane
    cases := map[string]struct {
        text     string
        problems int
        zone     PublicHolidayZone
    }{
        "absent":  {`{}`, 0, ZoneGuyane},
        "null":    {`{"public_holidays": null}`, 0, ZoneGuyane},
        "known":   {`{"public_holidays": "fr-mayotte"}`, 0, ZoneMayotte},
        "unknown": {`{"public_holidays": "corse"}`, 1, ZoneGuyane},
    }
    for name, c := range cases {
        imported, err := NewImportFile([]byte(`{"format": "timesheet-import", "version": 1, "settings": ` + c.text + `}`))
        if err != nil {
            t.Fatalf("%s: %v", name, err)
        }
        report, err := importer.Run(calendar, imported, NewImportOptions(), day(t, "2026-04-15"), month(t, "2027-04"))
        if (err != nil) || (len(report.Problems) != c.problems) || (*report.Calendar.Settings.PublicHolidays != c.zone) {
            t.Errorf("%s: %v %+v %s", name, err, report.Problems, *report.Calendar.Settings.PublicHolidays)
        }
    }
    imported, _ := NewImportFile([]byte(`{"format": "timesheet-import", "version": 1, "settings": {"public_holidays": "corse"}}`))
    report, _ := importer.Run(calendar, imported, NewImportOptions(), day(t, "2026-04-15"), month(t, "2027-04"))
    if text := report.Problems[0].Text(); text != `public holidays: zone "corse" unknown` {
        t.Errorf("problem text %q", text)
    }
}

func TestImportedNullCounterKeepsTheSettings(t *testing.T) {
    importer := NewImporter(authorConfiguration(), NewManualClock(time.Date(2026, time.April, 15, 10, 0, 0, 0, time.UTC)), NewIdentifierGenerator(rand.Reader), time.UTC)
    calendar := authorCalendar(t, "2026-01")
    imported, err := NewImportFile([]byte(`{"format": "timesheet-import", "version": 1, "settings": {"counters": {"TPC": null, "NOPE": null, "CP": {"enabled": true}}}}`))
    if err != nil {
        t.Fatal(err)
    }
    report, err := importer.Run(calendar, imported, NewImportOptions(), day(t, "2026-04-15"), month(t, "2027-04"))
    if (err != nil) || (len(report.Problems) != 0) || (report.Counters != 1) || (report.Calendar.Settings.Counters["TPC"].Amount == nil) || !report.Calendar.Settings.Counters["TPC"].Amount.Equal(*calendar.Settings.Counters["TPC"].Amount) {
        t.Fatalf("a null counter must keep the stored settings: %v %+v %+v", err, report.Problems, report.Calendar.Settings.Counters["TPC"])
    }
}

func TestImportedDaysMoveTheTrackingStartBeforeTheAdjustments(t *testing.T) {
    importer := NewImporter(authorConfiguration(), NewManualClock(time.Date(2026, time.April, 15, 10, 0, 0, 0, time.UTC)), NewIdentifierGenerator(rand.Reader), time.UTC)
    calendar := authorCalendar(t, "2025-01")
    calendar.Adjustments = append(calendar.Adjustments, Adjustment{ID: "a1", Date: day(t, "2025-02-10"), Counter: "CP", Amount: q(t, "1"), Reason: "Report"})
    cases := map[string]struct {
        problem string
        warning string
        start   string
    }{
        "2025-01-01": {"", "", "2025-01-01"},
        "2025-01-02": {"", "tracking start moved from 2025-01-01 to 2025-01-02", "2025-01-02"},
        "2025-02-03": {"", "tracking start moved from 2025-01-01 to 2025-02-03", "2025-02-03"},
        "2025-03-02": {"tracking start: adjustments are dated before 2025-03-01", "", "2025-03-01"},
    }
    for first, expected := range cases {
        imported, err := NewImportFile([]byte(`{"format": "timesheet-import", "version": 1, "settings": {"start": "2025-03-01"}, "days": {"` + first + `": {"am": "ABS", "pm": "ABS"}}}`))
        if err != nil {
            t.Fatal(err)
        }
        report, err := importer.Run(calendar, imported, NewImportOptions(), day(t, "2026-04-15"), month(t, "2027-04"))
        if err != nil {
            t.Fatal(err)
        }
        problem, warning := "", ""
        if len(report.Problems) > 0 {
            problem = report.Problems[0].Text()
        }
        if len(report.Warnings) > 0 {
            warning = report.Warnings[0].Text()
        }
        if (len(report.Problems) > 1) || (problem != expected.problem) || (warning != expected.warning) || (report.Calendar.Settings.Start.String() != expected.start) {
            t.Errorf("%s: problems %+v, warnings %+v, start %s", first, report.Problems, report.Warnings, report.Calendar.Settings.Start)
        }
    }
}

func TestImportedAdjustmentWithoutDateIsRefused(t *testing.T) {
    importer := NewImporter(authorConfiguration(), NewManualClock(time.Date(2026, time.April, 15, 10, 0, 0, 0, time.UTC)), NewIdentifierGenerator(rand.Reader), time.UTC)
    for _, adjustment := range []string{`{"counter": "CP", "amount": "1", "reason": "x"}`, `{"date": null, "counter": "CP", "amount": "1", "reason": "x"}`} {
        imported, err := NewImportFile([]byte(`{"format": "timesheet-import", "version": 1, "days": {"2026-01-02": {"am": "TT", "pm": "TT"}}, "adjustments": [` + adjustment + `]}`))
        if err != nil {
            t.Fatal(err)
        }
        report, err := importer.Run(authorCalendar(t, "2026-01"), imported, NewImportOptions(), day(t, "2026-04-15"), month(t, "2027-04"))
        if (err != nil) || (len(report.Problems) != 1) || (report.Adjustments != 1) || (report.Problems[0].Text() != "adjustment without a date") || (report.Problems[0].Label() != "Ajustement sans date") {
            t.Errorf("%s: %v %+v", adjustment, err, report.Problems)
        }
    }
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
