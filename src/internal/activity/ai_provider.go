/*
 * ai_provider.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "net/url"
    "slices"
    "strings"
)

// ---------------------------------------------------------------------------
// AIProvider declaration
// ---------------------------------------------------------------------------

type AIProvider struct {
    Name         string   `json:"name"`
    Label        string   `json:"label"`
    Format       AIFormat `json:"format"`
    BaseURL      string   `json:"base_url"`
    DefaultModel string   `json:"default_model"`
    Models       []string `json:"models"`
}

const (
    MaxAIProviders   = 20
    MaxAIModels      = 30
    MaxAIModelLength = 100
    maxProviderName  = 16
    maxProviderURL   = 200
)

// ---------------------------------------------------------------------------
// AIProvider implementation
// ---------------------------------------------------------------------------

func (p AIProvider) ValidName() bool {
    if (len(p.Name) < 1) || (len(p.Name) > maxProviderName) {
        return false
    }
    for _, letter := range []byte(p.Name) {
        if ((letter < 'a') || (letter > 'z')) && ((letter < '0') || (letter > '9')) && (letter != '-') {
            return false
        }
    }
    return true
}

func (p AIProvider) ValidURL() bool {
    parsed, err := url.Parse(p.BaseURL)
    return (err == nil) && (len(p.BaseURL) <= maxProviderURL) && (parsed.Scheme == "https") && (parsed.Host != "") && (parsed.User == nil) && (parsed.RawQuery == "") && (parsed.Fragment == "") && !strings.HasSuffix(p.BaseURL, "/")
}

func (p AIProvider) ValidModel(model string) bool {
    if (model == "") || (len(model) > MaxAIModelLength) {
        return false
    }
    for _, letter := range model {
        if (letter <= ' ') || (letter > '~') {
            return false
        }
    }
    return true
}

func (p AIProvider) Clone() AIProvider {
    clone := p
    clone.Models = slices.Clone(p.Models)
    return clone
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
