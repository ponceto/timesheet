/*
 * secrets_test.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "net/url"
    "strings"
    "testing"
    "time"
)

// ---------------------------------------------------------------------------
// secrets tests
// ---------------------------------------------------------------------------

func TestSecretsNeverReachTheJournal(t *testing.T) {
    site, _ := newAssistantSite(t, openAIText("secret answer of the model"))
    guest := site.client()
    _, _, body := guest.get("/login")
    token := guest.token(body)
    guest.post("/login", url.Values{"csrf": {token}, "login": {"typed-secret-login"}, "password": {"typed secret password"}})
    site.clock.Advance(10 * time.Minute)
    user := site.client().login("demo")
    cookies := user.client.Jar.Cookies(&url.URL{Scheme: "http", Host: strings.TrimPrefix(site.server.URL, "http://")})
    user.subscribe("openai|gpt-4o-mini", "sk-typed-secret-key")
    user.ask("", "typed secret question")
    site.logs.waitFor("assistant answered")
    journal := site.logs.waitFor("user=demo ")
    secrets := []string{"typed-secret-login", "typed secret password", testPassword, token, "sk-typed-secret-key", "typed secret question", "secret answer of the model"}
    for _, cookie := range cookies {
        secrets = append(secrets, cookie.Value)
    }
    for _, secret := range secrets {
        if (secret != "") && strings.Contains(journal, secret) {
            t.Errorf("the journal contains %q", secret)
        }
    }
    if !strings.Contains(journal, "login failed") || !strings.Contains(journal, "login=demo") || !strings.Contains(journal, "assistant answered") {
        t.Fatalf("expected login events in the journal:\n%s", journal)
    }
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
