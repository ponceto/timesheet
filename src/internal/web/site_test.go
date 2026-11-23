/*
 * site_test.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "crypto/rand"
    "net/http"
    "net/http/httptest"
    "net/url"
    "regexp"
    "strings"
    "testing"
    "time"

    "timesheet/internal/account"
    "timesheet/internal/activity"
)

// ---------------------------------------------------------------------------
// site tests
// ---------------------------------------------------------------------------

func TestGuestIsSentToTheLoginPage(t *testing.T) {
    site := newTestSite(t)
    guest := site.client()
    if status, _, body := guest.get("/"); (status != http.StatusOK) || !strings.Contains(body, `href="/login"`) {
        t.Fatalf("home: %d", status)
    }
    for _, path := range []string{"/dashboard/", "/dashboard/month/2026/10", "/admin/", "/admin/settings", "/dashboard/settings"} {
        if status, location, _ := guest.get(path); (status != http.StatusSeeOther) || (location != "/login") {
            t.Errorf("%s: %d %s", path, status, location)
        }
    }
    if status, location, _ := guest.get("/logout"); (status != http.StatusSeeOther) || (location != "/login") {
        t.Fatalf("logout: %d %s", status, location)
    }
}

func TestLoginFailsWithTheSameMessage(t *testing.T) {
    site := newTestSite(t)
    guest := site.client()
    for _, login := range []string{"demo", "nobody"} {
        _, _, body := guest.get("/login")
        status, _, page := guest.post("/login", url.Values{"csrf": {guest.token(body)}, "login": {login}, "password": {"wrong password!"}})
        if (status != http.StatusUnauthorized) || !strings.Contains(page, "Identifiant ou mot de passe incorrect") {
            t.Fatalf("%s: %d", login, status)
        }
        site.clock.Advance(10 * time.Minute)
    }
    if status, _, _ := guest.post("/login", url.Values{"csrf": {"forged"}, "login": {"demo"}, "password": {testPassword}}); status != http.StatusForbidden {
        t.Fatalf("forged token: %d", status)
    }
}

func TestProfilesGuardTheAdministration(t *testing.T) {
    site := newTestSite(t)
    user := site.client().login("demo")
    if status, _, body := user.get("/admin/"); (status != http.StatusForbidden) || !strings.Contains(body, "réservée aux administrateurs") {
        t.Fatalf("user on /admin/: %d", status)
    }
    if status, _, _ := user.get("/admin/accounts"); status != http.StatusForbidden {
        t.Fatalf("user on /admin/accounts: %d", status)
    }
    if status, _, body := user.get("/dashboard/"); (status != http.StatusOK) || !strings.Contains(body, `Soldes au <time datetime="2026-11-08">08/11/2026</time>`) || !strings.Contains(body, "<h3>Total</h3>") || !strings.Contains(body, "<small>jour saisi</small>") || !strings.Contains(body, "<dt>Dont à venir</dt>") || strings.Contains(body, "<select") || strings.Contains(body, "Synthèse") {
        t.Fatalf("dashboard: %d", status)
    }
    if status, location, _ := user.get("/dashboard/month"); (status != http.StatusSeeOther) || (location != "/dashboard/month/2026/11") {
        t.Fatalf("current month: %d %s", status, location)
    }
    if status, _, body := user.get("/dashboard/month/2026/11"); (status != http.StatusOK) || !strings.Contains(body, "<h1>Novembre 2026</h1>") || !strings.Contains(body, "<h2>Temps de travail") || strings.Contains(body, "Ajustements de") || strings.Contains(body[strings.Index(body, "<main>"):], `<form method="post"`) || !strings.Contains(body, "<h2>Totaux du mois</h2>") || !strings.Contains(body, "<h2>Synthèse de Novembre 2026</h2>") || !strings.Contains(body, "<th scope=\"row\">Report du mois précédent</th>") || !strings.Contains(body, "<th scope=\"row\">Absence</th>\n                            <td>0</td>") || (strings.Index(body, "<h2>Totaux du mois</h2>") > strings.Index(body, "<section class=\"calendar\">")) || strings.Contains(body, "Soldes au") {
        t.Fatalf("month: %d", status)
    }
    if status, _, body := user.get("/dashboard/activity/2026/11"); (status != http.StatusOK) || !strings.Contains(body, "<h2>Ajustements de Novembre 2026</h2>") || !strings.Contains(body, "Ajouter un ajustement") {
        t.Fatalf("activity: %d", status)
    }
    if status, _, body := user.get("/dashboard/year/2026"); (status != http.StatusOK) || !strings.Contains(body, "<section class=\"summary\">") || !strings.Contains(body, `<a href="/dashboard/year/2026">Année en cours</a>`) || !strings.Contains(body, "<table class=\"recap\">") || (strings.Index(body, "<table class=\"totals\">") > strings.Index(body, "<section class=\"calendar\">")) {
        t.Fatalf("year: %d", status)
    }
    if status, _, _ := user.get("/dashboard/counters/2026"); status != http.StatusNotFound {
        t.Fatalf("counters page: %d", status)
    }
    admin := site.client().login("root")
    if status, _, body := admin.get("/admin/accounts/demo/dashboard"); (status != http.StatusOK) || !strings.Contains(body, "Soldes de demo") {
        t.Fatalf("read-only balances: %d", status)
    }
    status, _, body := admin.get("/admin/accounts/demo/month/2026/10")
    if (status != http.StatusOK) || strings.Contains(body, `<select name="am:`) || !strings.Contains(body, "Octobre 2026 de demo") {
        t.Fatalf("read-only view: %d", status)
    }
    if main := body[strings.Index(body, "<main>"):strings.Index(body, "</main>")]; strings.Contains(main, `<form method="post"`) {
        t.Fatal("a read-only view must not contain a post form")
    }
    _, _, list := admin.get("/admin/accounts")
    if status, _, _ := admin.post("/admin/accounts/demo/month/2026/10", url.Values{"csrf": {admin.token(list)}}); status != http.StatusMethodNotAllowed {
        t.Fatalf("write on a read-only view: %d", status)
    }
    rootCreated := admin.field(admin.page("/admin/accounts/root"), "created")
    for _, action := range []string{"delete", "reset", "profile"} {
        if status, _, _ := admin.post("/admin/accounts/root", url.Values{"csrf": {admin.token(list)}, "created": {rootCreated}, "action": {action}, "confirm": {"root"}, "profile": {"USER"}, "password": {"another strong one"}}); status != http.StatusUnprocessableEntity {
            t.Fatalf("self %s: %d", action, status)
        }
    }
    if status, _, page := user.get("/admin/"); (status != http.StatusForbidden) || strings.Contains(page, "Navigation de l'administration") {
        t.Fatalf("a user must not get the admin chrome: %d", status)
    }
    if _, _, page := user.get("/dashboard/nope"); !strings.Contains(page, "Se déconnecter") {
        t.Fatal("a 404 for a connected user must keep the dashboard chrome")
    }
}

func TestTrackingMayStartWithinAMonth(t *testing.T) {
    site := newTestSite(t)
    user := site.client().login("demo")
    _, _, page := user.get("/dashboard/settings")
    form := url.Values{"csrf": {user.token(page)}, "revision": {user.field(page, "revision")}, "configuration": {user.field(page, "configuration")}, "start": {"2026-09-15"}, "enabled:CP": {"on"}}
    for _, match := range regexp.MustCompile(`name="(color:[A-Za-z]+)" value="(#[0-9a-f]{6})"`).FindAllStringSubmatch(page, -1) {
        form.Set(match[1], match[2])
    }
    if status, _, _ := user.post("/dashboard/settings", form); status != http.StatusSeeOther {
        t.Fatalf("settings: %d", status)
    }
    _, _, body := user.get("/dashboard/activity/2026/09")
    if strings.Contains(body, `name="am:2026-09-14"`) || strings.Contains(body, `name="comment:2026-09-14"`) || !strings.Contains(body, `name="am:2026-09-15"`) {
        t.Fatal("only the days from the tracking start must be editable")
    }
    if !strings.Contains(body, `<td class="day-untracked" data-blank="day-untracked">Hors suivi</td>`) || !strings.Contains(body, `min="2026-09-15"`) {
        t.Fatal("the days before the tracking start must be shown out of the tracking and refused as adjustment dates")
    }
    overrides := map[string]string{}
    holidays := activity.NewPublicHolidayCalendar(activity.ZoneMetropole)
    for d := activity.NewDate(2026, time.September, 15); d.Month().Number() == 9; d = d.AddDays(1) {
        overrides["day:" + d.String()] = "TT"
        if !holidays.IsWorkingDay(d) {
            overrides["day:" + d.String()] = "WE"
        }
    }
    if status, _, _ := user.post("/dashboard/activity/2026/09", user.monthForm(body, overrides)); status != http.StatusSeeOther {
        t.Fatalf("save: %d", status)
    }
    _, _, body = user.get("/dashboard/activity/2026/09")
    validate := url.Values{"csrf": {user.token(body)}, "revision": {user.field(body, "revision")}}
    if status, location, _ := user.post("/dashboard/activity/2026/09/validate", validate); (status != http.StatusSeeOther) || (location != "/dashboard/activity/2026/09") {
        t.Fatalf("validate: %d %s", status, location)
    }
    if _, _, statements := user.get("/dashboard/month/2026/09"); strings.Contains(statements, "Acquisition septembre 2026") {
        t.Fatal("a partial start month must not accrue")
    }
}

func TestToolbarsStopAtTheBoundsOfTheCalendar(t *testing.T) {
    site := newTestSite(t)
    user := site.client().login("demo")
    for path, rel := range map[string]string{"/dashboard/month/2099/12": `rel="next"`, "/dashboard/month/2000/01": `rel="prev"`, "/dashboard/year/2099": `rel="next"`, "/dashboard/year/2000": `rel="prev"`} {
        if status, _, body := user.get(path); (status != http.StatusOK) || strings.Contains(body, rel) {
            t.Errorf("%s: %d, link %s", path, status, rel)
        }
    }
}

func TestMonthLifecycle(t *testing.T) {
    site := newTestSite(t)
    user := site.client().login("demo")
    _, _, body := user.get("/dashboard/activity/2026/09")
    overrides := map[string]string{}
    holidays := activity.NewPublicHolidayCalendar(activity.ZoneMetropole)
    for d := activity.NewDate(2026, time.September, 1); d.Month().Number() == 9; d = d.AddDays(1) {
        overrides["day:" + d.String()] = "TT"
        if !holidays.IsWorkingDay(d) {
            overrides["day:" + d.String()] = "WE"
        }
    }
    overrides["day:2026-09-15"] = "CP"
    overrides["comment:2026-09-15"] = "<script>alert(1)</script> & co"
    form := user.monthForm(body, overrides)
    if status, location, _ := user.post("/dashboard/activity/2026/09", form); (status != http.StatusSeeOther) || (location != "/dashboard/activity/2026/09") {
        t.Fatalf("save: %d %s", status, location)
    }
    if status, _, _ := user.post("/dashboard/activity/2026/09", form); status != http.StatusConflict {
        t.Fatalf("stale revision: %d", status)
    }
    _, _, body = user.get("/dashboard/activity/2026/09")
    if strings.Contains(body, "<script>alert") || !strings.Contains(body, "&lt;script&gt;alert(1)&lt;/script&gt; &amp; co") {
        t.Fatal("the comment is not escaped")
    }
    if !strings.Contains(body, `<option value="WE" selected>Week-end</option>`) || !strings.Contains(body, "<small>Week-end</small>") || !strings.Contains(body, "<small>Semaine</small>") || strings.Contains(body, "<small>Samedi</small>") {
        t.Fatal("a Saturday must be entered as WE and labelled Week-end, a working day Semaine")
    }
    _, _, november := user.get("/dashboard/activity/2026/11")
    if !strings.Contains(november, `<select name="am:2026-11-11" aria-label="Matin du 11 novembre"><option value="" selected>Aucun</option>`) || !strings.Contains(november, "<small>Armistice 1918</small>") || strings.Contains(november, "Jour férié : ") {
        t.Fatal("a holiday must stay to be entered and keep its name")
    }
    weekend := user.monthForm(body, map[string]string{"am:2026-09-12": "CP"})
    if status, _, page := user.post("/dashboard/activity/2026/09", weekend); (status != http.StatusUnprocessableEntity) || !strings.Contains(page, "ne peut pas être posé") {
        t.Fatalf("leave on a Saturday: %d", status)
    }
    validate := url.Values{"csrf": {user.token(body)}, "revision": {user.field(body, "revision")}}
    if status, location, _ := user.post("/dashboard/activity/2026/09/validate", validate); (status != http.StatusSeeOther) || (location != "/dashboard/activity/2026/09") {
        t.Fatalf("validate: %d %s", status, location)
    }
    _, _, body = user.get("/dashboard/activity/2026/09")
    if strings.Contains(body, `<select name="am:`) || !strings.Contains(body, "Rouvrir le mois") {
        t.Fatal("a validated month must be read-only and reopenable")
    }
    reopen := url.Values{"csrf": {user.token(body)}, "revision": {user.field(body, "revision")}, "reason": {"oubli"}}
    if status, location, _ := user.post("/dashboard/activity/2026/09/reopen", reopen); (status != http.StatusSeeOther) || (location != "/dashboard/activity/2026/09") {
        t.Fatalf("reopen: %d %s", status, location)
    }
    _, _, body = user.get("/dashboard/activity/2026/09")
    if _, _, statements := user.get("/dashboard/month/2026/09"); !strings.Contains(statements, "Écritures annulées par contre-passation") || !strings.Contains(body, `<select name="am:`) {
        t.Fatal("the reopened month must show its reversals and be editable")
    }
    if status, _, _ := user.post("/dashboard/activity/2026/11/validate", url.Values{"csrf": {user.token(body)}, "revision": {user.field(body, "revision")}}); status != http.StatusConflict {
        t.Fatalf("current month validated: %d", status)
    }
}

func TestAdministratorCreatesAnAccount(t *testing.T) {
    site := newTestSite(t)
    admin := site.client().login("root")
    _, _, body := admin.get("/admin/accounts")
    form := url.Values{"csrf": {admin.token(body)}, "login": {"marie"}, "profile": {"USER"}, "password": {"provisoire 2026!"}}
    if status, location, _ := admin.post("/admin/accounts", form); (status != http.StatusSeeOther) || (location != "/admin/accounts") {
        t.Fatalf("create: %d %s", status, location)
    }
    marie := site.client()
    _, _, page := marie.get("/login")
    marie.post("/login", url.Values{"csrf": {marie.token(page)}, "login": {"marie"}, "password": {"provisoire 2026!"}})
    if status, location, _ := marie.get("/dashboard/"); (status != http.StatusSeeOther) || !strings.HasPrefix(location, "/dashboard/profile") {
        t.Fatalf("first login must ask for a new password: %d %s", status, location)
    }
    _, _, page = marie.get("/dashboard/profile")
    if !strings.Contains(page, "Changer de mot de passe (obligatoire)") {
        t.Fatal("the profile page must ask for the password change")
    }
    if status, location, _ := marie.post("/dashboard/settings", url.Values{"csrf": {marie.token(page)}, "start": {"2020-01-01"}}); (status != http.StatusSeeOther) || !strings.HasPrefix(location, "/dashboard/profile") {
        t.Fatalf("settings before the password change: %d %s", status, location)
    }
    change := url.Values{"csrf": {marie.token(page)}, "action": {"password"}, "current": {"provisoire 2026!"}, "password": {"mon nouveau secret"}, "confirm": {"mon nouveau secret"}}
    if status, location, _ := marie.post("/dashboard/profile", change); (status != http.StatusSeeOther) || (location != "/dashboard/profile") {
        t.Fatalf("password change: %d %s", status, location)
    }
    if status, _, _ := marie.get("/dashboard/"); status != http.StatusOK {
        t.Fatalf("dashboard after the change: %d", status)
    }
    _, _, page = admin.get("/admin/accounts")
    if status, _, _ := admin.post("/admin/accounts/marie", url.Values{"csrf": {admin.token(page)}, "created": {admin.field(admin.page("/admin/accounts/marie"), "created")}, "action": {"profile"}, "profile": {"ADMIN"}}); status != http.StatusSeeOther {
        t.Fatalf("promote: %d", status)
    }
    if status, location, _ := marie.get("/dashboard/"); (status != http.StatusSeeOther) || (location != "/login") {
        t.Fatalf("sessions must close after a profile change: %d %s", status, location)
    }
}

func TestASessionOlderThanItsAccountIsRefused(t *testing.T) {
    site := newTestSite(t)
    user := site.client().login("demo")
    if err := site.database.DeleteAccount("demo"); err != nil {
        t.Fatal(err)
    }
    site.clock.Advance(time.Second)
    hash, _ := account.NewPasswordHash(testPassword, account.MinIterations, rand.Reader)
    if err := site.database.CreateAccount(account.Account{Login: "demo", Profile: account.ProfileUser, State: account.StateEnabled, Password: hash, Created: site.clock.Now()}, activity.NewCalendar(activity.NewMonth(2026, time.September).First())); err != nil {
        t.Fatal(err)
    }
    if status, location, _ := user.get("/dashboard/"); (status != http.StatusSeeOther) || (location != "/login") {
        t.Fatalf("a session of a deleted account reaches the account created again with its login: %d %s", status, location)
    }
}

func TestAccountDatesFollowTheServerTimeZone(t *testing.T) {
    site := newTestSite(t)
    site.clock.Advance((13 * time.Hour) + (30 * time.Minute))
    admin := site.client().login("root")
    list := admin.page("/admin/accounts")
    form := url.Values{"csrf": {admin.token(list)}, "login": {"carol"}, "profile": {"USER"}, "password": {"provisional horse 2026"}}
    if status, _, _ := admin.post("/admin/accounts", form); status != http.StatusSeeOther {
        t.Fatalf("create: %d", status)
    }
    if page := admin.page("/admin/accounts/carol"); !strings.Contains(page, `<dd><time datetime="2026-11-09">09/11/2026</time></dd>`) {
        t.Fatal("the creation at 00:30 in Paris must be dated 09/11/2026")
    }
    carol := site.client()
    _, _, body := carol.get("/login")
    carol.post("/login", url.Values{"csrf": {carol.token(body)}, "login": {"carol"}, "password": {"provisional horse 2026"}})
    if status, location, _ := carol.get("/dashboard/"); (status != http.StatusSeeOther) || (location != "/dashboard/profile") {
        t.Fatalf("provisional password: %d %s", status, location)
    }
    if page := carol.page("/dashboard/profile"); !strings.Contains(page, `<p role="status" class="is-warning">Choisissez un nouveau mot de passe avant de continuer.</p>`) {
        t.Fatal("the request of a new password must be a warning")
    }
}

func TestAdministratorDisablesAndDeletesAnAccount(t *testing.T) {
    site := newTestSite(t)
    user := site.client().login("demo")
    admin := site.client().login("root")
    list := admin.page("/admin/accounts")
    for _, expected := range []string{`<th scope="col">Nom</th>`, `<th scope="col">État</th>`, `aria-label="Désactiver demo" title="Désactiver"`, `aria-label="Supprimer demo" title="Supprimer"`} {
        if !strings.Contains(list, expected) {
            t.Errorf("accounts list without %s", expected)
        }
    }
    if !strings.Contains(list, `aria-label="Désactiver root" title="Désactiver" disabled>`) || !strings.Contains(list, `<button type="button" class="icon-button" aria-label="Supprimer root" title="Supprimer" disabled>`) || strings.Contains(list, `aria-label="Fermer les sessions de root" title="Fermer les sessions" disabled>`) {
        t.Error("the own account of the administrator must show its actions, disabled except the closing of its sessions")
    }
    demo := admin.field(admin.page("/admin/accounts/demo"), "created")
    root := admin.field(admin.page("/admin/accounts/root"), "created")
    form := url.Values{"csrf": {admin.token(list)}, "action": {"name"}, "name": {"  Jean   Démo "}}
    if status, _, body := admin.post("/admin/accounts/demo", form); (status != http.StatusConflict) || !strings.Contains(body, "supprimé puis recréé") {
        t.Fatalf("a form without the identity of the account: %d", status)
    }
    form.Set("created", demo)
    if status, location, _ := admin.post("/admin/accounts/demo", form); (status != http.StatusSeeOther) || (location != "/admin/accounts/demo") {
        t.Fatalf("name: %d %s", status, location)
    }
    if page := admin.page("/admin/accounts/demo"); !strings.Contains(page, `name="name" value="Jean Démo"`) || !strings.Contains(admin.page("/admin/accounts"), `<td>Jean Démo</td>`) {
        t.Fatal("the name must be saved with its spaces collapsed")
    }
    form = url.Values{"csrf": {admin.token(list)}, "created": {demo}, "action": {"name"}, "name": {strings.Repeat("x", 61)}}
    if status, _, _ := admin.post("/admin/accounts/demo", form); status != http.StatusUnprocessableEntity {
        t.Fatalf("long name: %d", status)
    }
    form = url.Values{"csrf": {admin.token(list)}, "created": {demo}, "action": {"sessions"}, "return": {"list"}}
    if !strings.Contains(list, `aria-label="Fermer les sessions de demo" title="Fermer les sessions">`) {
        t.Error("accounts list without the closing of the sessions of demo")
    }
    if status, location, _ := admin.post("/admin/accounts/demo", form); (status != http.StatusSeeOther) || (location != "/admin/accounts") {
        t.Fatalf("close sessions: %d %s", status, location)
    }
    if status, location, _ := user.get("/dashboard/"); (status != http.StatusSeeOther) || (location != "/login") {
        t.Fatalf("a closed session stays open: %d %s", status, location)
    }
    user = site.client().login("demo")
    form = url.Values{"csrf": {admin.token(list)}, "created": {demo}, "action": {"disable"}, "return": {"list"}}
    if status, location, _ := admin.post("/admin/accounts/demo", form); (status != http.StatusSeeOther) || (location != "/admin/accounts") {
        t.Fatalf("disable: %d %s", status, location)
    }
    if status, location, _ := user.get("/dashboard/"); (status != http.StatusSeeOther) || (location != "/login") {
        t.Fatalf("a disabled account keeps its session: %d %s", status, location)
    }
    guest := site.client()
    _, _, body := guest.get("/login")
    if status, _, _ := guest.post("/login", url.Values{"csrf": {guest.token(body)}, "login": {"demo"}, "password": {testPassword}}); status != http.StatusUnauthorized {
        t.Fatalf("a disabled account logs in: %d", status)
    }
    if page := admin.page("/admin/accounts"); !strings.Contains(page, `<td>désactivé</td>`) || !strings.Contains(page, `aria-label="Activer demo" title="Activer"`) {
        t.Fatal("the list must show the disabled account")
    }
    form.Set("action", "enable")
    if status, location, _ := admin.post("/admin/accounts/demo", form); (status != http.StatusSeeOther) || (location != "/admin/accounts") {
        t.Fatalf("enable: %d %s", status, location)
    }
    site.clock.Advance(10 * time.Minute)
    site.client().login("demo")
    form = url.Values{"csrf": {admin.token(list)}, "created": {root}, "action": {"disable"}}
    if status, _, _ := admin.post("/admin/accounts/root", form); status != http.StatusUnprocessableEntity {
        t.Fatalf("self disable: %d", status)
    }
    form = url.Values{"csrf": {admin.token(list)}, "created": {demo}, "action": {"delete"}, "confirm": {"demox"}, "return": {"list"}}
    if status, _, body := admin.post("/admin/accounts/demo", form); (status != http.StatusUnprocessableEntity) || !strings.Contains(body, "<h1>Comptes</h1>") {
        t.Fatalf("a failed deletion from the list must stay on the list: %d", status)
    }
    form = url.Values{"csrf": {admin.token(list)}, "created": {demo}, "action": {"delete"}, "confirm": {"demo"}}
    if status, location, _ := admin.post("/admin/accounts/demo", form); (status != http.StatusSeeOther) || (location != "/admin/accounts") {
        t.Fatalf("delete from the list: %d %s", status, location)
    }
    form = url.Values{"csrf": {admin.token(list)}, "created": {root}, "action": {"sessions"}, "return": {"list"}}
    if status, _, body := admin.post("/admin/accounts/root", form); (status != http.StatusOK) || !strings.Contains(body, "Vous êtes déconnecté") {
        t.Fatalf("close one's own sessions: %d", status)
    }
    if status, location, _ := admin.get("/admin/"); (status != http.StatusSeeOther) || (location != "/login") {
        t.Fatalf("the own session stays open: %d %s", status, location)
    }
}

func TestAdministratorChangesTheConfiguration(t *testing.T) {
    site := newTestSite(t)
    admin := site.client().login("root")
    _, _, body := admin.get("/admin/settings")
    add := url.Values{"csrf": {admin.token(body)}, "revision": {admin.field(body, "revision")}, "action": {"activity-add"}, "id": {"TRAINING"}, "label": {"Formation"}, "group": {"leave"}, "effect": {"feeds"}, "counter": {"ABS"}, "color": {"#c8b6e2"}}
    if status, location, _ := admin.post("/admin/settings", add); (status != http.StatusSeeOther) || (location != "/admin/settings") {
        t.Fatalf("add: %d %s", status, location)
    }
    add.Set("id", "COACHING")
    if status, _, _ := admin.post("/admin/settings", add); status != http.StatusConflict {
        t.Fatalf("stale configuration: %d", status)
    }
    _, _, body = admin.get("/admin/settings")
    for id, message := range map[string]string{"JF": "Cet identifiant de code existe déjà", "lower": "Configuration refusée : identifiant invalide"} {
        attempt := url.Values{"csrf": {admin.token(body)}, "revision": {admin.field(body, "revision")}, "action": {"activity-add"}, "id": {id}, "label": {"Essai"}, "group": {"leave"}, "effect": {"feeds"}, "counter": {"ABS"}, "color": {"#c8b6e2"}}
        if status, _, page := admin.post("/admin/settings", attempt); (status != http.StatusUnprocessableEntity) || !strings.Contains(page, message) {
            t.Errorf("code %s: %d", id, status)
        }
    }
    noEffect := url.Values{"csrf": {admin.token(body)}, "revision": {admin.field(body, "revision")}, "action": {"activity-add"}, "id": {"NONE"}, "label": {"Essai"}, "group": {"leave"}, "counter": {"ABS"}}
    if status, _, page := admin.post("/admin/settings", noEffect); (status != http.StatusUnprocessableEntity) || !strings.Contains(page, "alimente ou décompte") {
        t.Errorf("code without effect: %d", status)
    }
    user := site.client().login("demo")
    if _, _, page := user.get("/dashboard/activity/2026/11"); !strings.Contains(page, `<option value="TRAINING">Formation</option>`) {
        t.Fatal("the new code is not offered")
    }
}

func TestPublicHolidaysFollowTheZoneOfTheAccount(t *testing.T) {
    site := newTestSite(t)
    admin := site.client().login("root")
    user := site.client().login("demo")
    if page := user.page("/dashboard/activity/2026/04"); strings.Contains(page, "Vendredi saint") {
        t.Fatal("Good Friday is not a public holiday of the metropole")
    }
    body := admin.page("/admin/settings")
    zone := url.Values{"csrf": {admin.token(body)}, "revision": {admin.field(body, "revision")}, "action": {"public-holidays-update"}, "public_holidays": {"corse"}}
    if status, _, page := admin.post("/admin/settings", zone); (status != http.StatusUnprocessableEntity) || !strings.Contains(page, "Calendrier des jours fériés inconnu.") {
        t.Fatalf("unknown default zone: %d", status)
    }
    zone.Set("public_holidays", "fr-alsace-moselle")
    if status, location, _ := admin.post("/admin/settings", zone); (status != http.StatusSeeOther) || (location != "/admin/settings") {
        t.Fatalf("default zone: %d %s", status, location)
    }
    if page := admin.page("/admin/settings"); !strings.Contains(page, `<option value="fr-alsace-moselle" selected>Alsace-Moselle</option>`) {
        t.Fatal("the default zone is not shown")
    }
    if page := user.page("/dashboard/activity/2026/04"); !strings.Contains(page, "Vendredi saint") {
        t.Fatal("an account without its own zone follows the default zone")
    }
    page := user.page("/dashboard/settings")
    if !strings.Contains(page, `<option value="" selected>Par défaut (Alsace-Moselle)</option>`) {
        t.Fatal("the settings page names the default zone")
    }
    form := url.Values{"csrf": {user.token(page)}, "revision": {user.field(page, "revision")}, "configuration": {user.field(page, "configuration")}, "start": {user.field(page, "start")}, "enabled:CP": {"on"}, "public_holidays": {"corse"}}
    for _, match := range regexp.MustCompile(`name="(color:[A-Za-z]+)" value="(#[0-9a-f]{6})"`).FindAllStringSubmatch(page, -1) {
        form.Set(match[1], match[2])
    }
    if status, _, refused := user.post("/dashboard/settings", form); (status != http.StatusUnprocessableEntity) || !strings.Contains(refused, "Calendrier des jours fériés invalide.") {
        t.Fatalf("unknown own zone: %d", status)
    }
    start := form.Get("start")
    form.Set("public_holidays", "fr-guyane")
    form.Set("start", "not a date")
    if status, _, refused := user.post("/dashboard/settings", form); (status != http.StatusUnprocessableEntity) || !strings.Contains(refused, `<option value="fr-guyane" selected>Guyane</option>`) {
        t.Fatalf("a refused form must keep the other values posted: %d", status)
    }
    form.Set("start", start)
    form.Set("public_holidays", "fr-metropole")
    if status, _, _ := user.post("/dashboard/settings", form); status != http.StatusSeeOther {
        t.Fatalf("own zone: %d", status)
    }
    form.Set("public_holidays", "")
    form.Set("start", "not a date")
    if status, _, _ := user.post("/dashboard/settings", form); status != http.StatusConflict {
        t.Fatalf("stale settings form with an invalid value: %d", status)
    }
    form.Set("start", start)
    if page := user.page("/dashboard/activity/2026/04"); strings.Contains(page, "Vendredi saint") {
        t.Fatal("the own zone of the account prevails over the default zone")
    }
    if page := user.page("/dashboard/settings"); !strings.Contains(page, `<option value="fr-metropole" selected>Métropole</option>`) {
        t.Fatal("the own zone is not shown")
    }
}

func TestHostileInputsAreRefused(t *testing.T) {
    site := newTestSite(t)
    user := site.client().login("demo")
    _, _, body := user.get("/dashboard/activity/2026/10")
    hostile := map[string]string{
        "comment:2026-10-05": "spoof \u202e txt",
        "amount":             "-9223372036854775808/1",
    }
    form := user.monthForm(body, map[string]string{"comment:2026-10-05": "abc\u202edef"})
    form.Set("comment:2026-10-05", "abc\u202edef")
    if status, _, _ := user.post("/dashboard/activity/2026/10", form); status != http.StatusUnprocessableEntity {
        t.Fatalf("bidi comment: %d", status)
    }
    for _, amount := range []string{hostile["amount"], "1/999999", "367", "0"} {
        adjust := url.Values{"csrf": {user.token(body)}, "revision": {user.field(body, "revision")}, "action": {"add"}, "date": {"2026-10-05"}, "counter": {"CP"}, "amount": {amount}, "reason": {"test"}}
        if status, _, _ := user.post("/dashboard/adjustments", adjust); status != http.StatusUnprocessableEntity {
            t.Fatalf("amount %s: %d", amount, status)
        }
    }
    if status, _, _ := user.post("/dashboard/activity/2029/01", user.monthForm(body, nil)); status != http.StatusConflict {
        t.Fatalf("month beyond the planning horizon: %d", status)
    }
}

func TestSecurityHeadersAndErrors(t *testing.T) {
    site := newTestSite(t)
    response, err := http.Get(site.server.URL + "/")
    if err != nil {
        t.Fatal(err)
    }
    response.Body.Close()
    if csp := response.Header.Get("Content-Security-Policy"); !strings.Contains(csp, "require-trusted-types-for 'script'") || strings.Contains(csp, "unsafe-inline") {
        t.Fatalf("csp: %s", csp)
    }
    recorder := httptest.NewRecorder()
    NewSecurityHeaders(http.NotFoundHandler(), true, 48 * time.Hour).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
    if hsts := recorder.Header().Get("Strict-Transport-Security"); hsts != "max-age=172800" {
        t.Fatalf("HSTS follows its setting with https origins: %q", hsts)
    }
    guest := site.client()
    if status, _, body := guest.get("/nope"); (status != http.StatusNotFound) || !strings.Contains(body, "Page introuvable") {
        t.Fatalf("404: %d", status)
    }
    request, _ := http.NewRequest(http.MethodPost, site.server.URL + "/login", strings.NewReader("login=demo"))
    request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    response, err = http.DefaultClient.Do(request)
    if err != nil {
        t.Fatal(err)
    }
    response.Body.Close()
    if response.StatusCode != http.StatusForbidden {
        t.Fatalf("post without origin: %d", response.StatusCode)
    }
}

func TestStaleOrPartialFormsAreRefused(t *testing.T) {
    site := newTestSite(t)
    user := site.client().login("demo")
    admin := site.client().login("root")
    _, _, page := user.get("/dashboard/settings")
    form := url.Values{"csrf": {user.token(page)}, "revision": {user.field(page, "revision")}, "configuration": {user.field(page, "configuration")}, "start": {"2026-09-01"}}
    for _, match := range regexp.MustCompile(`name="(color:[A-Za-z]+)" value="(#[0-9a-f]{6})"`).FindAllStringSubmatch(page, -1) {
        form.Set(match[1], match[2])
    }
    _, _, body := admin.get("/admin/settings")
    add := url.Values{"csrf": {admin.token(body)}, "revision": {admin.field(body, "revision")}, "action": {"counter-add"}, "id": {"NEW"}, "label": {"Nouveau"}, "short": {"NEW"}, "enabled": {"on"}, "amount": {"1"}, "every": {"month"}, "period": {"01-01"}}
    if status, _, _ := admin.post("/admin/settings", add); status != http.StatusSeeOther {
        t.Fatalf("counter-add: %d", status)
    }
    if status, _, _ := user.post("/dashboard/settings", form); status != http.StatusConflict {
        t.Fatalf("settings form older than the configuration: %d", status)
    }
    if _, calendar, _ := site.database.Account("demo"); len(calendar.Settings.Counters) != 0 {
        t.Fatal("the stale settings form was saved")
    }
    _, _, body = user.get("/dashboard/activity/2026/10")
    whole := user.monthForm(body, map[string]string{"comment:2026-10-05": "kept"})
    if status, _, _ := user.post("/dashboard/activity/2026/10", whole); status != http.StatusSeeOther {
        t.Fatalf("save: %d", status)
    }
    _, _, body = user.get("/dashboard/activity/2026/10")
    partial := url.Values{"csrf": {user.token(body)}, "revision": {user.field(body, "revision")}}
    if status, _, _ := user.post("/dashboard/activity/2026/10", partial); status != http.StatusBadRequest {
        t.Fatalf("grid without its fields: %d", status)
    }
    if _, calendar, _ := site.database.Account("demo"); len(calendar.Days) != 1 {
        t.Fatal("a partial grid emptied the month")
    }
    current := user.monthForm(user.page("/dashboard/activity/2026/10"), map[string]string{})
    current.Set("revision", "+" + current.Get("revision"))
    if status, _, _ := user.post("/dashboard/activity/2026/10", current); status != http.StatusSeeOther {
        t.Fatalf("current revision written with a sign: %d", status)
    }
    whole.Set("am:2026-10-10", "CP")
    whole.Set("comment:2026-10-05", "")
    if status, _, _ := user.post("/dashboard/activity/2026/10", whole); status != http.StatusConflict {
        t.Fatalf("stale grid with an invalid entry: %d", status)
    }
    if _, calendar, _ := site.database.Account("demo"); calendar.Days[activity.NewDate(2026, time.October, 5)].Comment != "kept" {
        t.Fatal("a stale grid overwrote the saved entry")
    }
    staleAdjustment := url.Values{"csrf": {user.token(body)}, "revision": {whole.Get("revision")}, "action": {"add"}, "date": {"2026-10-05"}, "counter": {"CP"}, "amount": {"zero"}, "reason": {"test"}}
    if status, _, _ := user.post("/dashboard/adjustments", staleAdjustment); status != http.StatusConflict {
        t.Fatalf("stale adjustment with an invalid amount: %d", status)
    }
    staleAdjustment.Set("revision", "abc")
    if status, _, _ := user.post("/dashboard/adjustments", staleAdjustment); status != http.StatusBadRequest {
        t.Fatalf("unreadable revision of an adjustment: %d", status)
    }
    adjust := url.Values{"csrf": {user.token(body)}, "revision": {user.field(body, "revision")}, "date": {"2026-10-05"}, "counter": {"CP"}, "amount": {"1"}, "reason": {"test"}}
    if status, _, _ := user.post("/dashboard/adjustments", adjust); status != http.StatusBadRequest {
        t.Fatalf("adjustment without action: %d", status)
    }
    _, _, page = user.get("/dashboard/profile")
    if status, _, _ := user.post("/dashboard/profile", url.Values{"csrf": {user.token(page)}, "action": {"bogus"}}); status != http.StatusBadRequest {
        t.Fatalf("profile with an unknown action: %d", status)
    }
}

func TestNegativeBalancesAreMarkedInTheStatements(t *testing.T) {
    site := newTestSite(t)
    user := site.client().login("demo")
    _, _, body := user.get("/dashboard/activity/2026/10")
    adjust := url.Values{"csrf": {user.token(body)}, "revision": {user.field(body, "revision")}, "action": {"add"}, "date": {"2026-10-05"}, "counter": {"CSS"}, "amount": {"-3"}, "reason": {"test"}}
    if status, _, _ := user.post("/dashboard/adjustments", adjust); status != http.StatusSeeOther {
        t.Fatalf("adjust: %d", status)
    }
    if _, _, page := user.get("/dashboard/month/2026/10"); strings.Count(page, `<td class="is-negative">-3,00</td>`) < 2 {
        t.Fatal("the statement of the month does not mark its negative balances")
    }
    if _, _, page := user.get("/dashboard/year/2026"); strings.Count(page, `<td class="is-negative">-3,00</td>`) < 3 {
        t.Fatal("the recap and the ledger of the year do not mark their negative balances")
    }
    if page := user.page("/dashboard/"); !strings.Contains(page, `<dd class="is-negative">-3,00</dd>`) {
        t.Fatal("the dashboard does not mark a negative forecast")
    }
}

func TestMonthPathsHaveFourDigitYearsAndTwoDigitMonths(t *testing.T) {
    site := newTestSite(t)
    user := site.client().login("demo")
    for path, want := range map[string]int{"/dashboard/month/2026/09": http.StatusOK, "/dashboard/activity/2026/01": http.StatusOK, "/dashboard/month/2026/9": http.StatusNotFound, "/dashboard/activity/2026/1": http.StatusNotFound, "/dashboard/month/2026/009": http.StatusNotFound, "/dashboard/month/2026/+9": http.StatusNotFound, "/dashboard/month/2026/00": http.StatusNotFound, "/dashboard/month/2026/13": http.StatusNotFound} {
        if status, _, _ := user.get(path); status != want {
            t.Errorf("%s: %d, want %d", path, status, want)
        }
    }
    _, _, page := user.get("/dashboard/year/2026")
    if !strings.Contains(page, `href="/dashboard/month/2026/01"`) || !strings.Contains(page, `href="/dashboard/month/2026/09"`) || strings.Contains(page, `href="/dashboard/month/2026/9"`) {
        t.Error("the links of the year page must write the month on two digits")
    }
}

func TestVersionIsShownOnlyOnTheServerPage(t *testing.T) {
    site := newTestSite(t)
    admin := site.client().login("root")
    for _, path := range []string{"/", "/dashboard/", "/dashboard/profile", "/admin/accounts", "/admin/settings"} {
        if _, _, page := admin.get(path); strings.Contains(page, "<dd><code>test</code></dd>") || strings.Contains(page, "version test") {
            t.Errorf("%s shows the version", path)
        }
    }
    if _, _, page := site.client().get("/"); strings.Contains(page, "version test") {
        t.Error("the public home page shows the version")
    }
    if _, _, page := admin.get("/admin/"); strings.Count(page, "test</code>") != 1 {
        t.Error("the server page must show the version once, in the section Logiciel")
    }
}

func TestServerPageShowsEveryServerSetting(t *testing.T) {
    site := newTestSite(t)
    admin := site.client().login("root")
    _, _, body := admin.get("/admin/")
    for _, text := range []string{"<h2>Logiciel</h2>", "<dt>Nom</dt>", "<dd><code>Timesheet</code></dd>", "<dt>Version</dt>", "<dd><code>test</code></dd>", "<dt>Licence</dt>", "<dd><code>GNU GPL version 2 ou ultérieure</code></dd>", "<h1>Serveur</h1>", `<a href="/admin/" aria-current="page">Serveur</a>`, "<h2>Sessions</h2>", "<dt>Origines publiques</dt>", "<dd><code>http://127.0.0.1:8080</code></dd>\n                    <dd><code>http://localhost:8080</code></dd>", "<dd><code>10.0.0.0/8</code></dd>\n                    <dd><code>172.16.0.0/12</code></dd>", "<dt>Expiration sans activité</dt>", "<dd><code>8 h</code></dd>", "<dd><code>1 min</code></dd>", "<dt>Taille maximale</dt>", "<dd><code>64 Mio</code></dd>", "<dd><code>256 Kio</code></dd>", "<dd><code>2048 octets</code></dd>", "<dt>Certificat TLS</dt>", "<dd><code>aucun</code></dd>", "<dd><code>/srv/timesheet/timesheet-database.json</code></dd>", "<h2>Assistant</h2>", "<dt>Tours d&#39;outils par question</dt>", "<dd><code>4</code></dd>", "<dt>Durée d&#39;un lien de téléchargement</dt>", "<dd><code>30 min</code></dd>", "<dt>Nettoyage des téléchargements</dt>", "<dd><code>5 min</code></dd>"} {
        if !strings.Contains(body, text) {
            t.Errorf("missing %q", text)
        }
    }
    if software, network := strings.Index(body, "<h2>Logiciel</h2>"), strings.Index(body, "<h2>Réseau</h2>"); (software < 0) || (network < 0) || (software > network) {
        t.Errorf("the section Logiciel must come before the server settings: %d, %d", software, network)
    }
    if _, _, page := admin.get("/admin/settings"); strings.Contains(page, "Réseau") || strings.Contains(page, "<dt>Origines publiques</dt>") {
        t.Error("the configuration page must no longer show the server settings")
    }
}

func TestTheLargestMonthFormFitsTheSmallestBodyLimit(t *testing.T) {
    form := url.Values{"csrf": {strings.Repeat("c", 43)}, "revision": {"9223372036854775807"}}
    code := strings.Repeat("X", 16)
    for d := activity.NewDate(2026, time.October, 1); d.Month().Number() == 10; d = d.AddDays(1) {
        form.Set("am:" + d.String(), code)
        form.Set("pm:" + d.String(), code)
        form.Set("day:" + d.String(), code)
        form.Set("comment:" + d.String(), strings.Repeat("\U0001F600", activity.MaxCommentLength))
    }
    size := int64(len(form.Encode()))
    settings := activity.NewDefaultServerSettings()
    settings.MaxBodyBytes = size - 1
    if err := settings.Validate(); err == nil {
        t.Fatalf("a body limit below the largest legitimate form (%d bytes) must be refused", size)
    }
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
