/*
 * server_configuration.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "fmt"
    "strconv"
    "strings"
)

// ---------------------------------------------------------------------------
// ServerConfiguration declaration
// ---------------------------------------------------------------------------

type ServerConfiguration struct {
    Server    ServerSettings    `json:"server"`
    Database  DatabaseSettings  `json:"database"`
    Sessions  SessionSettings   `json:"sessions"`
    Login     LoginSettings     `json:"login"`
    Passwords PasswordSettings  `json:"passwords"`
    Limits    LimitSettings     `json:"limits"`
    Assistant AssistantSettings `json:"assistant"`
}

type serverConfigurationKey struct {
    name string
    get  func() string
    set  func(string) error
}

const (
    MaxRestorableSessions = 16384

    noValue = "null"
)

var ErrUnknownSetting = errors.New("activity: unknown setting")

// ---------------------------------------------------------------------------
// ServerConfiguration implementation
// ---------------------------------------------------------------------------

func NewDefaultServerConfiguration() ServerConfiguration {
    return ServerConfiguration{Server: NewDefaultServerSettings(), Database: NewDefaultDatabaseSettings(), Sessions: NewDefaultSessionSettings(), Login: NewDefaultLoginSettings(), Passwords: NewDefaultPasswordSettings(), Limits: NewDefaultLimitSettings(), Assistant: NewDefaultAssistantSettings()}
}

func NewOptionalServerConfiguration() ServerConfiguration {
    configuration := NewDefaultServerConfiguration()
    configuration.Server = NewOptionalServerSettings()
    return configuration
}

func (c ServerConfiguration) Validate() error {
    checks := []func() error{c.Server.Validate, c.Database.Validate, c.Sessions.Validate, c.Login.Validate, c.Passwords.Validate, c.Limits.Validate, c.Assistant.Validate}
    for _, check := range checks {
        if err := check(); err != nil {
            return err
        }
    }
    if (c.Limits.Accounts * c.Sessions.PerAccount) > MaxRestorableSessions {
        return NewSettingCheck("sessions").Fail("per_account", fmt.Sprintf("at most %d sessions for limits.accounts accounts are expected", MaxRestorableSessions))
    }
    return nil
}

func (c ServerConfiguration) Clone() ServerConfiguration {
    clone := c
    clone.Server = c.Server.Clone()
    return clone
}

func (c ServerConfiguration) Keys() []string {
    names := []string{}
    for _, key := range c.keys() {
        names = append(names, key.name)
    }
    return names
}

func (c ServerConfiguration) Value(name string) (string, error) {
    for _, key := range c.keys() {
        if key.name == name {
            return key.get(), nil
        }
    }
    return "", fmt.Errorf("%w: %s", ErrUnknownSetting, name)
}

func (c *ServerConfiguration) Set(name string, value string) error {
    for _, key := range c.keys() {
        if key.name == name {
            return key.set(value)
        }
    }
    return fmt.Errorf("%w: %s", ErrUnknownSetting, name)
}

func (c *ServerConfiguration) keys() []serverConfigurationKey {
    server, database, sessions, login, passwords, limits, assistant := &c.Server, &c.Database, &c.Sessions, &c.Login, &c.Passwords, &c.Limits, &c.Assistant
    return []serverConfigurationKey{
        c.text("server.listen", &server.Listen),
        c.list("server.origins", &server.Origins),
        c.list("server.trusted_proxies", &server.TrustedProxies),
        c.text("server.timezone", &server.Timezone),
        c.text("server.log_level", &server.LogLevel),
        c.path("server.tls_cert", &server.TLSCert),
        c.path("server.tls_key", &server.TLSKey),
        c.text("server.tls_min_version", &server.TLSMinVersion),
        c.duration("server.hsts_max_age", &server.HSTSMaxAge),
        c.duration("server.read_header_timeout", &server.ReadHeaderTimeout),
        c.duration("server.read_timeout", &server.ReadTimeout),
        c.duration("server.write_timeout", &server.WriteTimeout),
        c.duration("server.keep_alive_timeout", &server.KeepAliveTimeout),
        c.duration("server.shutdown_timeout", &server.ShutdownTimeout),
        c.integer("server.max_header_bytes", &server.MaxHeaderBytes),
        c.integer("server.max_body_bytes", &server.MaxBodyBytes),
        c.integer("server.max_url_bytes", &server.MaxURLBytes),
        c.path("database.path", &database.Path),
        c.integer("database.max_bytes", &database.MaxBytes),
        c.duration("sessions.idle_timeout", &sessions.IdleTimeout),
        c.duration("sessions.max_lifetime", &sessions.MaxLifetime),
        c.integer("sessions.per_account", &sessions.PerAccount),
        c.duration("sessions.save_interval", &sessions.SaveInterval),
        c.duration("login.form_lifetime", &login.FormLifetime),
        c.duration("login.initial_delay", &login.InitialDelay),
        c.duration("login.max_delay", &login.MaxDelay),
        c.duration("login.budget_window", &login.BudgetWindow),
        c.integer("login.new_addresses_per_window", &login.NewAddressesPerWindow),
        c.integer("login.tracked_addresses", &login.TrackedAddresses),
        c.duration("login.forget_after", &login.ForgetAfter),
        c.integer("passwords.iterations", &passwords.Iterations),
        c.integer("limits.accounts", &limits.Accounts),
        c.integer("limits.account_bytes", &limits.AccountBytes),
        c.integer("limits.adjustments", &limits.Adjustments),
        c.duration("assistant.timeout", &assistant.Timeout),
        c.integer("assistant.max_rounds", &assistant.MaxRounds),
        c.integer("assistant.max_tokens", &assistant.MaxTokens),
        c.integer("assistant.conversations", &assistant.Conversations),
        c.integer("assistant.exchanges", &assistant.Exchanges),
        c.integer("assistant.question_bytes", &assistant.QuestionBytes),
        c.duration("assistant.download_ttl", &assistant.DownloadTTL),
        c.duration("assistant.download_sweep", &assistant.DownloadSweep),
    }
}

func (c *ServerConfiguration) text(name string, field *string) serverConfigurationKey {
    return serverConfigurationKey{name: name, get: func() string { return *field }, set: func(value string) error {
        *field = value
        return nil
    }}
}

func (c *ServerConfiguration) list(name string, field *[]string) serverConfigurationKey {
    return serverConfigurationKey{name: name, get: func() string { return strings.Join(*field, ",") }, set: func(value string) error {
        values := []string{}
        for _, element := range strings.Split(value, ",") {
            if element = strings.TrimSpace(element); element != "" {
                values = append(values, element)
            }
        }
        *field = values
        return nil
    }}
}

func (c *ServerConfiguration) path(name string, field *NullableText) serverConfigurationKey {
    return serverConfigurationKey{name: name, get: func() string {
        if *field == "" {
            return noValue
        }
        return string(*field)
    }, set: func(value string) error {
        if value == noValue {
            value = ""
        }
        *field = NullableText(value)
        return nil
    }}
}

func (c *ServerConfiguration) duration(name string, field *Duration) serverConfigurationKey {
    return serverConfigurationKey{name: name, get: func() string { return field.String() }, set: func(value string) error {
        parsed, err := ParseDuration(value)
        if err != nil {
            return fmt.Errorf("%w: %s: a duration such as 500ms, 30s, 5m or 8h is expected", ErrInvalidConfiguration, name)
        }
        *field = parsed
        return nil
    }}
}

func (c *ServerConfiguration) integer(name string, field *int64) serverConfigurationKey {
    return serverConfigurationKey{name: name, get: func() string { return strconv.FormatInt(*field, 10) }, set: func(value string) error {
        parsed, err := strconv.ParseInt(value, 10, 64)
        if err != nil {
            return fmt.Errorf("%w: %s: an integer is expected", ErrInvalidConfiguration, name)
        }
        *field = parsed
        return nil
    }}
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
