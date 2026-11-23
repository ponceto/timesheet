/*
 * server_configuration_test.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "errors"
    "strings"
    "testing"
    "time"
)

// ---------------------------------------------------------------------------
// ServerConfiguration tests
// ---------------------------------------------------------------------------

func TestDefaultServerConfigurationNamesNoDatabase(t *testing.T) {
    if NewDefaultServerConfiguration().Database.Path != "" {
        t.Fatal("the database has no default path")
    }
    if err := NewDefaultServerConfiguration().Validate(); err != nil {
        t.Fatalf("a configuration without database is valid: %v", err)
    }
    if value, err := NewDefaultServerConfiguration().Value("database.path"); (err != nil) || (value != "null") {
        t.Fatalf("no database is written null: %q %v", value, err)
    }
    if err := testServerConfiguration().Validate(); err != nil {
        t.Fatal(err)
    }
    for path, valid := range map[string]bool{"db/ts.json": true, "db/.ts.json": true, "db/ts.json.lock": false, "db/ts.json.sessions": false, "db/ts.json.1.tmp": false, "db/": false, "": false, "db/.": false, "db/..": false, "..": false, "db/ts.json.LOCK": false, "db/ts.json.Sessions": false, "db/TS.JSON": true} {
        if ValidDatabasePath(path) != valid {
            t.Errorf("%q: valid %v expected", path, valid)
        }
    }
}

func TestServerConfigurationValidationNamesTheField(t *testing.T) {
    cases := map[string]func(*ServerConfiguration){
        "database.path":                  func(c *ServerConfiguration) { c.Database.Path = "var/lib/ts.json.lock" },
        "database.max_bytes":             func(c *ServerConfiguration) { c.Database.MaxBytes = 1024 },
        "sessions.idle_timeout":          func(c *ServerConfiguration) { c.Sessions.IdleTimeout = Duration(time.Minute) },
        "sessions.max_lifetime":          func(c *ServerConfiguration) { c.Sessions.MaxLifetime = Duration(time.Hour) },
        "sessions.per_account":           func(c *ServerConfiguration) { c.Sessions.PerAccount = 0 },
        "sessions.save_interval":         func(c *ServerConfiguration) { c.Sessions.SaveInterval = Duration(time.Second) },
        "login.form_lifetime":            func(c *ServerConfiguration) { c.Login.FormLifetime = Duration(48 * time.Hour) },
        "login.initial_delay":            func(c *ServerConfiguration) { c.Login.InitialDelay = Duration(10 * time.Minute) },
        "login.max_delay":                func(c *ServerConfiguration) { c.Login.MaxDelay = Duration(2 * time.Hour) },
        "login.budget_window":            func(c *ServerConfiguration) { c.Login.BudgetWindow = Duration(time.Second) },
        "login.new_addresses_per_window": func(c *ServerConfiguration) { c.Login.NewAddressesPerWindow = 0 },
        "login.tracked_addresses":        func(c *ServerConfiguration) { c.Login.TrackedAddresses = 8 },
        "login.forget_after":             func(c *ServerConfiguration) { c.Login.ForgetAfter = Duration(time.Minute) },
        "passwords.iterations":           func(c *ServerConfiguration) { c.Passwords.Iterations = 1000 },
        "limits.accounts":                func(c *ServerConfiguration) { c.Limits.Accounts = 0 },
        "limits.account_bytes":           func(c *ServerConfiguration) { c.Limits.AccountBytes = 1 << 30 },
        "limits.adjustments":             func(c *ServerConfiguration) { c.Limits.Adjustments = 0 },
        "assistant.timeout":              func(c *ServerConfiguration) { c.Assistant.Timeout = Duration(time.Second) },
        "assistant.max_rounds":           func(c *ServerConfiguration) { c.Assistant.MaxRounds = 0 },
        "assistant.max_tokens":           func(c *ServerConfiguration) { c.Assistant.MaxTokens = 100 },
        "assistant.conversations":        func(c *ServerConfiguration) { c.Assistant.Conversations = 0 },
        "assistant.exchanges":            func(c *ServerConfiguration) { c.Assistant.Exchanges = 1000 },
        "assistant.question_bytes":       func(c *ServerConfiguration) { c.Assistant.QuestionBytes = 10 },
        "assistant.download_ttl":         func(c *ServerConfiguration) { c.Assistant.DownloadTTL = Duration(30 * time.Second) },
        "assistant.download_sweep":       func(c *ServerConfiguration) { c.Assistant.DownloadSweep = Duration(2 * time.Hour) },
    }
    for path, mutate := range cases {
        configuration := testServerConfiguration().Clone()
        mutate(&configuration)
        err := configuration.Validate()
        if !errors.Is(err, ErrInvalidConfiguration) || !strings.Contains(err.Error(), path + ":") {
            t.Errorf("%s: got %v", path, err)
        }
    }
    crowded := testServerConfiguration().Clone()
    crowded.Limits.Accounts = 1000
    if err := crowded.Validate(); !errors.Is(err, ErrInvalidConfiguration) || !strings.Contains(err.Error(), "sessions.per_account: at most 16384 sessions") {
        t.Fatalf("the sessions of every account must fit in the session file: %v", err)
    }
}

func TestServerConfigurationKeysReadAndWriteTheirValue(t *testing.T) {
    configuration := testServerConfiguration().Clone()
    if keys := configuration.Keys(); (len(keys) != 42) || (keys[0] != "server.listen") || (keys[len(keys) - 1] != "assistant.download_sweep") {
        t.Fatalf("got %v", keys)
    }
    values := map[string]string{"server.origins": "http://a.test:8080, http://b.test:8080", "server.tls_cert": "cert.pem", "sessions.idle_timeout": "1h30m", "limits.accounts": "300"}
    for key, value := range values {
        if err := configuration.Set(key, value); err != nil {
            t.Fatalf("%s: %v", key, err)
        }
    }
    wanted := map[string]string{"server.origins": "http://a.test:8080,http://b.test:8080", "server.tls_cert": "cert.pem", "server.tls_key": "null", "sessions.idle_timeout": "90m", "limits.accounts": "300", "database.path": "/srv/timesheet/timesheet-database.json"}
    for key, want := range wanted {
        if got, err := configuration.Value(key); (err != nil) || (got != want) {
            t.Errorf("%s: got %q %v", key, got, err)
        }
    }
    if err := configuration.Set("server.tls_cert", "null"); (err != nil) || (configuration.Server.TLSCert != "") {
        t.Fatalf("null removes a path: %v", err)
    }
    for key, value := range map[string]string{"sessions.idle_timeout": "soon", "limits.accounts": "many"} {
        if err := configuration.Set(key, value); !errors.Is(err, ErrInvalidConfiguration) || !strings.Contains(err.Error(), key + ":") {
            t.Errorf("%s: got %v", key, err)
        }
    }
    if _, err := configuration.Value("server.addr"); !errors.Is(err, ErrUnknownSetting) {
        t.Fatalf("got %v", err)
    }
    if err := configuration.Set("server.addr", "x"); !errors.Is(err, ErrUnknownSetting) {
        t.Fatalf("got %v", err)
    }
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
