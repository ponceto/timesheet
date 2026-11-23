/*
 * database_settings.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "path/filepath"
    "strings"
)

// ---------------------------------------------------------------------------
// DatabaseSettings declaration
// ---------------------------------------------------------------------------

type DatabaseSettings struct {
    Path     NullableText `json:"path"`
    MaxBytes int64        `json:"max_bytes"`
}

const (
    LockSuffix      = ".lock"
    SessionsSuffix  = ".sessions"
    KeySuffix       = ".key"
    DownloadsSuffix = ".downloads"
    TemporarySuffix = ".tmp"

    minDatabaseBytes = 1 << 20
    maxDatabaseBytes = 1 << 30
)

// ---------------------------------------------------------------------------
// DatabaseSettings implementation
// ---------------------------------------------------------------------------

func NewDefaultDatabaseSettings() DatabaseSettings {
    return DatabaseSettings{MaxBytes: 64 << 20}
}

func ValidDatabasePath(path string) bool {
    if (path == "") || strings.HasSuffix(path, "/") {
        return false
    }
    name := strings.ToLower(filepath.Base(path))
    if (name == ".") || (name == "..") {
        return false
    }
    return !strings.HasSuffix(name, LockSuffix) && !strings.HasSuffix(name, SessionsSuffix) && !strings.HasSuffix(name, KeySuffix) && !strings.HasSuffix(name, DownloadsSuffix) && !strings.HasSuffix(name, TemporarySuffix)
}

func (s DatabaseSettings) Validate() error {
    check := NewSettingCheck("database")
    if (s.Path != "") && !ValidDatabasePath(string(s.Path)) {
        return check.Fail("path", "a file, not . or .., whose name does not end with " + LockSuffix + ", " + SessionsSuffix + ", " + KeySuffix + ", " + DownloadsSuffix + " or " + TemporarySuffix + " in any case is expected")
    }
    return check.Integer("max_bytes", s.MaxBytes, minDatabaseBytes, maxDatabaseBytes)
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
