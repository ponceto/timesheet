/*
 * build_info.go - Copyright (c) 2023-2026 - Olivier Poncet
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

package app

import (
    "runtime/debug"
    "strings"
)

// ---------------------------------------------------------------------------
// BuildInfo declaration
// ---------------------------------------------------------------------------

type BuildInfo struct {
    version  string
    revision string
    time     string
    modified bool
}

const ReleaseVersion = "0.50.0"

// ---------------------------------------------------------------------------
// BuildInfo implementation
// ---------------------------------------------------------------------------

func NewBuildInfo() *BuildInfo {
    info := &BuildInfo{version: ReleaseVersion}
    build, ok := debug.ReadBuildInfo()
    if !ok {
        return info
    }
    for _, setting := range build.Settings {
        switch setting.Key {
            case "vcs.revision":
                info.revision = setting.Value
            case "vcs.time":
                info.time = setting.Value
            case "vcs.modified":
                info.modified = setting.Value == "true"
        }
    }
    return info
}

func (b *BuildInfo) Version() string {
    return b.version
}

func (b *BuildInfo) String() string {
    details := []string{}
    if b.revision != "" {
        details = append(details, "revision " + b.revision)
    }
    if b.time != "" {
        details = append(details, b.time)
    }
    if b.modified {
        details = append(details, "modified")
    }
    text := "timesheet " + b.version
    if len(details) > 0 {
        text += " (" + strings.Join(details, ", ") + ")"
    }
    return text
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
