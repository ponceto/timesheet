/*
 * download.go - Copyright (c) 2023-2026 - Olivier Poncet
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

package assistant

import (
    "errors"
    "time"
)

// ---------------------------------------------------------------------------
// Download declaration
// ---------------------------------------------------------------------------

type Download struct {
    Data    []byte
    Created time.Time
}

const (
    DownloadsPath    = "/dashboard/assistant/downloads/"
    downloadIDLength = 32
)

var (
    ErrDownloadMissing = errors.New("assistant: unknown download")
    ErrDownloadExpired = errors.New("assistant: expired download")
)

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
