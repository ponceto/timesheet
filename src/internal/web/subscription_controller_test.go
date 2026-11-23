/*
 * subscription_controller_test.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "net/http"
    "net/url"
    "os"
    "regexp"
    "strings"
    "testing"
)

// ---------------------------------------------------------------------------
// SubscriptionController tests
// ---------------------------------------------------------------------------

func TestSubscriptionsAreSealedSharedAndWithdrawn(t *testing.T) {
    site := newTestSite(t)
    demo := site.client().login("demo")
    if strings.Contains(demo.page("/dashboard/"), `href="/dashboard/assistant"`) {
        t.Fatal("the assistant link needs a subscription")
    }
    demo.subscribe("anthropic|claude-sonnet-5-5", "sk-ant-very-secret")
    profile := demo.page("/dashboard/subscriptions")
    if strings.Contains(profile, "sk-ant-very-secret") || !strings.Contains(profile, "La souscription IA a été enregistrée.") || !strings.Contains(profile, "Anthropic (Claude) claude-sonnet-5-5") {
        t.Fatalf("the profile must list the subscription without its key:\n%s", profile)
    }
    if !strings.Contains(demo.page("/dashboard/"), `href="/dashboard/assistant"`) {
        t.Fatal("a subscription enables the assistant link")
    }
    data, _ := os.ReadFile(site.database.Path())
    if strings.Contains(string(data), "sk-ant-very-secret") {
        t.Fatal("the key must be sealed in the database")
    }
    if info, err := os.Stat(site.database.KeyPath()); (err != nil) || (info.Mode().Perm() != 0o600) {
        t.Fatalf("the key file must be private: %v %v", info, err)
    }
    id := regexp.MustCompile(`name="id" value="([0-9a-f]+)"`).FindStringSubmatch(profile)[1]
    status, _, page := demo.post("/dashboard/subscriptions", url.Values{"csrf": {demo.token(profile)}, "action": {"subscription-share"}, "id": {id}, "login": {"ghost"}})
    if (status != http.StatusUnprocessableEntity) || !strings.Contains(page, "Partage impossible") {
        t.Fatalf("an unknown account is refused: %d", status)
    }
    if status, _, _ := demo.post("/dashboard/subscriptions", url.Values{"csrf": {demo.token(profile)}, "action": {"subscription-share"}, "id": {id}, "login": {"root"}}); status != http.StatusSeeOther {
        t.Fatalf("share: %d", status)
    }
    root := site.client().login("root")
    shared := root.page("/dashboard/subscriptions")
    if !strings.Contains(shared, "Souscriptions partagées avec moi") || !strings.Contains(root.page("/dashboard/"), `href="/dashboard/assistant"`) || strings.Contains(shared, "sk-ant-very-secret") {
        t.Fatal("a shared subscription is offered to the other account, without its key")
    }
    if status, _, _ := demo.post("/dashboard/subscriptions", url.Values{"csrf": {demo.token(profile)}, "action": {"subscription-unshare"}, "id": {id}, "login": {"root"}}); status != http.StatusSeeOther {
        t.Fatalf("unshare: %d", status)
    }
    if strings.Contains(root.page("/dashboard/"), `href="/dashboard/assistant"`) {
        t.Fatal("withdrawing the share removes the assistant link")
    }
    if status, _, _ := root.post("/dashboard/subscriptions", url.Values{"csrf": {root.token(shared)}, "action": {"subscription-delete"}, "id": {id}}); status != http.StatusNotFound {
        t.Fatalf("only the owner changes a subscription: %d", status)
    }
    if status, _, _ := demo.post("/dashboard/subscriptions", url.Values{"action": {"subscription-delete"}, "id": {id}}); status != http.StatusForbidden {
        t.Fatalf("a form without token is refused: %d", status)
    }
}

func TestSubscriptionsPageSuggestsOnlyLogins(t *testing.T) {
    site := newTestSite(t)
    demo := site.client().login("demo")
    page := demo.page("/dashboard/subscriptions")
    if !strings.Contains(page, "<h1>Mes souscriptions</h1>") || !strings.Contains(page, `<a href="/dashboard/subscriptions" aria-current="page">Mes souscriptions</a>`) {
        t.Fatalf("got\n%s", page)
    }
    if strings.Contains(demo.page("/dashboard/profile"), "subscription-add") {
        t.Fatal("the subscriptions left the profile page")
    }
    status, headers, body := demo.download("/dashboard/subscriptions/logins?q=ro")
    if (status != http.StatusOK) || !strings.HasPrefix(headers.Get("Content-Type"), "application/json") || (body != `["root"]`) {
        t.Fatalf("got %d %v %s", status, headers, body)
    }
    for _, query := range []string{"r", "demo", "xyz"} {
        if _, _, body := demo.download("/dashboard/subscriptions/logins?q=" + query); body != `[]` {
            t.Fatalf("%s: got %s", query, body)
        }
    }
    if status, _, _ := site.client().get("/dashboard/subscriptions/logins?q=ro"); status != http.StatusSeeOther {
        t.Fatalf("a guest gets no login: %d", status)
    }
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
