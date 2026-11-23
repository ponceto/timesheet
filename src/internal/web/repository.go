/*
 * repository.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "timesheet/internal/activity"
    "timesheet/internal/assistant"
)

// ---------------------------------------------------------------------------
// Repository declaration
// ---------------------------------------------------------------------------

type Repository interface {
    Configuration() activity.Configuration
    Account(login account.Login) (account.Account, activity.Calendar, bool)
    Accounts() []account.Account
    ActivityUsed(id string) bool
    CounterUsed(id string) bool
    CreateAccount(created account.Account, calendar activity.Calendar) error
    DeleteAccount(login account.Login) error
    SetName(login account.Login, name string) error
    SetProfile(login account.Login, profile account.Profile) error
    SetState(login account.Login, state account.State) error
    SetPassword(login account.Login, hash account.PasswordHash, mustChange bool) error
    ChangePassword(login account.Login, current account.PasswordHash, hash account.PasswordHash) error
    TouchLogin(login account.Login, at time.Time) error
    SaveCalendar(login account.Login, revision int64, calendar activity.Calendar) error
    EraseCalendar(login account.Login, revision int64, start activity.Date) error
    CheckCalendar(login account.Login, revision int64, calendar activity.Calendar) error
    SaveConfiguration(revision int64, config activity.Configuration) error
    AssistantState(login account.Login) (assistant.State, bool)
    SharedSubscriptions(login account.Login) []assistant.SharedSubscription
    SaveAssistantState(login account.Login, revision int64, state assistant.State, keys map[string]string) error
    SubscriptionKey(login account.Login, ref assistant.SubscriptionRef) (assistant.Subscription, string, error)
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
