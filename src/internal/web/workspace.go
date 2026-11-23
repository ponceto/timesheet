/*
 * workspace.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "strconv"
    "time"

    "timesheet/internal/account"
    "timesheet/internal/activity"
)

// ---------------------------------------------------------------------------
// Workspace declaration
// ---------------------------------------------------------------------------

type Workspace struct {
    Viewer   *Viewer
    Owner    account.Account
    Calendar activity.Calendar
    Config   activity.Configuration
    Book     *activity.Bookkeeper
    Today    activity.Date
    Location *time.Location
    Base     string
    ReadOnly bool
}

const (
    PageActivity = "activity"
    PageMonth    = "month"
)

// ---------------------------------------------------------------------------
// Workspace implementation
// ---------------------------------------------------------------------------

func (w *Workspace) Horizon() activity.Month {
    horizon := w.Today.Month()
    for range PlanningMonths {
        horizon = horizon.Next()
    }
    return horizon
}

func (w *Workspace) PagePath(page string, m activity.Month) string {
    return w.Base + "/" + page + "/" + m.Path()
}

func (w *Workspace) MonthPath(m activity.Month) string {
    return w.PagePath(PageMonth, m)
}

func (w *Workspace) FormPath(m activity.Month) string {
    return "/dashboard/" + PageActivity + "/" + m.Path()
}

func (w *Workspace) ExportName(d activity.Date) string {
    return "timesheet-" + string(w.Owner.Login) + "-" + d.String() + ".json"
}

func (w *Workspace) YearPath(year int) string {
    return w.Base + "/year/" + strconv.Itoa(year)
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
