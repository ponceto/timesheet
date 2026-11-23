/*
 * database.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "log/slog"
    "math"
    "slices"
    "strings"
    "sync"
    "time"

    "timesheet/internal/account"
    "timesheet/internal/activity"
    "timesheet/internal/assistant"
)

// ---------------------------------------------------------------------------
// Database declaration
// ---------------------------------------------------------------------------

type Database struct {
    mu     sync.RWMutex
    file   *SecureFile
    keys   *KeyRing
    limits activity.LimitSettings
    record DatabaseRecord
}

type loginSet map[account.Login]bool

const (
    maxQuantity    = 366
    maxLabelLength = 1000
)

var (
    ErrDatabaseExists     = errors.New("store: the database already exists")
    ErrDatabaseMissing    = errors.New("store: the database does not exist, run init first")
    ErrMalformedDatabase  = errors.New("store: malformed database")
    ErrUnsupportedVersion = errors.New("store: the database was written by a newer version")
)

// ---------------------------------------------------------------------------
// Database implementation
// ---------------------------------------------------------------------------

func NewDatabase(path string, logger *slog.Logger, maxBytes int64, limits activity.LimitSettings) *Database {
    return &Database{file: NewSecureFile(path, logger, maxBytes), keys: NewKeyRing(path + activity.KeySuffix, logger), limits: limits}
}

func (d *Database) Path() string {
    return d.file.Path()
}

func (d *Database) Lock() error {
    if err := d.file.Lock(); err != nil {
        return err
    }
    if err := d.file.Prepare(); err != nil {
        d.file.Unlock()
        return err
    }
    return nil
}

func (d *Database) Unlock() {
    d.file.Unlock()
}

func (d *Database) Create(config activity.Configuration) error {
    if err := config.Validate(); err != nil {
        return err
    }
    if err := d.file.Prepare(); err != nil {
        return err
    }
    if _, found, err := d.file.Read(); err != nil {
        return err
    } else if found {
        return fmt.Errorf("%w: %s", ErrDatabaseExists, d.Path())
    }
    config.Revision = 1
    record := DatabaseRecord{Version: DatabaseVersion, Revision: 1, Configuration: config, Accounts: []AccountRecord{}}
    d.mu.Lock()
    defer d.mu.Unlock()
    if err := d.write(record); err != nil {
        return err
    }
    d.record = record
    return nil
}

func (d *Database) Load() error {
    data, found, err := d.file.Read()
    if err != nil {
        return err
    }
    if !found {
        return fmt.Errorf("%w: %s", ErrDatabaseMissing, d.Path())
    }
    probe := struct {
        Version int `json:"version"`
    }{}
    if err := json.Unmarshal(data, &probe); err != nil {
        return fmt.Errorf("%w: %v", ErrMalformedDatabase, err)
    }
    if probe.Version > DatabaseVersion {
        return ErrUnsupportedVersion
    }
    if probe.Version != DatabaseVersion {
        return fmt.Errorf("%w: unknown version %d", ErrMalformedDatabase, probe.Version)
    }
    record := DatabaseRecord{}
    decoder := json.NewDecoder(bytes.NewReader(data))
    decoder.DisallowUnknownFields()
    if err := decoder.Decode(&record); err != nil {
        return fmt.Errorf("%w: %v", ErrMalformedDatabase, err)
    }
    if _, err := decoder.Token(); err != io.EOF {
        return fmt.Errorf("%w: data after the database", ErrMalformedDatabase)
    }
    if err := d.validate(record); err != nil {
        return err
    }
    if _, err := d.encode(record); err != nil {
        return err
    }
    for i, item := range record.Accounts {
        record.Accounts[i].Calendar = item.Calendar.Clone()
        if item.Assistant != nil {
            state := item.Assistant.Clone()
            record.Accounts[i].Assistant = &state
        }
    }
    if record.Accounts == nil {
        record.Accounts = []AccountRecord{}
    }
    d.mu.Lock()
    defer d.mu.Unlock()
    d.record = record
    return nil
}

func (d *Database) validate(record DatabaseRecord) error {
    if err := record.Configuration.Validate(); err != nil {
        return fmt.Errorf("%w: configuration: %v", ErrMalformedDatabase, err)
    }
    logins := d.logins(record)
    seen := map[account.Login]bool{}
    for i, item := range record.Accounts {
        path := fmt.Sprintf("accounts[%d]", i)
        if !item.Login.Valid() || seen[item.Login] {
            return fmt.Errorf("%w: %s.login: invalid or duplicate login", ErrMalformedDatabase, path)
        }
        seen[item.Login] = true
        if !item.Profile.Persistent() || !item.State.Valid() || !item.Password.Valid() || !d.validName(item.Name) {
            return fmt.Errorf("%w: %s: invalid profile, state, password hash or name", ErrMalformedDatabase, path)
        }
        if err := d.validateCalendar(record.Configuration, item.Calendar); err != nil {
            return fmt.Errorf("%w: %s.%v", ErrMalformedDatabase, path, err)
        }
        if item.Assistant != nil {
            if err := item.Assistant.Validate(record.Configuration, item.Login, logins.Has); err != nil {
                return fmt.Errorf("%w: %s.assistant: %v", ErrMalformedDatabase, path, err)
            }
        }
    }
    return nil
}

func (d *Database) logins(record DatabaseRecord) loginSet {
    logins := loginSet{}
    for _, item := range record.Accounts {
        logins[item.Login] = true
    }
    return logins
}

func (s loginSet) Has(login account.Login) bool {
    return s[login]
}

func (d *Database) validateCalendar(config activity.Configuration, calendar activity.Calendar) error {
    if !calendar.Settings.Start.Month().Valid() {
        return errors.New("settings.start: invalid date")
    }
    if (calendar.Settings.PublicHolidays != nil) && !calendar.Settings.PublicHolidays.Valid() {
        return errors.New("settings.public_holidays: unknown zone")
    }
    text := activity.NewTextPolicy()
    for date, entry := range calendar.Days {
        if !text.Accepts(entry.Comment, activity.MaxCommentLength) {
            return fmt.Errorf("days.%s.comment: invalid comment", date)
        }
        for _, id := range []string{entry.AM, entry.PM} {
            if _, known := config.Activity(id); (id != "") && !known {
                return fmt.Errorf("days.%s: unknown activity code %q", date, id)
            }
        }
    }
    limit := activity.NewQuantity(maxQuantity, 1)
    for id, settings := range calendar.Settings.Counters {
        if _, known := config.Counter(id); !known {
            return fmt.Errorf("settings.counters.%q: unknown counter", id)
        }
        if ((settings.Amount != nil) && ((settings.Amount.Sign() < 0) || (settings.Amount.Cmp(limit) > 0))) || (settings.Initial.Abs().Cmp(limit) > 0) {
            return fmt.Errorf("settings.counters.%q: amount or initial balance out of bounds", id)
        }
    }
    if t, invalid := calendar.Settings.Colors.Invalid(); invalid {
        return fmt.Errorf("settings.colors.%q: invalid colour", t)
    }
    for id := range calendar.Settings.Colors {
        if _, known := config.Activity(id); !known {
            return fmt.Errorf("settings.colors.%q: unknown code", id)
        }
    }
    adjustments := map[string]bool{}
    for i, adjustment := range calendar.Adjustments {
        if _, known := config.Counter(adjustment.Counter); !known || (adjustment.ID == "") || adjustments[adjustment.ID] || adjustment.Date.IsZero() || adjustment.Amount.IsZero() || (adjustment.Amount.Abs().Cmp(limit) > 0) || (strings.TrimSpace(adjustment.Reason) == "") || !text.Accepts(adjustment.Reason, activity.MaxReasonLength) {
            return fmt.Errorf("adjustments[%d]: invalid adjustment", i)
        }
        adjustments[adjustment.ID] = true
    }
    identifiers := map[string]bool{}
    for i, entry := range calendar.Ledger {
        if _, known := config.Counter(entry.Counter); !known || !entry.Kind.Valid() || (entry.ID == "") || identifiers[entry.ID] || entry.Date.IsZero() || (entry.Date.Month() != entry.Month) || !text.Accepts(entry.Label, maxLabelLength) || ((entry.Kind == activity.KindReversal) && !identifiers[entry.Ref]) {
            return fmt.Errorf("ledger[%d]: invalid entry", i)
        }
        identifiers[entry.ID] = true
    }
    validated := 0
    for m, record := range calendar.Months {
        if (record.Status != activity.MonthOpen) && (record.Status != activity.MonthValidated) {
            return fmt.Errorf("months.%s.status: open or validated expected", m)
        }
        if record.Status == activity.MonthValidated {
            validated++
            if record.ValidatedAt.IsZero() || (record.Indicators == nil) {
                return fmt.Errorf("months.%s: validation time and indicators expected", m)
            }
        }
        for id := range record.Rules {
            if _, known := config.Counter(id); !known {
                return fmt.Errorf("months.%s.rules.%q: unknown counter", m, id)
            }
        }
        for _, event := range record.History {
            if ((event.Action != activity.ActionValidated) && (event.Action != activity.ActionReopened)) || !text.Accepts(event.Reason, activity.MaxReasonLength) {
                return fmt.Errorf("months.%s.history: invalid reason", m)
            }
        }
    }
    closed := 0
    for m := calendar.Settings.Start.Month(); calendar.IsValidated(m); m = m.Next() {
        closed++
    }
    if closed != validated {
        return errors.New("months: the validated months must follow each other from the tracking start")
    }
    return d.validateLedger(calendar)
}

func (d *Database) validateLedger(calendar activity.Calendar) error {
    entries := map[string]activity.LedgerEntry{}
    reversed := map[string]bool{}
    for i, entry := range calendar.Ledger {
        entries[entry.ID] = entry
        if entry.Kind != activity.KindReversal {
            continue
        }
        original := entries[entry.Ref]
        if (original.Kind == activity.KindReversal) || reversed[entry.Ref] || (original.Counter != entry.Counter) || (original.Month != entry.Month) || (original.Date != entry.Date) || !original.Amount.Neg().Equal(entry.Amount) {
            return fmt.Errorf("ledger[%d]: a reversal cancels one earlier entry of the same counter, month and date, once", i)
        }
        reversed[entry.Ref] = true
    }
    for i, entry := range calendar.Ledger {
        if (entry.Kind != activity.KindReversal) && !reversed[entry.ID] && !calendar.IsValidated(entry.Month) {
            return fmt.Errorf("ledger[%d]: active entry in a month that is not validated", i)
        }
    }
    return nil
}

func (d *Database) encode(record DatabaseRecord) ([]byte, error) {
    var data bytes.Buffer
    encoder := json.NewEncoder(&data)
    encoder.SetEscapeHTML(false)
    encoder.SetIndent("", "  ")
    if err := encoder.Encode(record); err != nil {
        return nil, fmt.Errorf("%w: %v", ErrMalformedDatabase, err)
    }
    check := DatabaseRecord{}
    decoder := json.NewDecoder(bytes.NewReader(data.Bytes()))
    decoder.DisallowUnknownFields()
    if err := decoder.Decode(&check); err != nil {
        return nil, fmt.Errorf("%w: the data cannot be read back: %v", ErrMalformedDatabase, err)
    }
    return data.Bytes(), nil
}

func (d *Database) size(item AccountRecord) int {
    var data bytes.Buffer
    encoder := json.NewEncoder(&data)
    encoder.SetEscapeHTML(false)
    if encoder.Encode(item) != nil {
        return math.MaxInt
    }
    return data.Len()
}

func (d *Database) write(record DatabaseRecord) error {
    data, err := d.encode(record)
    if err != nil {
        return err
    }
    return d.file.Write(data)
}

func (d *Database) begin() DatabaseRecord {
    next := d.record
    next.Accounts = slices.Clone(d.record.Accounts)
    return next
}

func (d *Database) commit(next DatabaseRecord) error {
    next.Revision++
    if err := d.write(next); err != nil {
        return err
    }
    d.record = next
    return nil
}

func (d *Database) find(record DatabaseRecord, login account.Login) int {
    for i, item := range record.Accounts {
        if item.Login == login {
            return i
        }
    }
    return -1
}

func (d *Database) Configuration() activity.Configuration {
    d.mu.RLock()
    defer d.mu.RUnlock()
    return d.record.Configuration.Clone()
}

func (d *Database) Account(login account.Login) (account.Account, activity.Calendar, bool) {
    d.mu.RLock()
    defer d.mu.RUnlock()
    index := d.find(d.record, login)
    if index < 0 {
        return account.Account{}, activity.Calendar{}, false
    }
    item := d.record.Accounts[index]
    return item.Account, item.Calendar.Clone(), true
}

func (d *Database) Accounts() []account.Account {
    d.mu.RLock()
    defer d.mu.RUnlock()
    accounts := make([]account.Account, 0, len(d.record.Accounts))
    for _, item := range d.record.Accounts {
        accounts = append(accounts, item.Account)
    }
    return accounts
}

func (d *Database) AdminCount() int {
    d.mu.RLock()
    defer d.mu.RUnlock()
    return d.admins(d.record)
}

func (d *Database) admins(record DatabaseRecord) int {
    count := 0
    for _, item := range record.Accounts {
        if (item.Profile == account.ProfileAdmin) && item.State.Enabled() {
            count++
        }
    }
    return count
}

func (d *Database) lastAdmin(record DatabaseRecord, index int) bool {
    item := record.Accounts[index]
    return (item.Profile == account.ProfileAdmin) && item.State.Enabled() && (d.admins(record) <= 1)
}

func (d *Database) SetState(login account.Login, state account.State) error {
    if !state.Valid() {
        return account.ErrInvalidState
    }
    d.mu.Lock()
    defer d.mu.Unlock()
    next := d.begin()
    index := d.find(next, login)
    if index < 0 {
        return account.ErrUnknownAccount
    }
    if !state.Enabled() && d.lastAdmin(next, index) {
        return account.ErrLastAdmin
    }
    next.Accounts[index].State = state
    next.Accounts[index].Calendar.Revision++
    return d.commit(next)
}

func (d *Database) ActivityUsed(id string) bool {
    d.mu.RLock()
    defer d.mu.RUnlock()
    for _, item := range d.record.Accounts {
        if item.Calendar.UsesActivity(id) {
            return true
        }
    }
    return false
}

func (d *Database) CounterUsed(id string) bool {
    d.mu.RLock()
    defer d.mu.RUnlock()
    for _, item := range d.record.Accounts {
        if item.Calendar.UsesCounter(id) {
            return true
        }
    }
    return false
}

func (d *Database) CreateAccount(created account.Account, calendar activity.Calendar) error {
    if !created.Login.Valid() || !created.Profile.Persistent() || !created.State.Valid() || !created.Password.Valid() {
        return account.ErrInvalidLogin
    }
    if !d.validName(created.Name) {
        return account.ErrInvalidName
    }
    d.mu.Lock()
    defer d.mu.Unlock()
    next := d.begin()
    if err := d.validateCalendar(next.Configuration, calendar); err != nil {
        return fmt.Errorf("%w: %v", ErrMalformedDatabase, err)
    }
    if d.find(next, created.Login) >= 0 {
        return account.ErrDuplicateLogin
    }
    if int64(len(next.Accounts)) >= d.limits.Accounts {
        return account.ErrTooManyAccounts
    }
    calendar.Revision = 1
    next.Accounts = append(next.Accounts, AccountRecord{Account: created, Calendar: calendar.Clone()})
    return d.commit(next)
}

func (d *Database) DeleteAccount(login account.Login) error {
    d.mu.Lock()
    defer d.mu.Unlock()
    next := d.begin()
    index := d.find(next, login)
    if index < 0 {
        return account.ErrUnknownAccount
    }
    if d.lastAdmin(next, index) {
        return account.ErrLastAdmin
    }
    next.Accounts = slices.Delete(next.Accounts, index, index + 1)
    for i, item := range next.Accounts {
        if item.Assistant != nil {
            state := item.Assistant.WithoutShare(login)
            next.Accounts[i].Assistant = &state
        }
    }
    return d.commit(next)
}

func (d *Database) SetName(login account.Login, name string) error {
    if !d.validName(name) {
        return account.ErrInvalidName
    }
    d.mu.Lock()
    defer d.mu.Unlock()
    next := d.begin()
    index := d.find(next, login)
    if index < 0 {
        return account.ErrUnknownAccount
    }
    next.Accounts[index].Name = name
    return d.commit(next)
}

func (d *Database) validName(name string) bool {
    return (strings.TrimSpace(name) == name) && activity.NewTextPolicy().Accepts(name, account.MaxNameLength)
}

func (d *Database) SetProfile(login account.Login, profile account.Profile) error {
    if !profile.Persistent() {
        return account.ErrInvalidProfile
    }
    d.mu.Lock()
    defer d.mu.Unlock()
    next := d.begin()
    index := d.find(next, login)
    if index < 0 {
        return account.ErrUnknownAccount
    }
    if (profile != account.ProfileAdmin) && d.lastAdmin(next, index) {
        return account.ErrLastAdmin
    }
    next.Accounts[index].Profile = profile
    next.Accounts[index].Calendar.Revision++
    return d.commit(next)
}

func (d *Database) SetPassword(login account.Login, hash account.PasswordHash, mustChange bool) error {
    if !hash.Valid() {
        return account.ErrInvalidPasswordHash
    }
    d.mu.Lock()
    defer d.mu.Unlock()
    next := d.begin()
    index := d.find(next, login)
    if index < 0 {
        return account.ErrUnknownAccount
    }
    next.Accounts[index].Password = hash
    next.Accounts[index].MustChangePassword = mustChange
    next.Accounts[index].Calendar.Revision++
    return d.commit(next)
}

func (d *Database) ChangePassword(login account.Login, current account.PasswordHash, hash account.PasswordHash) error {
    if !hash.Valid() {
        return account.ErrInvalidPasswordHash
    }
    d.mu.Lock()
    defer d.mu.Unlock()
    next := d.begin()
    index := d.find(next, login)
    if index < 0 {
        return account.ErrUnknownAccount
    }
    if (next.Accounts[index].Password != current) || !next.Accounts[index].State.Enabled() {
        return activity.ErrStaleRevision
    }
    next.Accounts[index].Password = hash
    next.Accounts[index].MustChangePassword = false
    next.Accounts[index].Calendar.Revision++
    return d.commit(next)
}

func (d *Database) TouchLogin(login account.Login, at time.Time) error {
    d.mu.Lock()
    defer d.mu.Unlock()
    next := d.begin()
    index := d.find(next, login)
    if index < 0 {
        return account.ErrUnknownAccount
    }
    next.Accounts[index].LastLogin = at.UTC().Truncate(time.Second)
    return d.commit(next)
}

func (d *Database) SaveCalendar(login account.Login, revision int64, calendar activity.Calendar) error {
    d.mu.Lock()
    defer d.mu.Unlock()
    next := d.begin()
    if err := d.place(next, login, revision, calendar, true); err != nil {
        return err
    }
    return d.commit(next)
}

func (d *Database) EraseCalendar(login account.Login, revision int64, start activity.Date) error {
    d.mu.Lock()
    defer d.mu.Unlock()
    next := d.begin()
    if err := d.place(next, login, revision, activity.NewCalendar(start), false); err != nil {
        return err
    }
    if index := d.find(next, login); next.Accounts[index].Assistant != nil {
        state := next.Accounts[index].Assistant.Clone()
        state.Conversations, state.Revision = []assistant.Conversation{}, state.Revision + 1
        next.Accounts[index].Assistant = &state
    }
    return d.commit(next)
}

func (d *Database) CheckCalendar(login account.Login, revision int64, calendar activity.Calendar) error {
    d.mu.Lock()
    defer d.mu.Unlock()
    next := d.begin()
    return d.place(next, login, revision, calendar, true)
}

func (d *Database) place(next DatabaseRecord, login account.Login, revision int64, calendar activity.Calendar, keepHistory bool) error {
    index := d.find(next, login)
    if index < 0 {
        return account.ErrUnknownAccount
    }
    if next.Accounts[index].Calendar.Revision != revision {
        return activity.ErrStaleRevision
    }
    if err := d.validateCalendar(next.Configuration, calendar); err != nil {
        return fmt.Errorf("%w: %v", ErrMalformedDatabase, err)
    }
    previous := next.Accounts[index]
    if keepHistory && !calendar.KeepsHistoryOf(previous.Calendar) {
        return activity.ErrHistoryRewritten
    }
    calendar = calendar.Clone()
    calendar.Revision = revision + 1
    next.Accounts[index].Calendar = calendar
    if (int64(len(calendar.Adjustments)) > d.limits.Adjustments) && (len(calendar.Adjustments) > len(previous.Calendar.Adjustments)) {
        return activity.ErrCalendarTooLarge
    }
    if after := d.size(next.Accounts[index]); (int64(after) > d.limits.AccountBytes) && (after > d.size(previous)) {
        return activity.ErrCalendarTooLarge
    }
    return nil
}

func (d *Database) SaveConfiguration(revision int64, config activity.Configuration) error {
    if err := config.Validate(); err != nil {
        return err
    }
    d.mu.Lock()
    defer d.mu.Unlock()
    next := d.begin()
    if next.Configuration.Revision != revision {
        return activity.ErrStaleRevision
    }
    for i, item := range next.Accounts {
        var colors activity.ColorSet
        for id, color := range item.Calendar.Settings.Colors {
            if _, known := config.Activity(id); known {
                colors = colors.WithCode(id, color)
            }
        }
        next.Accounts[i].Calendar.Settings.Colors = colors
        if err := d.validateCalendar(config, next.Accounts[i].Calendar); err != nil {
            return fmt.Errorf("%w: %s.%v", activity.ErrConfigurationInUse, item.Login, err)
        }
        if item.Assistant != nil {
            if err := item.Assistant.Validate(config, item.Login, d.logins(next).Has); err != nil {
                return fmt.Errorf("%w: %s.assistant: %v", activity.ErrConfigurationInUse, item.Login, err)
            }
        }
        for _, code := range config.Activities {
            previous, known := next.Configuration.Activity(code.ID)
            if known && ((previous.Consumes != code.Consumes) || (previous.Feeds != code.Feeds)) && item.Calendar.UsesActivity(code.ID) {
                return fmt.Errorf("%w: %s.%s: the effect of a used code changed", activity.ErrConfigurationInUse, item.Login, code.ID)
            }
        }
    }
    config.Revision = revision + 1
    next.Configuration = config.Clone()
    return d.commit(next)
}

func (d *Database) KeyPath() string {
    return d.keys.Path()
}

func (d *Database) AssistantState(login account.Login) (assistant.State, bool) {
    d.mu.RLock()
    defer d.mu.RUnlock()
    index := d.find(d.record, login)
    if index < 0 {
        return assistant.State{}, false
    }
    if state := d.record.Accounts[index].Assistant; state != nil {
        return state.Clone(), true
    }
    return assistant.State{}, true
}

func (d *Database) SharedSubscriptions(login account.Login) []assistant.SharedSubscription {
    d.mu.RLock()
    defer d.mu.RUnlock()
    index := d.find(d.record, login)
    if index < 0 {
        return nil
    }
    own := assistant.State{}
    if d.record.Accounts[index].Assistant != nil {
        own = *d.record.Accounts[index].Assistant
    }
    shared := []assistant.SharedSubscription{}
    for _, item := range d.record.Accounts {
        if (item.Login == login) || (item.Assistant == nil) || !item.State.Enabled() {
            continue
        }
        for _, subscription := range item.Assistant.Subscriptions {
            if subscription.SharedTo(login) {
                subscription = subscription.Clone()
                subscription.Key, subscription.SharedWith = "", nil
                entry := assistant.SharedSubscription{Owner: item.Login, OwnerName: item.Name, Subscription: subscription}
                entry.Hidden = own.IsHidden(entry.Ref())
                shared = append(shared, entry)
            }
        }
    }
    return shared
}

func (d *Database) SaveAssistantState(login account.Login, revision int64, state assistant.State, keys map[string]string) error {
    d.mu.Lock()
    defer d.mu.Unlock()
    next := d.begin()
    index := d.find(next, login)
    if index < 0 {
        return account.ErrUnknownAccount
    }
    current := assistant.State{}
    if next.Accounts[index].Assistant != nil {
        current = *next.Accounts[index].Assistant
    }
    if current.Revision != revision {
        return activity.ErrStaleRevision
    }
    state = state.Clone()
    for id, plain := range keys {
        position, found := state.Subscription(id)
        if !found || (plain == "") || (len(plain) > assistant.MaxKeyLength) || !activity.NewTextPolicy().Accepts(plain, assistant.MaxKeyLength) {
            return assistant.ErrInvalidKey
        }
        sealed, err := d.keys.Seal(plain, d.secretContext(login, id))
        if err != nil {
            return err
        }
        state.Subscriptions[position].Key = sealed
    }
    if err := state.Validate(next.Configuration, login, d.logins(next).Has); err != nil {
        return err
    }
    state.Revision = revision + 1
    next.Accounts[index].Assistant = &state
    for (d.size(next.Accounts[index]) > int(d.limits.AccountBytes)) && (len(state.Conversations) > 1) {
        state.Conversations = d.withoutOldest(state.Conversations)
        next.Accounts[index].Assistant = &state
    }
    return d.commit(next)
}

func (d *Database) withoutOldest(conversations []assistant.Conversation) []assistant.Conversation {
    oldest := 0
    for i, conversation := range conversations {
        if conversation.Updated.Before(conversations[oldest].Updated) {
            oldest = i
        }
    }
    return slices.Delete(slices.Clone(conversations), oldest, oldest + 1)
}

func (d *Database) SubscriptionKey(login account.Login, ref assistant.SubscriptionRef) (assistant.Subscription, string, error) {
    d.mu.RLock()
    index := d.find(d.record, ref.Owner)
    if (index < 0) || (d.record.Accounts[index].Assistant == nil) || !d.record.Accounts[index].State.Enabled() {
        d.mu.RUnlock()
        return assistant.Subscription{}, "", assistant.ErrUnusable
    }
    state := d.record.Accounts[index].Assistant
    position, found := state.Subscription(ref.ID)
    if !found || ((ref.Owner != login) && !state.Subscriptions[position].SharedTo(login)) {
        d.mu.RUnlock()
        return assistant.Subscription{}, "", assistant.ErrUnusable
    }
    subscription := state.Subscriptions[position].Clone()
    d.mu.RUnlock()
    plain, err := d.keys.Open(subscription.Key, d.secretContext(ref.Owner, ref.ID))
    subscription.Key = ""
    return subscription, plain, err
}

func (d *Database) secretContext(owner account.Login, id string) string {
    return "timesheet/subscription/" + string(owner) + "/" + id
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
