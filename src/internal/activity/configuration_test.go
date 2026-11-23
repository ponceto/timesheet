/*
 * configuration_test.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "bytes"
    "encoding/json"
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "reflect"
    "strings"
    "testing"
    "time"
)

// ---------------------------------------------------------------------------
// Configuration tests
// ---------------------------------------------------------------------------

func TestDefaultConfigurationIsValid(t *testing.T) {
    if err := NewDefaultConfiguration().Validate(); err != nil {
        t.Fatal(err)
    }
}

func TestTemplateMirrorsTheDefaultConfiguration(t *testing.T) {
    data, err := os.ReadFile(filepath.Join("..", "..", "..", "share", "timesheet", "database-template.json"))
    if err != nil {
        t.Fatal(err)
    }
    template := Configuration{}
    decoder := json.NewDecoder(bytes.NewReader(data))
    decoder.DisallowUnknownFields()
    if err := decoder.Decode(&template); err != nil {
        t.Fatal(err)
    }
    if err := template.Validate(); err != nil {
        t.Fatal(err)
    }
    if !reflect.DeepEqual(template, NewDefaultConfiguration()) {
        t.Fatalf("share/timesheet/database-template.json differs from NewDefaultConfiguration:\n%+v\n%+v", template, NewDefaultConfiguration())
    }
}

func TestConfigurationValidationNamesTheField(t *testing.T) {
    cases := map[string]func(*Configuration){
        "activities[2].consumes": func(c *Configuration) { c.Activities[2].Consumes = "nope" },
        "activities[0].consumes": func(c *Configuration) { c.Activities[0].Consumes = "CP" },
        "activities[1].id":       func(c *Configuration) { c.Activities[1].ID = "TT" },
        "counters[0].accrual.amount": func(c *Configuration) { c.Counters[0].Accrual.Amount = NewQuantity(-1, 1) },
        "activities[0].color":    func(c *Configuration) { c.Activities[0].Color = "" },
        "activities[2].color":    func(c *Configuration) { c.Activities[2].Color = "#AFD095" },
        "counters[1].short_label": func(c *Configuration) { c.Counters[1].ShortLabel = "TOOLONG" },
        "activities[3].id":       func(c *Configuration) { c.Activities[3].ID = "rtt" },
        "activities[5].id":       func(c *Configuration) { c.Activities[5].ID = "JF" },
        "activities[8].consumes": func(c *Configuration) { c.Activities[8].Feeds = "" },
        "activities[0].feeds":    func(c *Configuration) { c.Activities[0].Feeds = "NOPE" },
        "counters[2].id":         func(c *Configuration) { c.Counters[2].ID = "TOO_LONG_IDENTIFIER" },
        "activities[4].label":    func(c *Configuration) { c.Activities[4].Label = "bad\u202elabel" },
        "public_holidays":        func(c *Configuration) { c.PublicHolidays = "corse" },
        "ai_providers[1].name":   func(c *Configuration) { c.AIProviders[1].Name = "mistral" },
        "ai_providers[0].format": func(c *Configuration) { c.AIProviders[0].Format = "other" },
        "ai_providers[2].base_url": func(c *Configuration) { c.AIProviders[2].BaseURL = "http://127.0.0.1:8080" },
        "ai_providers[3].base_url": func(c *Configuration) { c.AIProviders[3].BaseURL = "https://user@api.test/v1" },
        "ai_providers[0].default_model": func(c *Configuration) { c.AIProviders[0].DefaultModel = "bad model" },
        "ai_providers[1].models": func(c *Configuration) { c.AIProviders[1].Models = append(c.AIProviders[1].Models, "") },
    }
    for path, mutate := range cases {
        config := NewDefaultConfiguration().Clone()
        mutate(&config)
        err := config.Validate()
        if !errors.Is(err, ErrInvalidConfiguration) || !strings.Contains(err.Error(), path + ":") {
            t.Errorf("%s: got %v", path, err)
        }
    }
}

func TestConfigurationWithoutProvidersSuggestsTheDefaultOnes(t *testing.T) {
    config := NewDefaultConfiguration()
    config.AIProviders = nil
    if err := config.Validate(); err != nil {
        t.Fatal(err)
    }
    if provider, found := config.Provider("anthropic"); !found || (provider.Format != FormatAnthropic) || (len(config.Providers()) != 4) {
        t.Fatalf("got %+v %v", provider, found)
    }
    original := NewDefaultConfiguration()
    clone := original.Clone()
    clone.AIProviders[0].Models[0] = "changed"
    if original.AIProviders[0].Models[0] == "changed" {
        t.Fatal("the clone shares the models")
    }
}

func TestConfigurationWithoutPublicHolidaysFollowsTheMetropole(t *testing.T) {
    config := NewDefaultConfiguration()
    if config.PublicHolidayZone() != ZoneMetropole {
        t.Fatalf("default zone %s", config.PublicHolidayZone())
    }
    config.PublicHolidays = ""
    if err := config.Validate(); (err != nil) || (config.PublicHolidayZone() != ZoneMetropole) {
        t.Fatalf("absent zone: %v, %s", err, config.PublicHolidayZone())
    }
}

func TestEveryColourKeepsItsTextReadable(t *testing.T) {
    if horizon := NewBookkeeper(NewDefaultConfiguration(), NewCalendar(NewMonth(2099, time.January).First())).Horizon(NewDate(2099, time.December, 31)); horizon.Year() != MaxYear {
        t.Fatalf("horizon %s beyond the last year", horizon)
    }
    for red := 0; red < 256; red += 5 {
        for green := 0; green < 256; green += 5 {
            for blue := 0; blue < 256; blue += 5 {
                c := Color(fmt.Sprintf("#%02x%02x%02x", red, green, blue))
                if c.Contrast(c.Ink()) < 4.5 {
                    t.Fatalf("%s: contrast %.2f", c, c.Contrast(c.Ink()))
                }
            }
        }
    }
    for _, c := range []Color{"#7777dd", "#777777", "#757575", "#767676", "#ff0000", "#00ff00", "#0000ff"} {
        if c.Contrast(c.Ink()) < 4.5 {
            t.Errorf("%s: contrast %.2f", c, c.Contrast(c.Ink()))
        }
    }
    if Color("#AFD095").Valid() || Color("#afd09").Valid() || !Color("#afd095").Valid() {
        t.Fatal("colour validation")
    }
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
