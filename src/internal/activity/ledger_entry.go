/*
 * ledger_entry.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "encoding/json"
    "time"
)

// ---------------------------------------------------------------------------
// LedgerEntry declaration
// ---------------------------------------------------------------------------

type LedgerEntry struct {
    ID      string    `json:"id"`
    Month   Month     `json:"month"`
    Date    Date      `json:"date"`
    Counter string    `json:"counter"`
    Kind    EntryKind `json:"kind"`
    Amount  Quantity  `json:"amount"`
    Label   string    `json:"label"`
    Ref     string    `json:"ref"`
    Created time.Time `json:"created"`
}

// ---------------------------------------------------------------------------
// LedgerEntry implementation
// ---------------------------------------------------------------------------

func (e LedgerEntry) Same(other LedgerEntry) bool {
    created := e.Created.Equal(other.Created)
    e.Created, other.Created = time.Time{}, time.Time{}
    return created && (e == other)
}

func (e LedgerEntry) MarshalJSON() ([]byte, error) {
    return json.Marshal(struct {
        ID      string       `json:"id"`
        Month   Month        `json:"month"`
        Date    Date         `json:"date"`
        Counter string       `json:"counter"`
        Kind    EntryKind    `json:"kind"`
        Amount  Quantity     `json:"amount"`
        Label   string       `json:"label"`
        Ref     NullableText `json:"ref"`
        Created NullableTime `json:"created"`
    }{e.ID, e.Month, e.Date, e.Counter, e.Kind, e.Amount, e.Label, NullableText(e.Ref), NullableTime(e.Created)})
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
