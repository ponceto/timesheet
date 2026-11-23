/*
 * site_dependencies.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "io"
    "log/slog"
    "net/http"
    "net/netip"

    "timesheet/internal/activity"
)

// ---------------------------------------------------------------------------
// SiteDependencies declaration
// ---------------------------------------------------------------------------

type SiteDependencies struct {
    Repository     Repository
    Sessions       SessionArchive
    Clock          activity.Clock
    Random         io.Reader
    Logger         *slog.Logger
    Origins        *Origins
    Server         activity.ServerConfiguration
    TrustedProxies activity.TrustedProxies
    LocalPrefixes  []netip.Prefix
    Iterations     int
    Version        string
    Client         *http.Client
    Downloads      Downloads
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
