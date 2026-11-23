/*
 * environment_overrides_test.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "bytes"
    "context"
    "errors"
    "strings"
    "testing"
    "time"

    "timesheet/internal/activity"
)

// ---------------------------------------------------------------------------
// EnvironmentOverrides tests
// ---------------------------------------------------------------------------

func TestEnvironmentOverridesReplaceTheDownloadSettings(t *testing.T) {
    variables := map[string]string{DownloadTTLVariable: "2m", DownloadSweepVariable: ""}
    lookup := func(name string) (string, bool) {
        value, set := variables[name]
        return value, set
    }
    server, err := NewEnvironmentOverrides(lookup).Apply(activity.NewDefaultServerConfiguration())
    if (err != nil) || (server.Assistant.DownloadTTL.Std() != 2 * time.Minute) || (server.Assistant.DownloadSweep.Std() != 5 * time.Minute) {
        t.Fatalf("got %+v %v", server.Assistant, err)
    }
    variables[DownloadSweepVariable] = "2h"
    if _, err := NewEnvironmentOverrides(lookup).Apply(activity.NewDefaultServerConfiguration()); !errors.Is(err, activity.ErrInvalidConfiguration) || !strings.Contains(err.Error(), "assistant.download_sweep") {
        t.Fatalf("an out-of-bounds value is refused: %v", err)
    }
}

func TestRunStopsOnAnInvalidEnvironmentVariable(t *testing.T) {
    _, database := newTestDirectory(t)
    lookup := func(name string) (string, bool) {
        return "soon", name == DownloadTTLVariable
    }
    stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
    code := NewApplication([]string{"run", database, "--listen=127.0.0.1:0"}, lookup, strings.NewReader(""), stdout, stderr, time.UTC).Run(context.Background())
    if (code != ExitError) || !strings.Contains(stderr.String(), "invalid environment variable") || !strings.Contains(stderr.String(), "assistant.download_ttl") {
        t.Fatalf("got %d %s", code, stderr.String())
    }
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
