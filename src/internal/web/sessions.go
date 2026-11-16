/*
 * sessions.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "crypto/sha256"
    "encoding/base64"
    "encoding/hex"
    "errors"
    "io"
    "log/slog"
    "net/http"
    "slices"
    "sync"
    "time"

    "timesheet/internal/account"
    "timesheet/internal/activity"
)

// ---------------------------------------------------------------------------
// Sessions declaration
// ---------------------------------------------------------------------------

type Sessions struct {
    mu         sync.Mutex
    writing    sync.Mutex
    clock      activity.Clock
    random     io.Reader
    secure     bool
    sliding    time.Duration
    absolute   time.Duration
    max        int
    interval   time.Duration
    sessions   map[string]*Session
    archive    SessionArchive
    logger     *slog.Logger
    saved      time.Time
    generation int64
    written    int64
}

type sessionSnapshot struct {
    records    []account.SessionRecord
    generation int64
}

const (
    SessionCookie = "ts_session"
    sessionBytes  = 32
    securePrefix  = "__Host-"
)

var ErrRandomFailure = errors.New("web: random source failed")

// ---------------------------------------------------------------------------
// Sessions implementation
// ---------------------------------------------------------------------------

func NewSessions(clock activity.Clock, random io.Reader, secure bool, settings activity.SessionSettings) *Sessions {
    return &Sessions{clock: clock, random: random, secure: secure, sliding: settings.IdleTimeout.Std(), absolute: settings.MaxLifetime.Std(), max: int(settings.PerAccount), interval: settings.SaveInterval.Std(), sessions: map[string]*Session{}}
}

func (s *Sessions) Restore(archive SessionArchive, logger *slog.Logger) {
    s.mu.Lock()
    s.archive, s.logger = archive, logger
    records, err := archive.LoadSessions()
    if err != nil {
        logger.Warn("sessions not restored", "error", err.Error())
    }
    now := s.clock.Now()
    for _, record := range records {
        session := &Session{login: record.Login, csrf: record.CSRF, created: record.Created, seen: record.Seen}
        if !session.Expired(now, s.sliding, s.absolute) {
            s.sessions[record.Key] = session
        }
    }
    snapshot := s.snapshot(now)
    s.mu.Unlock()
    s.write(snapshot)
}

func (s *Sessions) Flush() {
    s.mu.Lock()
    snapshot := s.snapshot(s.clock.Now())
    s.mu.Unlock()
    s.write(snapshot)
}

func (s *Sessions) snapshot(now time.Time) sessionSnapshot {
    if s.archive == nil {
        return sessionSnapshot{}
    }
    records := []account.SessionRecord{}
    for key, session := range s.sessions {
        records = append(records, account.SessionRecord{Key: key, Login: session.login, CSRF: session.csrf, Created: session.created, Seen: session.seen})
    }
    slices.SortFunc(records, s.compare)
    s.generation++
    s.saved = now
    return sessionSnapshot{records: records, generation: s.generation}
}

func (s *Sessions) write(snapshot sessionSnapshot) {
    if snapshot.generation == 0 {
        return
    }
    s.writing.Lock()
    defer s.writing.Unlock()
    if snapshot.generation <= s.written {
        return
    }
    s.written = snapshot.generation
    if err := s.archive.SaveSessions(snapshot.records); err != nil {
        s.logger.Error("cannot save the sessions", "error", err.Error())
    }
}

func (s *Sessions) compare(a account.SessionRecord, b account.SessionRecord) int {
    return a.Created.Compare(b.Created)
}

func (s *Sessions) Token() (string, error) {
    buffer := make([]byte, sessionBytes)
    if _, err := io.ReadFull(s.random, buffer); err != nil {
        return "", ErrRandomFailure
    }
    return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func (s *Sessions) Hash(id string) string {
    sum := sha256.Sum256([]byte(id))
    return hex.EncodeToString(sum[:])
}

func (s *Sessions) Open(login account.Login, previous string) (string, error) {
    id, err := s.Token()
    if err != nil {
        return "", err
    }
    csrf, err := s.Token()
    if err != nil {
        return "", err
    }
    s.mu.Lock()
    now := s.clock.Now()
    if previous != "" {
        delete(s.sessions, s.Hash(previous))
    }
    s.prune(now)
    count, oldest, oldestKey := 0, now, ""
    for key, session := range s.sessions {
        if session.login == login {
            count++
            if session.created.Before(oldest) || (oldestKey == "") {
                oldest, oldestKey = session.created, key
            }
        }
    }
    if (count >= s.max) && (oldestKey != "") {
        delete(s.sessions, oldestKey)
    }
    s.sessions[s.Hash(id)] = NewSession(login, csrf, now)
    snapshot := s.snapshot(now)
    s.mu.Unlock()
    s.write(snapshot)
    return id, nil
}

func (s *Sessions) Lookup(id string) (account.Login, string, string, bool) {
    if id == "" {
        return "", "", "", false
    }
    key := s.Hash(id)
    s.mu.Lock()
    session, ok := s.sessions[key]
    now := s.clock.Now()
    if !ok || session.Expired(now, s.sliding, s.absolute) {
        snapshot := sessionSnapshot{}
        if ok {
            delete(s.sessions, key)
            snapshot = s.snapshot(now)
        }
        s.mu.Unlock()
        s.write(snapshot)
        return "", "", "", false
    }
    session.seen = now
    snapshot := sessionSnapshot{}
    if now.Sub(s.saved) >= s.interval {
        snapshot = s.snapshot(now)
    }
    login, csrf := session.login, session.csrf
    s.mu.Unlock()
    s.write(snapshot)
    return login, csrf, key, true
}

func (s *Sessions) SetNotice(key string, code string) {
    s.mu.Lock()
    defer s.mu.Unlock()
    if session, ok := s.sessions[key]; ok {
        session.notice = code
    }
}

func (s *Sessions) TakeNotice(key string) string {
    s.mu.Lock()
    defer s.mu.Unlock()
    session, ok := s.sessions[key]
    if !ok {
        return ""
    }
    code := session.notice
    session.notice = ""
    return code
}

func (s *Sessions) Created(key string) time.Time {
    s.mu.Lock()
    defer s.mu.Unlock()
    if session, ok := s.sessions[key]; ok {
        return session.created
    }
    return time.Time{}
}

func (s *Sessions) Close(key string) {
    s.mu.Lock()
    delete(s.sessions, key)
    snapshot := s.snapshot(s.clock.Now())
    s.mu.Unlock()
    s.write(snapshot)
}

func (s *Sessions) CloseAccount(login account.Login, keep string) {
    s.mu.Lock()
    for key, session := range s.sessions {
        if (session.login == login) && (key != keep) {
            delete(s.sessions, key)
        }
    }
    snapshot := s.snapshot(s.clock.Now())
    s.mu.Unlock()
    s.write(snapshot)
}

func (s *Sessions) Count(login account.Login) int {
    s.mu.Lock()
    defer s.mu.Unlock()
    now := s.clock.Now()
    count := 0
    for _, session := range s.sessions {
        if (session.login == login) && !session.Expired(now, s.sliding, s.absolute) {
            count++
        }
    }
    return count
}

func (s *Sessions) prune(now time.Time) {
    for key, session := range s.sessions {
        if session.Expired(now, s.sliding, s.absolute) {
            delete(s.sessions, key)
        }
    }
}

func (s *Sessions) CookieName(name string) string {
    if s.secure {
        return securePrefix + name
    }
    return name
}

func (s *Sessions) Cookie(name string, value string, maxAge int) *http.Cookie {
    return &http.Cookie{Name: s.CookieName(name), Value: value, Path: "/", MaxAge: maxAge, HttpOnly: true, Secure: s.secure, SameSite: http.SameSiteStrictMode}
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
