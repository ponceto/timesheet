/*
 * key_ring_test.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "errors"
    "log/slog"
    "os"
    "path/filepath"
    "strings"
    "testing"
    "time"

    "timesheet/internal/account"
    "timesheet/internal/activity"
    "timesheet/internal/assistant"
)

// ---------------------------------------------------------------------------
// KeyRing tests
// ---------------------------------------------------------------------------

func TestKeyRingSealsWithAPrivateKeyFile(t *testing.T) {
    path := filepath.Join(t.TempDir(), "timesheet.json.key")
    ring := NewKeyRing(path, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
    if _, err := ring.Open("AAAA", "ctx"); !errors.Is(err, ErrUnreadableSecret) {
        t.Fatalf("no key file yet: %v", err)
    }
    sealed, err := ring.Seal("sk-secret", "ctx")
    if err != nil {
        t.Fatal(err)
    }
    info, err := os.Stat(path)
    if (err != nil) || (info.Mode().Perm() != 0o600) {
        t.Fatalf("the key file must be private: %v %v", info, err)
    }
    again := NewKeyRing(path, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
    if plain, err := again.Open(sealed, "ctx"); (err != nil) || (plain != "sk-secret") {
        t.Fatalf("got %q %v", plain, err)
    }
    if _, err := again.Open(sealed, "other"); !errors.Is(err, ErrUnreadableSecret) {
        t.Fatalf("a sealed key is bound to its subscription: %v", err)
    }
}

func TestDatabaseKeepsTheAssistantOfEachAccount(t *testing.T) {
    database, path := newTestDatabase(t)
    start, _ := activity.NewDateFromText("2026-01-01")
    for _, login := range []string{"olivier", "demo", "other"} {
        if err := database.CreateAccount(newTestAccount(t, login, account.ProfileAdmin), activity.NewCalendar(start)); err != nil {
            t.Fatal(err)
        }
    }
    now := time.Date(2026, time.November, 8, 10, 0, 0, 0, time.UTC)
    state := assistant.State{Subscriptions: []assistant.Subscription{{ID: "a1", Name: "Claude", Provider: "anthropic", Model: "claude-sonnet-5-5", SharedWith: []account.Login{"demo"}, Created: now}}}
    if err := database.SaveAssistantState("olivier", 0, state, map[string]string{"a1": "sk-ant-secret"}); err != nil {
        t.Fatal(err)
    }
    if err := database.SaveAssistantState("olivier", 0, state, nil); !errors.Is(err, activity.ErrStaleRevision) {
        t.Fatalf("a stale revision must be refused: %v", err)
    }
    data, _ := os.ReadFile(path)
    if strings.Contains(string(data), "sk-ant-secret") || !strings.Contains(string(data), `"assistant"`) {
        t.Fatal("the key must be sealed in the database")
    }
    ref := assistant.SubscriptionRef{Owner: "olivier", ID: "a1"}
    for _, login := range []account.Login{"olivier", "demo"} {
        if _, key, err := database.SubscriptionKey(login, ref); (err != nil) || (key != "sk-ant-secret") {
            t.Fatalf("%s: got %q %v", login, key, err)
        }
    }
    if _, _, err := database.SubscriptionKey("other", ref); !errors.Is(err, assistant.ErrUnusable) {
        t.Fatalf("a subscription that is not shared is not usable: %v", err)
    }
    if shared := database.SharedSubscriptions("demo"); (len(shared) != 1) || (shared[0].Subscription.Key != "") || (shared[0].Owner != "olivier") {
        t.Fatalf("got %+v", shared)
    }
    reloaded := NewDatabase(path, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)), activity.NewDefaultDatabaseSettings().MaxBytes, activity.NewDefaultLimitSettings())
    if err := reloaded.Load(); err != nil {
        t.Fatal(err)
    }
    if err := reloaded.DeleteAccount("demo"); err != nil {
        t.Fatal(err)
    }
    kept, _ := reloaded.AssistantState("olivier")
    if (kept.Revision != 1) || (len(kept.Subscriptions[0].SharedWith) != 0) {
        t.Fatalf("deleting an account removes its shares: %+v", kept)
    }
}

func TestDatabaseRefusesASubscriptionWithoutKeyOrShare(t *testing.T) {
    database, _ := newTestDatabase(t)
    start, _ := activity.NewDateFromText("2026-01-01")
    if err := database.CreateAccount(newTestAccount(t, "olivier", account.ProfileAdmin), activity.NewCalendar(start)); err != nil {
        t.Fatal(err)
    }
    now := time.Date(2026, time.November, 8, 10, 0, 0, 0, time.UTC)
    state := assistant.State{Subscriptions: []assistant.Subscription{{ID: "a1", Name: "Mistral", Provider: "mistral", Model: "mistral-small-latest", Created: now}}}
    if err := database.SaveAssistantState("olivier", 0, state, nil); !errors.Is(err, assistant.ErrInvalidState) {
        t.Fatalf("a subscription needs a key: %v", err)
    }
    state.Subscriptions[0].SharedWith = []account.Login{"ghost"}
    if err := database.SaveAssistantState("olivier", 0, state, map[string]string{"a1": "key"}); !errors.Is(err, assistant.ErrInvalidState) {
        t.Fatalf("a share needs an existing account: %v", err)
    }
}

func TestDatabaseEraseForgetsTheConversations(t *testing.T) {
    database, _ := newTestDatabase(t)
    start, _ := activity.NewDateFromText("2026-01-01")
    if err := database.CreateAccount(newTestAccount(t, "olivier", account.ProfileAdmin), activity.NewCalendar(start)); err != nil {
        t.Fatal(err)
    }
    now := time.Date(2026, time.November, 8, 10, 0, 0, 0, time.UTC)
    state := assistant.State{
        Subscriptions: []assistant.Subscription{{ID: "a1", Name: "Mistral", Provider: "mistral", Model: "mistral-small-latest", Created: now}},
        Conversations: []assistant.Conversation{{ID: "c1", Title: "Soldes", Subscription: assistant.SubscriptionRef{Owner: "olivier", ID: "a1"}, Updated: now, Exchanges: []assistant.Exchange{{Question: "?", Asked: now}}}},
    }
    if err := database.SaveAssistantState("olivier", 0, state, map[string]string{"a1": "key"}); err != nil {
        t.Fatal(err)
    }
    if err := database.EraseCalendar("olivier", 1, start); err != nil {
        t.Fatal(err)
    }
    kept, _ := database.AssistantState("olivier")
    if (len(kept.Conversations) != 0) || (len(kept.Subscriptions) != 1) {
        t.Fatalf("got %+v", kept)
    }
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
