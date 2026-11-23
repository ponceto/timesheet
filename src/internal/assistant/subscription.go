/*
 * subscription.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "slices"
    "time"

    "timesheet/internal/account"
)

// ---------------------------------------------------------------------------
// Subscription declaration
// ---------------------------------------------------------------------------

type Subscription struct {
    ID         string          `json:"id"`
    Name       string          `json:"name"`
    Provider   string          `json:"provider"`
    Model      string          `json:"model"`
    Key        string          `json:"key"`
    SharedWith []account.Login `json:"shared_with"`
    Created    time.Time       `json:"created"`
}

const (
    MaxSubscriptions    = 20
    MaxShares           = 50
    MaxSubscriptionName = 60
    MaxKeyLength        = 512
)

// ---------------------------------------------------------------------------
// Subscription implementation
// ---------------------------------------------------------------------------

func (s Subscription) SharedTo(login account.Login) bool {
    return slices.Contains(s.SharedWith, login)
}

func (s Subscription) Clone() Subscription {
    clone := s
    clone.SharedWith = append([]account.Login{}, s.SharedWith...)
    return clone
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
