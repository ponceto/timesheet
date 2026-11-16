/*
 * monthly_indicators.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "maps"
)

// ---------------------------------------------------------------------------
// MonthlyIndicators declaration
// ---------------------------------------------------------------------------

type MonthlyIndicators struct {
    Holidays    Quantity            `json:"holidays"`
    Total       Quantity            `json:"total"`
    WorkingDays Quantity            `json:"working_days"`
    Missing     Quantity            `json:"missing"`
    Upcoming    Quantity            `json:"upcoming"`
    ByCounter   map[string]Quantity `json:"by_counter"`
    ByCode      map[string]Quantity `json:"by_code"`
}

// ---------------------------------------------------------------------------
// MonthlyIndicators implementation
// ---------------------------------------------------------------------------

func (i MonthlyIndicators) Same(other MonthlyIndicators) bool {
    return (i.Holidays == other.Holidays) && (i.Total == other.Total) && (i.WorkingDays == other.WorkingDays) && (i.Missing == other.Missing) && (i.Upcoming == other.Upcoming) && maps.Equal(i.ByCounter, other.ByCounter) && maps.Equal(i.ByCode, other.ByCode)
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
