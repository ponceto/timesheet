/*
 * chrome_test.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "strings"
    "testing"
)

// ---------------------------------------------------------------------------
// chrome tests
// ---------------------------------------------------------------------------

func TestUserMenuOffersProfileSettingsAndLogout(t *testing.T) {
    site := newTestSite(t)
    user := site.client().login("demo")
    admin := site.client().login("root")
    _, _, page := user.get("/dashboard/")
    for _, expected := range []string{`<details class="user-menu">`, `<span class="avatar" aria-hidden="true">DE</span>`, `href="/dashboard/profile"`, `href="/dashboard/settings"`, `<form method="post" action="/logout">`, `<script type="module" src="/static/js/chrome_page.js">`} {
        if !strings.Contains(page, expected) {
            t.Errorf("user menu: missing %s", expected)
        }
    }
    if strings.Contains(page, `href="/admin/"`) {
        t.Error("a user must not see the administration link")
    }
    if _, _, page := admin.get("/dashboard/"); strings.Count(page, `<li><a href="/admin/">Administration</a></li>
                    <li><a href="/dashboard/" aria-current="page">Tableau de bord</a></li>`) != 2 {
        t.Error("an administrator must see the administration link left of the dashboard, in the header and in the burger menu")
    }
    if _, _, page := admin.get("/"); !strings.Contains(page, `<li><a href="/admin/">Administration</a></li>
                    <li><a href="/dashboard/">Tableau de bord</a></li>`) {
        t.Error("a connected administrator must see the administration link on the public pages")
    }
    for _, path := range []string{"/dashboard/", "/admin/settings"} {
        _, _, page := admin.get(path)
        menu := page[strings.Index(page, `<details class="user-menu">`):]
        menu = menu[:strings.Index(menu, "</details>")]
        if strings.Contains(menu, "Administration") || !strings.Contains(menu, `<p class="menu-heading">Administrateur</p>
                    <ul>
                        <li><a href="/dashboard/profile">Mon profil</a></li>
                        <li><a href="/dashboard/subscriptions">Mes souscriptions</a></li>
                        <li><a href="/dashboard/settings">Mes réglages</a></li>
                        <li><a href="/dashboard/data">Mes données</a></li>`) || !strings.Contains(menu, `<button type="submit">Se déconnecter</button>`) {
            t.Errorf("%s: the user menu must hold only the profile, « Mon profil », « Mes souscriptions », « Mes réglages », « Mes données » and the logout", path)
        }
    }
    if status, _, body := user.post("/logout", url.Values{"csrf": {user.token(page)}}); (status != http.StatusOK) || !strings.Contains(body, "Vous êtes déconnecté") || strings.Contains(body[strings.Index(body, "<main>"):strings.Index(body, "</main>")], "<a ") || strings.Contains(body[strings.Index(body, "<main>"):strings.Index(body, "</main>")], "<button") {
        t.Fatalf("logout from the menu: %d", status)
    }
    if status, location, _ := user.get("/dashboard/"); (status != http.StatusSeeOther) || (location != "/login") {
        t.Fatalf("the session must be closed: %d %s", status, location)
    }
}

func TestNarrowHeadersOfferABurgerMenu(t *testing.T) {
    site := newTestSite(t)
    admin := site.client().login("root")
    _, _, page := admin.get("/dashboard/")
    menu := `<details class="nav-menu">
                <summary><img src="/static/img/icons/menu.svg" alt="Menu" width="20" height="20"></summary>
                <nav aria-label="Navigation principale">`
    if !strings.Contains(page, menu) || (strings.Count(page, `<a href="/dashboard/" aria-current="page">Tableau de bord</a>`) != 2) {
        t.Error("the dashboard must repeat its navigation in the burger menu")
    }
    if _, _, page := admin.get("/admin/settings"); !strings.Contains(page, `<details class="nav-menu">`) || (strings.Count(page, `<a href="/admin/settings" aria-current="page">Configuration</a>`) != 2) {
        t.Error("the administration must repeat its navigation in the burger menu")
    }
    if _, _, page := admin.get("/admin/accounts"); strings.Count(page, `<ul>
                    <li><a href="/dashboard/">Tableau de bord</a></li>
                    <li><a href="/admin/">Serveur</a></li>`) != 2 {
        t.Error("the administration must start its navigation with the dashboard, in the header and in the burger menu")
    }
    if _, _, page := site.client().get("/"); strings.Contains(page, `class="nav-menu"`) {
        t.Error("the public pages need no burger menu")
    }
}

func TestEveryPageOffersAThemeToggle(t *testing.T) {
    site := newTestSite(t)
    guest := site.client()
    toggle := `<button type="button" class="theme-toggle" aria-label="Thème sombre" aria-pressed="false" hidden><img src="/static/img/icons/sun.svg" alt="" width="20" height="20"></button>`
    script := `<script type="module" src="/static/js/chrome_page.js">`
    if login := guest.page("/login"); strings.Contains(login[strings.Index(login, "<main>"):strings.Index(login, "</main>")], "<a ") {
        t.Error("the login page must not link back to the home page")
    }
    _, _, home := guest.get("/")
    if !strings.Contains(home, toggle) || !strings.Contains(home, script) {
        t.Error("the home page must offer the theme toggle")
    }
    if strings.Contains(home[strings.Index(home, "<main>"):], `href="/login"`) {
        t.Error("the title block must not link to the login page")
    }
    user := site.client().login("demo")
    admin := site.client().login("root")
    for _, page := range []string{user.page("/"), user.page("/dashboard/"), user.page("/dashboard/settings"), admin.page("/admin/")} {
        if !strings.Contains(page, toggle) || !strings.Contains(page, script) || strings.Contains(page, "data-theme=") || strings.Contains(page, "/dashboard/theme") || (!strings.Contains(page, toggle + "\n            <details class=\"nav-menu\">") && !strings.Contains(page, toggle + "\n        </header>")) {
            t.Error("a connected page must offer the theme toggle and keep no theme on the server")
        }
    }
}

func TestMenuShowsTheNameAndTheProfile(t *testing.T) {
    site := newTestSite(t)
    user := site.client().login("demo")
    page := user.page("/dashboard/profile")
    if !strings.Contains(page, `<strong>demo</strong>`) || !strings.Contains(page, `<p class="menu-heading">Utilisateur</p>`) {
        t.Fatal("without a name, the menu shows the login and the profile as its heading")
    }
    form := url.Values{"csrf": {user.token(page)}, "action": {"name"}, "name": {"  Marie   Curie "}}
    if status, location, _ := user.post("/dashboard/profile", form); (status != http.StatusSeeOther) || (location != "/dashboard/profile") {
        t.Fatalf("name: %d %s", status, location)
    }
    if page := user.page("/dashboard/"); !strings.Contains(page, `<strong>Marie Curie</strong>`) || !strings.Contains(page, `<span class="avatar" aria-hidden="true">MC</span>`) {
        t.Fatal("the menu must show the name and its initials")
    }
    if found, _, _ := site.database.Account("demo"); found.Name != "Marie Curie" {
        t.Fatalf("stored name: %q", found.Name)
    }
    for _, name := range []string{strings.Repeat("x", 61), "Marie\u202eCurie"} {
        form.Set("name", name)
        if status, _, _ := user.post("/dashboard/profile", form); status != http.StatusUnprocessableEntity {
            t.Errorf("%q: %d", name, status)
        }
    }
}

func TestInitialsComeFromTheFirstAndLastWords(t *testing.T) {
    renderer := &Renderer{}
    for label, expected := range map[string]string{"Olivier PONCET": "OP", "Jean Pierre de la Fontaine": "JF", "olivier.poncet": "OL", "demo": "DE", "Élise": "ÉL", "x": "X", "Marie - Curie": "MC", "": ""} {
        if initials := renderer.initials(label); initials != expected {
            t.Errorf("%q: %q, want %q", label, initials, expected)
        }
    }
}

func TestNoticesFollowTheActionOnceWithoutAnyParameter(t *testing.T) {
    site := newTestSite(t)
    user := site.client().login("demo")
    other := site.client().login("demo")
    page := user.page("/dashboard/profile")
    form := url.Values{"csrf": {user.token(page)}, "action": {"name"}, "name": {"Marie Curie"}}
    if status, location, _ := user.post("/dashboard/profile", form); (status != http.StatusSeeOther) || (location != "/dashboard/profile") {
        t.Fatalf("name: %d %s", status, location)
    }
    notice := `<p role="status">Le nom a été enregistré.</p>`
    if !strings.Contains(other.page("/dashboard/profile"), "Marie Curie") || strings.Contains(other.page("/dashboard/profile"), notice) {
        t.Fatal("the notice belongs to the session that acted")
    }
    if !strings.Contains(user.page("/dashboard/profile"), notice) {
        t.Fatal("the page that follows the action must show its notice")
    }
    if strings.Contains(user.page("/dashboard/profile"), notice) {
        t.Fatal("the notice must be shown once")
    }
    if strings.Contains(user.page("/dashboard/profile?notice=name"), notice) {
        t.Fatal("a notice must never come from the URL")
    }
    change := url.Values{"csrf": {user.token(page)}, "action": {"password"}, "current": {testPassword}, "password": {"another horse battery"}, "confirm": {"another horse battery"}}
    if status, location, _ := user.post("/dashboard/profile", change); (status != http.StatusSeeOther) || (location != "/dashboard/profile") {
        t.Fatalf("password: %d %s", status, location)
    }
    if !strings.Contains(user.page("/dashboard/profile"), `<p role="status">Le mot de passe a été changé.</p>`) {
        t.Fatal("the notice must follow the new session opened by a password change")
    }
}

func TestProfileClosesEverySession(t *testing.T) {
    site := newTestSite(t)
    first := site.client().login("demo")
    second := site.client().login("demo")
    page := first.page("/dashboard/profile")
    if !strings.Contains(page, `<button type="submit">Déconnecter toutes mes sessions</button>`) {
        t.Fatal("the profile must offer to close every session")
    }
    if status, _, body := first.post("/dashboard/profile", url.Values{"csrf": {first.token(page)}, "action": {"sessions"}}); (status != http.StatusOK) || !strings.Contains(body, "Vous êtes déconnecté") {
        t.Fatalf("close sessions: %d", status)
    }
    for _, client := range []*testClient{first, second} {
        if status, location, _ := client.get("/dashboard/"); (status != http.StatusSeeOther) || (location != "/login") {
            t.Fatalf("a closed session still opens the dashboard: %d %s", status, location)
        }
    }
}

func TestSettingsNoLongerChangeThePassword(t *testing.T) {
    site := newTestSite(t)
    user := site.client().login("demo")
    _, _, page := user.get("/dashboard/settings")
    if strings.Contains(page, `type="password"`) {
        t.Error("the settings page must not hold the password form")
    }
    if status, _, page := user.get("/dashboard/profile"); (status != http.StatusOK) || !strings.Contains(page, `name="current"`) || !strings.Contains(page, "<dd>demo</dd>") {
        t.Fatalf("profile page: %d", status)
    }
}

func TestThemeIsNotStoredOnTheServer(t *testing.T) {
    site := newTestSite(t)
    user := site.client().login("demo")
    token := user.token(user.page("/dashboard/"))
    if status, _, _ := user.post("/dashboard/theme", url.Values{"csrf": {token}, "theme": {"dark"}}); status != http.StatusNotFound {
        t.Fatalf("theme route: %d", status)
    }
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
