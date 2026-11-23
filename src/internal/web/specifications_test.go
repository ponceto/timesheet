/*
 * specifications_test.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "os"
    "path/filepath"
    "regexp"
    "slices"
    "strings"
    "testing"
)

// ---------------------------------------------------------------------------
// specifications tests
// ---------------------------------------------------------------------------

func TestRoutesMatchTheSpecifications(t *testing.T) {
    data, err := os.ReadFile(filepath.Join("..", "..", "..", "doc", "SPECIFICATIONS.md"))
    if err != nil {
        t.Fatal(err)
    }
    section := string(data)[strings.Index(string(data), "## ROUTES"):]
    section = section[:strings.Index(section, "\n## ")]
    rows := regexp.MustCompile("(?m)^\\| `([^`]+)` \\| ([A-Z]+) \\|").FindAllStringSubmatch(section, -1)
    documented := []string{}
    for _, row := range rows {
        documented = append(documented, row[2] + " " + row[1])
    }
    registered := []string{}
    for _, route := range newTestSite(t).site.Routes() {
        registered = append(registered, route.Method + " " + strings.ReplaceAll(route.Pattern, "{$}", ""))
    }
    slices.Sort(documented)
    slices.Sort(registered)
    if !slices.Equal(documented, registered) {
        t.Fatalf("routes differ:\ndocumented %v\nregistered %v", documented, registered)
    }
}

func TestNoticesMatchTheSpecifications(t *testing.T) {
    data, err := os.ReadFile(filepath.Join("..", "..", "..", "doc", "SPECIFICATIONS.md"))
    if err != nil {
        t.Fatal(err)
    }
    for _, code := range []string{"saved", "validated", "reopened", "adjusted", "unadjusted", "settings", "password", "name", "created", "disabled", "enabled", "sessions", "deleted", "profile", "reset", "configuration", "change-password", "imported", "erased", "subscription", "subscription-deleted", "shared", "unshared", "subscription-hidden", "subscription-shown", "cancelled", "conversation-deleted", "conversations-deleted"} {
        if NewNotices().Message(code) == "" {
            t.Errorf("notice %s has no message", code)
        }
        if !strings.Contains(string(data), "`" + code + "`") {
            t.Errorf("notice %s is not documented", code)
        }
    }
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
