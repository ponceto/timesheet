/*
 * downloads.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "time"

    "timesheet/internal/account"
    "timesheet/internal/assistant"
)

// ---------------------------------------------------------------------------
// Downloads declaration
// ---------------------------------------------------------------------------

type Downloads interface {
    Save(login account.Login, data []byte, now time.Time) (string, error)
    Open(login account.Login, id string, now time.Time, ttl time.Duration) (assistant.Download, error)
    Sweep(now time.Time, ttl time.Duration) int
    RemoveAccount(login account.Login)
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
