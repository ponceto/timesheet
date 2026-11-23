/*
 * configuration.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "slices"
    "strings"
    "time"
)

// ---------------------------------------------------------------------------
// Configuration declaration
// ---------------------------------------------------------------------------

type Configuration struct {
    Revision       int64             `json:"revision"`
    PublicHolidays PublicHolidayZone `json:"public_holidays"`
    Activities     []ActivityCode    `json:"activities"`
    Counters       []CounterRule     `json:"counters"`
    AIProviders    []AIProvider      `json:"ai_providers"`
}

const (
    MaxActivities       = 100
    MaxCounters         = 20
    maxLabelLength      = 60
    maxShortLabel       = 6
    maxIdentifierLength = 16
)

var ErrInvalidConfiguration = errors.New("activity: invalid configuration")

// ---------------------------------------------------------------------------
// Configuration implementation
// ---------------------------------------------------------------------------

func NewDefaultConfiguration() Configuration {
    return Configuration{
        PublicHolidays: ZoneMetropole,
        Activities: []ActivityCode{
            {ID: "TT", Label: "Temps de travail", Feeds: "TT", Color: "#33d17a"},
            {ID: "WE", Label: "Week-end", Feeds: "WE", Color: "#9a9996"},
            {ID: "JF", Label: "Jour férié", Feeds: "JF", Color: "#ff7800"},
            {ID: "CP", Label: "Congé payé", Consumes: "CP", Color: "#e01b24"},
            {ID: "CSS", Label: "Congé sans solde", Feeds: "CSS", Color: "#e01b24"},
            {ID: "RTT", Label: "Récupération temps de travail", Consumes: "RTT", Color: "#e01b24"},
            {ID: "TP", Label: "Temps partiel", Feeds: "TP", Color: "#3584e4"},
            {ID: "TPC", Label: "Temps partiel complémentaire", Consumes: "TPC", Color: "#9141ac"},
            {ID: "ABS", Label: "Absence", Feeds: "ABS", Color: "#e01b24"},
        },
        Counters: []CounterRule{
            {ID: "TT", Label: "Temps de travail", ShortLabel: "TT", EnabledByDefault: true, Accrual: Accrual{Every: EveryYear, Amount: NewQuantity(0, 1)}, PeriodStart: NewPeriodStart(time.January, 1)},
            {ID: "WE", Label: "Week-ends", ShortLabel: "WE", EnabledByDefault: true, Accrual: Accrual{Every: EveryYear, Amount: NewQuantity(0, 1)}, PeriodStart: NewPeriodStart(time.January, 1)},
            {ID: "JF", Label: "Jours fériés", ShortLabel: "JF", EnabledByDefault: true, Accrual: Accrual{Every: EveryYear, Amount: NewQuantity(0, 1)}, PeriodStart: NewPeriodStart(time.January, 1)},
            {ID: "CP", Label: "Congés payés", ShortLabel: "CP", EnabledByDefault: true, Accrual: Accrual{Every: EveryMonth, Amount: NewQuantity(208, 100)}, PeriodStart: NewPeriodStart(time.June, 1), CarryOver: true},
            {ID: "CSS", Label: "Congés sans solde", ShortLabel: "CSS", EnabledByDefault: true, Accrual: Accrual{Every: EveryYear, Amount: NewQuantity(0, 1)}, PeriodStart: NewPeriodStart(time.January, 1)},
            {ID: "RTT", Label: "Récupérations temps de travail", ShortLabel: "RTT", EnabledByDefault: true, Accrual: Accrual{Every: EveryMonth, Amount: NewQuantity(8, 12)}, PeriodStart: NewPeriodStart(time.January, 1)},
            {ID: "TP", Label: "Temps partiels", ShortLabel: "TP", EnabledByDefault: true, Accrual: Accrual{Every: EveryYear, Amount: NewQuantity(0, 1)}, PeriodStart: NewPeriodStart(time.January, 1)},
            {ID: "TPC", Label: "Temps partiels complémentaires", ShortLabel: "TPC", Accrual: Accrual{Every: EveryMonth, Amount: NewQuantity(0, 1)}, PeriodStart: NewPeriodStart(time.January, 1)},
            {ID: "ABS", Label: "Absences", ShortLabel: "ABS", EnabledByDefault: true, Accrual: Accrual{Every: EveryYear, Amount: NewQuantity(0, 1)}, PeriodStart: NewPeriodStart(time.January, 1)},
        },
        AIProviders: NewDefaultAIProviders(),
    }
}

func NewDefaultAIProviders() []AIProvider {
    return []AIProvider{
        {Name: "mistral", Label: "Mistral", Format: FormatOpenAI, BaseURL: "https://api.mistral.ai/v1", DefaultModel: "mistral-small-latest", Models: []string{"mistral-large-latest", "mistral-medium-latest", "mistral-small-latest", "magistral-medium-latest", "open-mistral-nemo", "codestral-latest"}},
        {Name: "anthropic", Label: "Anthropic (Claude)", Format: FormatAnthropic, BaseURL: "https://api.anthropic.com/v1", DefaultModel: "claude-sonnet-5-5", Models: []string{"claude-opus-5-5", "claude-sonnet-5-5", "claude-haiku-5-5", "claude-sonnet-4-5", "claude-haiku-4-5"}},
        {Name: "openai", Label: "OpenAI", Format: FormatOpenAI, BaseURL: "https://api.openai.com/v1", DefaultModel: "gpt-4o-mini", Models: []string{"gpt-4.1", "gpt-4.1-mini", "gpt-4o", "gpt-4o-mini", "o4-mini", "o3"}},
        {Name: "google", Label: "Google Gemini", Format: FormatGemini, BaseURL: "https://generativelanguage.googleapis.com/v1beta", DefaultModel: "gemini-2.5-flash", Models: []string{"gemini-2.5-pro", "gemini-2.5-flash", "gemini-2.0-flash"}},
    }
}

func (c Configuration) Providers() []AIProvider {
    if len(c.AIProviders) == 0 {
        return NewDefaultAIProviders()
    }
    return c.AIProviders
}

func (c Configuration) Provider(name string) (AIProvider, bool) {
    for _, provider := range c.Providers() {
        if provider.Name == name {
            return provider, true
        }
    }
    return AIProvider{}, false
}

func (c Configuration) Activity(id string) (ActivityCode, bool) {
    for _, code := range c.Activities {
        if code.ID == id {
            return code, true
        }
    }
    return ActivityCode{}, false
}

func (c Configuration) Consumed(counter string) bool {
    for _, code := range c.Activities {
        if code.Consumes == counter {
            return true
        }
    }
    return false
}

func (c Configuration) PublicHolidayZone() PublicHolidayZone {
    if c.PublicHolidays == "" {
        return ZoneMetropole
    }
    return c.PublicHolidays
}

func (c Configuration) Counter(id string) (CounterRule, bool) {
    for _, rule := range c.Counters {
        if rule.ID == id {
            return rule, true
        }
    }
    return CounterRule{}, false
}

func (c Configuration) Validate() error {
    if !c.PublicHolidayZone().Valid() {
        return c.fail("public_holidays", "a known zone of public holidays expected")
    }
    if (len(c.Activities) == 0) || (len(c.Activities) > MaxActivities) {
        return c.fail("activities", "between 1 and %d activity codes expected", MaxActivities)
    }
    if len(c.Counters) > MaxCounters {
        return c.fail("counters", "at most %d counters expected", MaxCounters)
    }
    counters := map[string]CounterRule{}
    for i, rule := range c.Counters {
        path := fmt.Sprintf("counters[%d]", i)
        if err := c.validateCounter(path, rule); err != nil {
            return err
        }
        if _, taken := counters[rule.ID]; taken {
            return c.fail(path + ".id", "duplicate identifier %q", rule.ID)
        }
        counters[rule.ID] = rule
    }
    seen := map[string]bool{}
    for i, code := range c.Activities {
        path := fmt.Sprintf("activities[%d]", i)
        if err := c.validateActivity(path, code, counters); err != nil {
            return err
        }
        if seen[code.ID] {
            return c.fail(path + ".id", "duplicate identifier %q", code.ID)
        }
        seen[code.ID] = true
    }
    return c.validateProviders()
}

func (c Configuration) validateProviders() error {
    if len(c.AIProviders) > MaxAIProviders {
        return c.fail("ai_providers", "at most %d providers expected", MaxAIProviders)
    }
    names := map[string]bool{}
    for i, provider := range c.AIProviders {
        path := fmt.Sprintf("ai_providers[%d]", i)
        switch {
            case !provider.ValidName() || names[provider.Name]:
                return c.fail(path + ".name", "a unique name [a-z0-9-]{1,16} expected")
            case !c.validLabel(provider.Label, maxLabelLength):
                return c.fail(path + ".label", "1 to %d characters expected", maxLabelLength)
            case !provider.Format.Valid():
                return c.fail(path + ".format", "openai, anthropic or gemini expected")
            case !provider.ValidURL():
                return c.fail(path + ".base_url", "an https URL without query, fragment nor final slash expected")
            case !provider.ValidModel(provider.DefaultModel):
                return c.fail(path + ".default_model", "a model name of 1 to %d printable characters expected", MaxAIModelLength)
            case len(provider.Models) > MaxAIModels:
                return c.fail(path + ".models", "at most %d models expected", MaxAIModels)
        }
        for _, model := range provider.Models {
            if !provider.ValidModel(model) {
                return c.fail(path + ".models", "model names of 1 to %d printable characters expected", MaxAIModelLength)
            }
        }
        names[provider.Name] = true
    }
    return nil
}

func (c Configuration) validateCounter(path string, rule CounterRule) error {
    if !c.ValidIdentifier(rule.ID) {
        return c.fail(path + ".id", "identifier [A-Z][A-Z0-9_]{0,15} expected")
    }
    if !c.validLabel(rule.Label, maxLabelLength) {
        return c.fail(path + ".label", "1 to %d characters expected", maxLabelLength)
    }
    if !c.validLabel(rule.ShortLabel, maxShortLabel) {
        return c.fail(path + ".short_label", "1 to %d characters expected", maxShortLabel)
    }
    if !rule.Accrual.Every.Valid() {
        return c.fail(path + ".accrual.every", "month or year expected")
    }
    if (rule.Accrual.Amount.Sign() < 0) || (rule.Accrual.Amount.Cmp(NewQuantity(366, 1)) > 0) {
        return c.fail(path + ".accrual.amount", "an amount between 0 and 366 expected")
    }
    if !rule.PeriodStart.Valid() {
        return c.fail(path + ".period_start", "a date MM-DD expected")
    }
    return nil
}

func (c Configuration) validateActivity(path string, code ActivityCode, counters map[string]CounterRule) error {
    if !c.ValidIdentifier(code.ID) {
        return c.fail(path + ".id", "identifier [A-Z][A-Z0-9_]{0,15} expected")
    }
    if !c.validLabel(code.Label, maxLabelLength) {
        return c.fail(path + ".label", "1 to %d characters expected", maxLabelLength)
    }
    if code.Consuming() == code.Feeding() {
        return c.fail(path + ".consumes", "exactly one of consumes and feeds expected")
    }
    if _, known := counters[code.Counter()]; !known {
        return c.fail(path + "." + code.CounterField(), "an existing counter expected")
    }
    if !code.Color.Valid() {
        return c.fail(path + ".color", "a colour #rrggbb is expected")
    }
    return nil
}

func (c Configuration) ValidIdentifier(id string) bool {
    if (len(id) < 1) || (len(id) > maxIdentifierLength) || (id[0] < 'A') || (id[0] > 'Z') {
        return false
    }
    for _, letter := range []byte(id[1:]) {
        if ((letter < 'A') || (letter > 'Z')) && ((letter < '0') || (letter > '9')) && (letter != '_') {
            return false
        }
    }
    return true
}

func (c Configuration) validLabel(label string, max int) bool {
    return (strings.TrimSpace(label) == label) && (label != "") && NewTextPolicy().Accepts(label, max)
}

func (c Configuration) fail(path string, format string, args ...any) error {
    return fmt.Errorf("%w: %s: %s", ErrInvalidConfiguration, path, fmt.Sprintf(format, args...))
}

func (c Configuration) Clone() Configuration {
    clone := c
    clone.Activities = slices.Clone(c.Activities)
    clone.Counters = slices.Clone(c.Counters)
    if c.AIProviders != nil {
        clone.AIProviders = make([]AIProvider, 0, len(c.AIProviders))
        for _, provider := range c.AIProviders {
            clone.AIProviders = append(clone.AIProviders, provider.Clone())
        }
    }
    return clone
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
