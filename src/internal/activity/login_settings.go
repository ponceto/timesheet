/*
 * login_settings.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "time"
)

// ---------------------------------------------------------------------------
// LoginSettings declaration
// ---------------------------------------------------------------------------

type LoginSettings struct {
    FormLifetime          Duration `json:"form_lifetime"`
    InitialDelay          Duration `json:"initial_delay"`
    MaxDelay              Duration `json:"max_delay"`
    BudgetWindow          Duration `json:"budget_window"`
    NewAddressesPerWindow int64    `json:"new_addresses_per_window"`
    TrackedAddresses      int64    `json:"tracked_addresses"`
    ForgetAfter           Duration `json:"forget_after"`
}

const (
    minFormLifetime     = time.Minute
    maxFormLifetime     = 24 * time.Hour
    minInitialDelay     = 100 * time.Millisecond
    maxInitialDelay     = 10 * time.Second
    minMaxDelay         = time.Second
    maxMaxDelay         = time.Hour
    minBudgetWindow     = 10 * time.Second
    maxBudgetWindow     = time.Hour
    maxNewAddresses     = 10000
    minTrackedAddresses = 16
    maxTrackedAddresses = 65536
    maxForgetAfter      = 7 * 24 * time.Hour
)

// ---------------------------------------------------------------------------
// LoginSettings implementation
// ---------------------------------------------------------------------------

func NewDefaultLoginSettings() LoginSettings {
    return LoginSettings{FormLifetime: Duration(time.Hour), InitialDelay: Duration(time.Second), MaxDelay: Duration(5 * time.Minute), BudgetWindow: Duration(time.Minute), NewAddressesPerWindow: 100, TrackedAddresses: 4096, ForgetAfter: Duration(time.Hour)}
}

func (s LoginSettings) Validate() error {
    check := NewSettingCheck("login")
    if err := check.Duration("form_lifetime", s.FormLifetime, minFormLifetime, maxFormLifetime); err != nil {
        return err
    }
    if err := check.Duration("max_delay", s.MaxDelay, minMaxDelay, maxMaxDelay); err != nil {
        return err
    }
    if err := check.Duration("initial_delay", s.InitialDelay, minInitialDelay, min(maxInitialDelay, s.MaxDelay.Std())); err != nil {
        return err
    }
    if err := check.Duration("budget_window", s.BudgetWindow, minBudgetWindow, maxBudgetWindow); err != nil {
        return err
    }
    if err := check.Integer("new_addresses_per_window", s.NewAddressesPerWindow, 1, maxNewAddresses); err != nil {
        return err
    }
    if err := check.Integer("tracked_addresses", s.TrackedAddresses, minTrackedAddresses, maxTrackedAddresses); err != nil {
        return err
    }
    return check.Duration("forget_after", s.ForgetAfter, s.MaxDelay.Std(), maxForgetAfter)
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
