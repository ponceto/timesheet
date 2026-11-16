/*
 * database_test.go - Copyright (c) 2023-2026 - Olivier Poncet
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

package store

import (
    "bytes"
    "crypto/rand"
    "errors"
    "log/slog"
    "os"
    "path/filepath"
    "strings"
    "testing"
    "time"

    "timesheet/internal/account"
    "timesheet/internal/activity"
)

// ---------------------------------------------------------------------------
// Database tests
// ---------------------------------------------------------------------------

func newTestDatabase(t *testing.T) (*Database, string) {
    t.Helper()
    path := filepath.Join(t.TempDir(), "timesheet.json")
    database := NewDatabase(path, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)), activity.NewDefaultDatabaseSettings().MaxBytes, activity.NewDefaultLimitSettings())
    if err := database.Create(activity.NewDefaultConfiguration()); err != nil {
        t.Fatal(err)
    }
    return database, path
}

func newTestAccount(t *testing.T, login string, profile account.Profile) account.Account {
    t.Helper()
    hash, err := account.NewPasswordHash("correct horse battery", account.MinIterations, rand.Reader)
    if err != nil {
        t.Fatal(err)
    }
    return account.Account{Login: account.Login(login), Profile: profile, State: account.StateEnabled, Password: hash, Created: time.Date(2026, time.November, 8, 0, 0, 0, 0, time.UTC)}
}

func TestDatabaseRoundTrip(t *testing.T) {
    database, path := newTestDatabase(t)
    if err := database.Create(activity.NewDefaultConfiguration()); !errors.Is(err, ErrDatabaseExists) {
        t.Fatalf("second create: %v", err)
    }
    start, _ := activity.NewDateFromText("2026-01-01")
    if err := database.CreateAccount(newTestAccount(t, "olivier", account.ProfileAdmin), activity.NewCalendar(start)); err != nil {
        t.Fatal(err)
    }
    _, calendar, found := database.Account("olivier")
    if !found || (calendar.Revision != 1) {
        t.Fatalf("account: %v %d", found, calendar.Revision)
    }
    d, _ := activity.NewDateFromText("2026-11-03")
    calendar.Days[d] = activity.DayEntry{AM: "TT", PM: "TT", Comment: "DEVFEST <TOULOUSE> & co"}
    if err := database.SaveCalendar("olivier", 1, calendar); err != nil {
        t.Fatal(err)
    }
    if err := database.SaveCalendar("olivier", 1, calendar); !errors.Is(err, activity.ErrStaleRevision) {
        t.Fatalf("stale revision: %v", err)
    }
    reloaded := NewDatabase(path, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)), activity.NewDefaultDatabaseSettings().MaxBytes, activity.NewDefaultLimitSettings())
    if err := reloaded.Load(); err != nil {
        t.Fatal(err)
    }
    _, again, _ := reloaded.Account("olivier")
    if (again.Days[d].Comment != "DEVFEST <TOULOUSE> & co") || (again.Revision != 2) {
        t.Fatalf("reloaded: %+v", again.Days[d])
    }
    info, err := os.Stat(path)
    if (err != nil) || (info.Mode().Perm() != 0o600) {
        t.Fatalf("mode: %v", err)
    }
    data, _ := os.ReadFile(path)
    for _, field := range []string{`"version": 1`, `"configuration"`, `"accounts"`, `"days"`, `"2026-11-03"`, `"ledger"`} {
        if !strings.Contains(string(data), field) {
            t.Errorf("missing %s in the file", field)
        }
    }
}

func TestDatabaseRefusesToRewriteAValidatedMonth(t *testing.T) {
    database, _ := newTestDatabase(t)
    start, _ := activity.NewDateFromText("2026-09-01")
    if err := database.CreateAccount(newTestAccount(t, "olivier", account.ProfileUser), activity.NewCalendar(start)); err != nil {
        t.Fatal(err)
    }
    _, calendar, _ := database.Account("olivier")
    september := start.Month()
    for d := september.First(); !d.After(september.Last()); d = d.AddDays(1) {
        code := "TT"
        if d.IsWeekend() {
            code = "WE"
        }
        calendar.Days[d] = activity.DayEntry{AM: code, PM: code}
    }
    today, _ := activity.NewDateFromText("2026-11-08")
    closing := activity.NewMonthClosing(activity.NewBookkeeper(database.Configuration(), calendar), activity.NewIdentifierGenerator(rand.Reader))
    validated, err := closing.Validate(september, time.Date(2026, time.November, 8, 10, 0, 0, 0, time.UTC), today)
    if err != nil {
        t.Fatal(err)
    }
    if err := database.SaveCalendar("olivier", calendar.Revision, validated); err != nil {
        t.Fatal(err)
    }
    _, stored, _ := database.Account("olivier")
    rewritten := stored.Clone()
    rewritten.Days[september.First().AddDays(14)] = activity.DayEntry{AM: "CP", PM: "CP"}
    if err := database.SaveCalendar("olivier", stored.Revision, rewritten); !errors.Is(err, activity.ErrHistoryRewritten) {
        t.Fatalf("rewrite of a validated month: %v", err)
    }
    if err := database.CheckCalendar("olivier", stored.Revision, rewritten); !errors.Is(err, activity.ErrHistoryRewritten) {
        t.Fatalf("check of a rewrite of a validated month: %v", err)
    }
    credited := stored.Clone()
    credited.Ledger = append(credited.Ledger, activity.LedgerEntry{ID: "forged", Month: september, Date: september.Last(), Counter: "CP", Kind: activity.KindCredit, Amount: activity.NewQuantity(10, 1), Label: "test"})
    if err := database.SaveCalendar("olivier", stored.Revision, credited); !errors.Is(err, activity.ErrHistoryRewritten) {
        t.Fatalf("entry added to a validated month: %v", err)
    }
    november := today.Month().First()
    if err := database.EraseCalendar("olivier", stored.Revision, november); err != nil {
        t.Fatal(err)
    }
    if _, erased, _ := database.Account("olivier"); erased.HasHistory() || (len(erased.Days) != 0) || (len(erased.Ledger) != 0) || (erased.Settings.Start != november) || (erased.Revision != (stored.Revision + 1)) {
        t.Fatalf("erased: %+v", erased)
    }
}

func TestDatabaseWritesUnsetFieldsAsNull(t *testing.T) {
    database, path := newTestDatabase(t)
    start, _ := activity.NewDateFromText("2026-01-01")
    if err := database.CreateAccount(newTestAccount(t, "olivier", account.ProfileAdmin), activity.NewCalendar(start)); err != nil {
        t.Fatal(err)
    }
    _, calendar, _ := database.Account("olivier")
    d, _ := activity.NewDateFromText("2026-11-03")
    calendar.Days[d] = activity.DayEntry{AM: "TT"}
    if err := database.SaveCalendar("olivier", 1, calendar); err != nil {
        t.Fatal(err)
    }
    data, _ := os.ReadFile(path)
    for _, field := range []string{`"pm": null`, `"comment": null`, `"name": null`, `"last_login": null`, `"consumes": null`, `"feeds": null`} {
        if !strings.Contains(string(data), field) {
            t.Errorf("missing %s in the file", field)
        }
    }
    reloaded := NewDatabase(path, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)), activity.NewDefaultDatabaseSettings().MaxBytes, activity.NewDefaultLimitSettings())
    if err := reloaded.Load(); err != nil {
        t.Fatal(err)
    }
    if _, again, _ := reloaded.Account("olivier"); again.Days[d] != (activity.DayEntry{AM: "TT"}) {
        t.Fatalf("reloaded: %+v", again.Days[d])
    }
}

func TestDatabaseKeepsColoursOfKnownCodesOnly(t *testing.T) {
    database, _ := newTestDatabase(t)
    start, _ := activity.NewDateFromText("2026-01-01")
    if err := database.CreateAccount(newTestAccount(t, "olivier", account.ProfileAdmin), activity.NewCalendar(start)); err != nil {
        t.Fatal(err)
    }
    _, calendar, _ := database.Account("olivier")
    calendar.Settings.Colors = calendar.Settings.Colors.WithCode("NOPE", "#c8b6e2")
    if err := database.SaveCalendar("olivier", calendar.Revision, calendar); err == nil {
        t.Fatal("a colour for an unknown code must be refused")
    }
    calendar.Settings.Colors = activity.ColorSet{}.WithCode("ABS", "#c8b6e2")
    if err := database.SaveCalendar("olivier", calendar.Revision, calendar); err != nil {
        t.Fatal(err)
    }
    config := database.Configuration().Clone()
    config.Activities = config.Activities[:len(config.Activities) - 1]
    if err := database.SaveConfiguration(config.Revision, config); err != nil {
        t.Fatal(err)
    }
    if _, after, _ := database.Account("olivier"); len(after.Settings.Colors) != 0 {
        t.Fatalf("colour of a deleted code kept: %v", after.Settings.Colors)
    }
}

func TestDatabaseKeepsTheEffectOfAUsedCode(t *testing.T) {
    database, _ := newTestDatabase(t)
    start, _ := activity.NewDateFromText("2026-01-01")
    if err := database.CreateAccount(newTestAccount(t, "olivier", account.ProfileUser), activity.NewCalendar(start)); err != nil {
        t.Fatal(err)
    }
    _, calendar, _ := database.Account("olivier")
    d, _ := activity.NewDateFromText("2026-11-03")
    calendar.Days[d] = activity.DayEntry{AM: "CP"}
    if err := database.SaveCalendar("olivier", calendar.Revision, calendar); err != nil {
        t.Fatal(err)
    }
    config := database.Configuration().Clone()
    for i, code := range config.Activities {
        if code.ID == "CP" {
            config.Activities[i].Consumes = "RTT"
        }
    }
    if err := database.SaveConfiguration(config.Revision, config); !errors.Is(err, activity.ErrConfigurationInUse) {
        t.Fatalf("effect of a used code changed: %v", err)
    }
    _, calendar, _ = database.Account("olivier")
    delete(calendar.Days, d)
    if err := database.SaveCalendar("olivier", calendar.Revision, calendar); err != nil {
        t.Fatal(err)
    }
    if err := database.SaveConfiguration(config.Revision, config); err != nil {
        t.Fatalf("effect of an unused code changed: %v", err)
    }
}

func TestDatabaseProtectsTheLastAdmin(t *testing.T) {
    database, _ := newTestDatabase(t)
    start, _ := activity.NewDateFromText("2026-01-01")
    database.CreateAccount(newTestAccount(t, "root", account.ProfileAdmin), activity.NewCalendar(start))
    database.CreateAccount(newTestAccount(t, "demo", account.ProfileUser), activity.NewCalendar(start))
    if err := database.CreateAccount(newTestAccount(t, "demo", account.ProfileUser), activity.NewCalendar(start)); !errors.Is(err, account.ErrDuplicateLogin) {
        t.Fatalf("duplicate: %v", err)
    }
    if err := database.SetProfile("root", account.ProfileUser); !errors.Is(err, account.ErrLastAdmin) {
        t.Fatalf("demote: %v", err)
    }
    if err := database.DeleteAccount("root"); !errors.Is(err, account.ErrLastAdmin) {
        t.Fatalf("delete: %v", err)
    }
    if err := database.SetState("root", account.StateDisabled); !errors.Is(err, account.ErrLastAdmin) {
        t.Fatalf("disable: %v", err)
    }
    if err := database.SetProfile("demo", account.ProfileAdmin); err != nil {
        t.Fatal(err)
    }
    if err := database.SetState("demo", account.StateDisabled); err != nil {
        t.Fatal(err)
    }
    if err := database.DeleteAccount("root"); !errors.Is(err, account.ErrLastAdmin) {
        t.Fatalf("delete the last enabled administrator: %v", err)
    }
    if err := database.SetState("demo", account.StateEnabled); err != nil {
        t.Fatal(err)
    }
    if err := database.DeleteAccount("root"); err != nil {
        t.Fatal(err)
    }
    if database.AdminCount() != 1 {
        t.Fatal("admin count")
    }
}

func TestDatabaseRefusesInvalidFiles(t *testing.T) {
    database, path := newTestDatabase(t)
    data, _ := os.ReadFile(path)
    cases := map[string]error{
        strings.Replace(string(data), `"version": 1`, `"version": 9`, 1):               ErrUnsupportedVersion,
        strings.Replace(string(data), `"accounts": []`, `"accounts": [], "extra": 1`, 1): ErrMalformedDatabase,
        strings.Replace(string(data), `"#33d17a"`, `"green"`, 1):                       ErrMalformedDatabase,
        strings.Replace(string(data), `"fr-metropole"`, `"corse"`, 1):                     ErrMalformedDatabase,
    }
    for content, want := range cases {
        os.WriteFile(path, []byte(content), 0o600)
        if err := NewDatabase(path, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)), activity.NewDefaultDatabaseSettings().MaxBytes, activity.NewDefaultLimitSettings()).Load(); !errors.Is(err, want) {
            t.Errorf("got %v, want %v", err, want)
        }
    }
    missing := NewDatabase(filepath.Join(t.TempDir(), "none.json"), slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)), activity.NewDefaultDatabaseSettings().MaxBytes, activity.NewDefaultLimitSettings())
    if err := missing.Load(); !errors.Is(err, ErrDatabaseMissing) {
        t.Fatalf("missing: %v", err)
    }
    if err := database.SaveConfiguration(0, activity.NewDefaultConfiguration()); !errors.Is(err, activity.ErrStaleRevision) {
        t.Fatalf("stale configuration: %v", err)
    }
}

func TestDatabaseRefusesUnreadableData(t *testing.T) {
    database, path := newTestDatabase(t)
    start, _ := activity.NewDateFromText("2026-01-01")
    database.CreateAccount(newTestAccount(t, "bob", account.ProfileAdmin), activity.NewCalendar(start))
    orphan := filepath.Join(filepath.Dir(path), filepath.Base(path) + ".4242.tmp")
    os.WriteFile(orphan, []byte("x"), 0o600)
    if err := NewDatabase(path, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)), activity.NewDefaultDatabaseSettings().MaxBytes, activity.NewDefaultLimitSettings()).Load(); err != nil {
        t.Fatal(err)
    }
    if _, err := os.Stat(orphan); err != nil {
        t.Fatal("a read without lock must not remove the temporary files of the writer")
    }
    data, _ := os.ReadFile(path)
    for name, state := range map[string]string{"missing state": ``, "unknown state": `"state": "removed",`} {
        os.WriteFile(path, []byte(strings.Replace(string(data), `"state": "enabled",`, state, 1)), 0o600)
        if err := NewDatabase(path, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)), activity.NewDefaultDatabaseSettings().MaxBytes, activity.NewDefaultLimitSettings()).Load(); !errors.Is(err, ErrMalformedDatabase) || ((state == ``) && !strings.Contains(err.Error(), "accounts[0]: invalid profile, state")) {
            t.Errorf("%s: %v", name, err)
        }
    }
    broken := strings.Replace(string(data), `"adjustments": []`, `"adjustments": [{"id": "x1", "counter": "CP", "amount": "1", "reason": "r"}]`, 1)
    os.WriteFile(path, []byte(broken), 0o600)
    if err := NewDatabase(path, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)), activity.NewDefaultDatabaseSettings().MaxBytes, activity.NewDefaultLimitSettings()).Load(); !errors.Is(err, ErrMalformedDatabase) {
        t.Fatalf("adjustment without date: %v", err)
    }
    first := start.String()
    for name, adjustments := range map[string]string{
        "duplicate identifier": `[{"id": "x1", "date": "` + first + `", "counter": "CP", "amount": "1", "reason": "r"}, {"id": "x1", "date": "` + first + `", "counter": "CP", "amount": "2", "reason": "r"}]`,
        "zero amount": `[{"id": "x1", "date": "` + first + `", "counter": "CP", "amount": "0", "reason": "r"}]`,
        "empty reason": `[{"id": "x1", "date": "` + first + `", "counter": "CP", "amount": "1", "reason": ""}]`,
    } {
        os.WriteFile(path, []byte(strings.Replace(string(data), `"adjustments": []`, `"adjustments": ` + adjustments, 1)), 0o600)
        if err := NewDatabase(path, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)), activity.NewDefaultDatabaseSettings().MaxBytes, activity.NewDefaultLimitSettings()).Load(); !errors.Is(err, ErrMalformedDatabase) {
            t.Errorf("%s: %v", name, err)
        }
    }
    earlier := strings.Replace(string(data), `"adjustments": []`, `"adjustments": [{"id": "x1", "date": "2000-01-03", "counter": "CP", "amount": "1", "reason": "r"}]`, 1)
    if !strings.Contains(earlier, `"months": {}`) {
        t.Fatal("no empty months in the test database")
    }
    earlier = strings.Replace(earlier, `"months": {}`, `"months": {"2000-01": {"status": "open"}}`, 1)
    os.WriteFile(path, []byte(earlier), 0o600)
    if err := NewDatabase(path, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)), activity.NewDefaultDatabaseSettings().MaxBytes, activity.NewDefaultLimitSettings()).Load(); err != nil {
        t.Errorf("a month and an adjustment left before a later start must still load: %v", err)
    }
    os.WriteFile(path, []byte(strings.Replace(string(data), `"login": "bob",`, `"login": "bob", "name": "Bob\u000aMorane",`, 1)), 0o600)
    if err := NewDatabase(path, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)), activity.NewDefaultDatabaseSettings().MaxBytes, activity.NewDefaultLimitSettings()).Load(); !errors.Is(err, ErrMalformedDatabase) {
        t.Errorf("name on two lines: %v", err)
    }
    os.WriteFile(path, data, 0o600)
    _, calendar, _ := database.Account("bob")
    d, _ := activity.NewDateFromText("2026-01-05")
    calendar.Days[d] = activity.DayEntry{Comment: strings.Repeat("x", 500)}
    for i := 0; i < 2500; i++ {
        calendar.Days[d.AddDays(i)] = activity.DayEntry{AM: "TT", Comment: strings.Repeat("y", 499)}
    }
    if err := database.SaveCalendar("bob", calendar.Revision, calendar); !errors.Is(err, activity.ErrCalendarTooLarge) {
        t.Fatalf("oversized account: %v", err)
    }
}

func TestDatabaseReadsAndWritesShareNothing(t *testing.T) {
    database, _ := newTestDatabase(t)
    start, _ := activity.NewDateFromText("2026-01-01")
    calendar := activity.NewCalendar(start)
    enabled := true
    calendar.Settings.Counters["CP"] = activity.CounterSettings{Enabled: &enabled}
    if err := database.CreateAccount(newTestAccount(t, "bob", account.ProfileAdmin), calendar); err != nil {
        t.Fatal(err)
    }
    *calendar.Settings.Counters["CP"].Enabled = false
    _, read, _ := database.Account("bob")
    *read.Settings.Counters["CP"].Enabled = false
    read.Days[start] = activity.DayEntry{AM: "TT"}
    if _, again, _ := database.Account("bob"); !*again.Settings.Counters["CP"].Enabled || (len(again.Days) != 0) {
        t.Fatal("a caller changed the stored calendar through its copy")
    }
    config := database.Configuration()
    config.Activities[0].Label = "changed"
    if database.Configuration().Activities[0].Label == "changed" {
        t.Fatal("a caller changed the stored configuration through its copy")
    }
    if err := database.SaveConfiguration(config.Revision, database.Configuration()); err != nil {
        t.Fatal(err)
    }
    if data, _ := os.ReadFile(database.Path()); !strings.Contains(string(data), `"colors": null`) || strings.Contains(string(data), `"colors": {}`) {
        t.Fatal("an account without own colours must keep null colours after a configuration change")
    }
    if err := database.CreateAccount(account.Account{Login: "eve", Profile: account.ProfileUser, State: account.StateEnabled, Password: newTestAccount(t, "eve", account.ProfileUser).Password, Name: "two\nlines"}, activity.NewCalendar(start)); !errors.Is(err, account.ErrInvalidName) {
        t.Fatalf("account with an invalid name: %v", err)
    }
}

func TestDatabaseRefusesInconsistentMonthsAndLedgers(t *testing.T) {
    database, _ := newTestDatabase(t)
    start, _ := activity.NewDateFromText("2026-01-01")
    database.CreateAccount(newTestAccount(t, "bob", account.ProfileAdmin), activity.NewCalendar(start))
    january, february := start.Month(), start.Month().Next()
    validated := activity.MonthRecord{Status: activity.MonthValidated, ValidatedAt: time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC), Indicators: &activity.MonthlyIndicators{}}
    entry := func(id string, m activity.Month, kind activity.EntryKind, amount int64, ref string) activity.LedgerEntry {
        return activity.LedgerEntry{ID: id, Month: m, Date: m.First(), Counter: "CP", Kind: kind, Amount: activity.NewQuantity(amount, 1), Label: "test", Ref: ref}
    }
    build := func(months map[activity.Month]activity.MonthRecord, ledger ...activity.LedgerEntry) activity.Calendar {
        calendar := activity.NewCalendar(start)
        calendar.Months = months
        calendar.Ledger = ledger
        return calendar
    }
    closed := build(map[activity.Month]activity.MonthRecord{january: validated, february: validated}, entry("a", january, activity.KindCredit, 2, ""), entry("b", february, activity.KindCredit, 3, ""))
    if err := database.SaveCalendar("bob", 1, closed); err != nil {
        t.Fatalf("validated months refused: %v", err)
    }
    sound := build(map[activity.Month]activity.MonthRecord{january: validated}, entry("a", january, activity.KindCredit, 2, ""), entry("b", february, activity.KindCredit, 3, ""), entry("c", february, activity.KindReversal, -3, "b"))
    if err := database.CheckCalendar("bob", 2, sound); err != nil {
        t.Fatalf("sound calendar refused: %v", err)
    }
    cases := map[string]activity.Calendar{
        "hole in the validated months":     build(map[activity.Month]activity.MonthRecord{february: validated}),
        "validated month without snapshot": build(map[activity.Month]activity.MonthRecord{january: {Status: activity.MonthValidated}}),
        "active entry in an open month":    build(map[activity.Month]activity.MonthRecord{january: validated}, entry("a", february, activity.KindCredit, 2, "")),
        "entry reversed twice":             build(map[activity.Month]activity.MonthRecord{january: validated}, entry("a", january, activity.KindCredit, 2, ""), entry("b", january, activity.KindReversal, -2, "a"), entry("c", january, activity.KindReversal, -2, "a")),
        "reversal of another amount":       build(map[activity.Month]activity.MonthRecord{january: validated}, entry("a", january, activity.KindCredit, 2, ""), entry("b", january, activity.KindReversal, 77, "a")),
        "reversal of a reversal":           build(map[activity.Month]activity.MonthRecord{january: validated}, entry("a", january, activity.KindCredit, 2, ""), entry("b", january, activity.KindReversal, -2, "a"), entry("c", january, activity.KindReversal, 2, "b")),
    }
    corse := activity.PublicHolidayZone("corse")
    unknownZone := build(map[activity.Month]activity.MonthRecord{})
    unknownZone.Settings.PublicHolidays = &corse
    cases["unknown zone of public holidays"] = unknownZone
    for name, calendar := range cases {
        if err := database.CheckCalendar("bob", 2, calendar); !errors.Is(err, ErrMalformedDatabase) {
            t.Errorf("%s: %v", name, err)
        }
    }
}

func TestChangePasswordNeedsTheHashThatWasVerified(t *testing.T) {
    database, _ := newTestDatabase(t)
    start, _ := activity.NewDateFromText("2026-01-01")
    bob := newTestAccount(t, "bob", account.ProfileAdmin)
    database.CreateAccount(bob, activity.NewCalendar(start))
    reset := newTestAccount(t, "bob", account.ProfileAdmin).Password
    chosen := newTestAccount(t, "bob", account.ProfileAdmin).Password
    if err := database.SetPassword("bob", reset, true); err != nil {
        t.Fatal(err)
    }
    if err := database.ChangePassword("bob", bob.Password, chosen); !errors.Is(err, activity.ErrStaleRevision) {
        t.Fatalf("change over a reset: %v", err)
    }
    if found, _, _ := database.Account("bob"); (found.Password != reset) || !found.MustChangePassword {
        t.Fatal("the reset was overwritten")
    }
    if err := database.ChangePassword("bob", reset, chosen); err != nil {
        t.Fatal(err)
    }
    if found, _, _ := database.Account("bob"); (found.Password != chosen) || found.MustChangePassword {
        t.Fatal("the password was not changed")
    }
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
