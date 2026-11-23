/*
 * environment_overrides.go - Copyright (c) 2023-2026 - Olivier Poncet
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

package app

import (
    "timesheet/internal/activity"
)

// ---------------------------------------------------------------------------
// EnvironmentOverrides declaration
// ---------------------------------------------------------------------------

type EnvironmentOverrides struct {
    lookupEnv func(string) (string, bool)
}

const (
    DownloadTTLVariable   = "TIMESHEET_ASSISTANT_DOWNLOAD_TTL"
    DownloadSweepVariable = "TIMESHEET_ASSISTANT_DOWNLOAD_SWEEP"
)

// ---------------------------------------------------------------------------
// EnvironmentOverrides implementation
// ---------------------------------------------------------------------------

func NewEnvironmentOverrides(lookupEnv func(string) (string, bool)) *EnvironmentOverrides {
    return &EnvironmentOverrides{lookupEnv: lookupEnv}
}

func (o *EnvironmentOverrides) Keys() map[string]string {
    return map[string]string{DownloadTTLVariable: "assistant.download_ttl", DownloadSweepVariable: "assistant.download_sweep"}
}

func (o *EnvironmentOverrides) Apply(server activity.ServerConfiguration) (activity.ServerConfiguration, error) {
    server = server.Clone()
    for _, variable := range []string{DownloadTTLVariable, DownloadSweepVariable} {
        value, set := o.lookupEnv(variable)
        if !set || (value == "") {
            continue
        }
        if err := server.Set(o.Keys()[variable], value); err != nil {
            return activity.ServerConfiguration{}, err
        }
    }
    return server, server.Validate()
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
