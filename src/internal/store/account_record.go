/*
 * account_record.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "encoding/json"
    "time"

    "timesheet/internal/account"
    "timesheet/internal/activity"
    "timesheet/internal/assistant"
)

// ---------------------------------------------------------------------------
// AccountRecord declaration
// ---------------------------------------------------------------------------

type AccountRecord struct {
    account.Account
    activity.Calendar
    Assistant *assistant.State `json:"assistant"`
}

// ---------------------------------------------------------------------------
// AccountRecord implementation
// ---------------------------------------------------------------------------

func (r AccountRecord) MarshalJSON() ([]byte, error) {
    return json.Marshal(struct {
        Login              account.Login         `json:"login"`
        Name               activity.NullableText `json:"name"`
        Profile            account.Profile       `json:"profile"`
        Password           account.PasswordHash  `json:"password"`
        MustChangePassword bool                  `json:"must_change_password"`
        State              account.State         `json:"state"`
        Created            time.Time             `json:"created"`
        LastLogin          activity.NullableTime `json:"last_login"`
        activity.Calendar
        Assistant          *assistant.State      `json:"assistant"`
    }{r.Login, activity.NullableText(r.Name), r.Profile, r.Password, r.MustChangePassword, r.State, r.Created, activity.NullableTime(r.LastLogin), r.Calendar, r.Assistant})
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
