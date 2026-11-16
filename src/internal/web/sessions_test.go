/*
 * sessions_test.go - Copyright (c) 2023-2026 - Olivier Poncet
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

package web

import (
    "bytes"
    "log/slog"
    "strings"
    "testing"
    "time"

    "timesheet/internal/account"
    "timesheet/internal/activity"
)

// ---------------------------------------------------------------------------
// Sessions tests
// ---------------------------------------------------------------------------

func TestExpiredSessionsAreNotCounted(t *testing.T) {
    clock := activity.NewManualClock(time.Date(2026, time.November, 11, 21, 0, 0, 0, time.UTC))
    sessions := NewSessions(clock, strings.NewReader(strings.Repeat("a", 256)), false, activity.NewDefaultSessionSettings())
    if _, err := sessions.Open("oponcet", ""); err != nil {
        t.Fatal(err)
    }
    if sessions.Count("oponcet") != 1 {
        t.Fatal("an open session is counted")
    }
    clock.Advance(activity.NewDefaultSessionSettings().IdleTimeout.Std() + time.Minute)
    if count := sessions.Count("oponcet"); count != 0 {
        t.Fatalf("an expired session is still counted: %d", count)
    }
}

func TestSessionsSurviveARestart(t *testing.T) {
    clock := activity.NewManualClock(time.Date(2026, time.November, 11, 21, 0, 0, 0, time.UTC))
    archive := &memoryArchive{}
    logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
    before := NewSessions(clock, strings.NewReader(strings.Repeat("a", 256)), false, activity.NewDefaultSessionSettings())
    before.Restore(archive, logger)
    id, err := before.Open("oponcet", "")
    if (err != nil) || (len(archive.records) != 1) || (archive.records[0].Key != before.Hash(id)) || strings.Contains(archive.records[0].Key, id) {
        t.Fatalf("saved sessions: %+v %v", archive.records, err)
    }
    saves := archive.saves
    clock.Advance(10 * time.Second)
    before.Lookup(id)
    if archive.saves != saves {
        t.Fatal("the sliding expiry must not be saved at every request")
    }
    clock.Advance(2 * time.Minute)
    before.Lookup(id)
    if (archive.saves != (saves + 1)) || !archive.records[0].Seen.Equal(clock.Now()) {
        t.Fatal("the sliding expiry must be saved once a minute")
    }
    after := NewSessions(clock, strings.NewReader(strings.Repeat("b", 256)), false, activity.NewDefaultSessionSettings())
    after.Restore(archive, logger)
    if login, _, _, found := after.Lookup(id); !found || (login != "oponcet") {
        t.Fatal("a session must survive a restart")
    }
    after.CloseAccount("oponcet", "")
    if len(archive.records) != 0 {
        t.Fatal("closed sessions must leave the file")
    }
    again, _ := after.Open("oponcet", "")
    clock.Advance(10 * time.Second)
    after.Lookup(again)
    after.Flush()
    if (len(archive.records) != 1) || !archive.records[0].Seen.Equal(clock.Now()) {
        t.Fatal("a flush must save the last activity")
    }
    stale := sessionSnapshot{records: []account.SessionRecord{}, generation: 1}
    saves = archive.saves
    after.write(stale)
    if (archive.saves != saves) || (len(archive.records) != 1) {
        t.Fatal("an older state of the sessions must never replace a newer one")
    }
    clock.Advance(activity.NewDefaultSessionSettings().MaxLifetime.Std())
    archive.records = []account.SessionRecord{{Key: strings.Repeat("ab", 32), Login: "oponcet", CSRF: "t", Created: clock.Now().Add(-activity.NewDefaultSessionSettings().MaxLifetime.Std() - time.Hour), Seen: clock.Now()}}
    expired := NewSessions(clock, strings.NewReader(""), false, activity.NewDefaultSessionSettings())
    expired.Restore(archive, logger)
    if len(archive.records) != 0 {
        t.Fatal("expired sessions must be dropped at restart")
    }
}

func TestSecureCookiesTakeTheHostPrefix(t *testing.T) {
    clock := activity.NewManualClock(time.Date(2026, time.November, 11, 21, 0, 0, 0, time.UTC))
    plain := NewSessions(clock, strings.NewReader(""), false, activity.NewDefaultSessionSettings())
    secure := NewSessions(clock, strings.NewReader(""), true, activity.NewDefaultSessionSettings())
    if cookie := plain.Cookie(SessionCookie, "x", 0); (cookie.Name != "ts_session") || cookie.Secure {
        t.Fatalf("plain HTTP: %+v", cookie)
    }
    if cookie := secure.Cookie(LoginCookie, "x", 60); (cookie.Name != "__Host-ts_login") || !cookie.Secure || (cookie.Path != "/") || (cookie.Domain != "") {
        t.Fatalf("https: %+v", cookie)
    }
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
