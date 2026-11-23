/*
 * download_store.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "crypto/rand"
    "encoding/hex"
    "errors"
    "fmt"
    "io/fs"
    "log/slog"
    "os"
    "path/filepath"
    "slices"
    "strings"
    "sync"
    "time"

    "timesheet/internal/account"
    "timesheet/internal/activity"
    "timesheet/internal/assistant"
)

// ---------------------------------------------------------------------------
// DownloadStore declaration
// ---------------------------------------------------------------------------

type DownloadStore struct {
    mu        sync.Mutex
    directory string
    logger    *slog.Logger
}

type downloadEntry struct {
    name    string
    login   account.Login
    created time.Time
}

const (
    MaxDownloadsPerAccount = 20
    maxDownloadBytes       = 16 << 20
    downloadIDBytes        = 16
    downloadExtension      = ".json"
)

// ---------------------------------------------------------------------------
// DownloadStore implementation
// ---------------------------------------------------------------------------

func NewDownloadStore(directory string, logger *slog.Logger) *DownloadStore {
    return &DownloadStore{directory: directory, logger: logger}
}

func (s *DownloadStore) Directory() string {
    if absolute, err := filepath.Abs(s.directory); err == nil {
        return absolute
    }
    return s.directory
}

func (s *DownloadStore) Save(login account.Login, data []byte, now time.Time) (string, error) {
    if !login.Valid() {
        return "", account.ErrInvalidLogin
    }
    s.mu.Lock()
    defer s.mu.Unlock()
    if err := s.prepare(); err != nil {
        return "", err
    }
    own := []downloadEntry{}
    for _, entry := range s.entries() {
        if entry.login == login {
            own = append(own, entry)
        }
    }
    slices.SortFunc(own, s.older)
    for len(own) >= MaxDownloadsPerAccount {
        os.Remove(filepath.Join(s.directory, own[0].name))
        own = own[1:]
    }
    buffer := make([]byte, downloadIDBytes)
    if _, err := rand.Read(buffer); err != nil {
        return "", err
    }
    id := hex.EncodeToString(buffer)
    path := s.path(id, login)
    if err := NewSecureFile(path, s.logger, maxDownloadBytes).Write(data); err != nil {
        return "", err
    }
    if err := os.Chtimes(path, now, now); err != nil {
        os.Remove(path)
        return "", err
    }
    return id, nil
}

func (s *DownloadStore) Open(login account.Login, id string, now time.Time, ttl time.Duration) (assistant.Download, error) {
    if _, owner, valid := s.parse(id + "-" + string(login) + downloadExtension); !valid || (owner != login) {
        return assistant.Download{}, assistant.ErrDownloadMissing
    }
    s.mu.Lock()
    defer s.mu.Unlock()
    path := s.path(id, login)
    info, err := os.Lstat(path)
    if errors.Is(err, fs.ErrNotExist) {
        return assistant.Download{}, assistant.ErrDownloadMissing
    }
    if err != nil {
        return assistant.Download{}, err
    }
    if !info.Mode().IsRegular() {
        return assistant.Download{}, assistant.ErrDownloadMissing
    }
    if now.Sub(info.ModTime()) > ttl {
        os.Remove(path)
        return assistant.Download{}, assistant.ErrDownloadExpired
    }
    data, found, err := NewSecureFile(path, s.logger, maxDownloadBytes).Read()
    if !found {
        return assistant.Download{}, assistant.ErrDownloadMissing
    }
    return assistant.Download{Data: data, Created: info.ModTime()}, err
}

func (s *DownloadStore) Sweep(now time.Time, ttl time.Duration) int {
    s.mu.Lock()
    defer s.mu.Unlock()
    entries, err := os.ReadDir(s.directory)
    if err != nil {
        return 0
    }
    removed := 0
    for _, entry := range entries {
        info, err := entry.Info()
        if (err != nil) || !info.Mode().IsRegular() || (now.Sub(info.ModTime()) <= ttl) {
            continue
        }
        if _, _, valid := s.parse(entry.Name()); valid || strings.HasSuffix(entry.Name(), activity.TemporarySuffix) {
            if os.Remove(filepath.Join(s.directory, entry.Name())) == nil {
                removed++
            }
        }
    }
    return removed
}

func (s *DownloadStore) RemoveAccount(login account.Login) {
    s.mu.Lock()
    defer s.mu.Unlock()
    for _, entry := range s.entries() {
        if entry.login == login {
            os.Remove(filepath.Join(s.directory, entry.name))
        }
    }
}

func (s *DownloadStore) prepare() error {
    info, err := os.Lstat(s.directory)
    if errors.Is(err, fs.ErrNotExist) {
        if err := os.Mkdir(s.directory, 0o700); err != nil {
            return err
        }
        info, err = os.Lstat(s.directory)
    }
    if err != nil {
        return err
    }
    if !info.IsDir() || ((info.Mode().Perm() & 0o077) != 0) {
        return fmt.Errorf("%w: %s must be a directory without any right for the group or others", ErrInsecureDirectory, s.Directory())
    }
    return nil
}

func (s *DownloadStore) entries() []downloadEntry {
    found := []downloadEntry{}
    entries, err := os.ReadDir(s.directory)
    if err != nil {
        return found
    }
    for _, entry := range entries {
        _, login, valid := s.parse(entry.Name())
        info, err := entry.Info()
        if !valid || (err != nil) || !info.Mode().IsRegular() {
            continue
        }
        found = append(found, downloadEntry{name: entry.Name(), login: login, created: info.ModTime()})
    }
    return found
}

func (s *DownloadStore) parse(name string) (string, account.Login, bool) {
    id, rest, found := strings.Cut(name, "-")
    login := account.Login(strings.TrimSuffix(rest, downloadExtension))
    if !found || (len(id) != (2 * downloadIDBytes)) || !strings.HasSuffix(rest, downloadExtension) || !login.Valid() {
        return "", "", false
    }
    if _, err := hex.DecodeString(id); (err != nil) || (strings.ToLower(id) != id) {
        return "", "", false
    }
    return id, login, true
}

func (s *DownloadStore) older(a downloadEntry, b downloadEntry) int {
    return a.created.Compare(b.created)
}

func (s *DownloadStore) path(id string, login account.Login) string {
    return filepath.Join(s.directory, id + "-" + string(login) + downloadExtension)
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
