/*
 * semantics_test.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "io/fs"
    "net/http"
    "net/url"
    "regexp"
    "strconv"
    "strings"
    "testing"
)

// ---------------------------------------------------------------------------
// semantics tests
// ---------------------------------------------------------------------------

func TestRenderedPagesHaveASemanticSkeleton(t *testing.T) {
    site := newTestSite(t)
    guest := site.client()
    user := site.client().login("demo")
    admin := site.client().login("root")
    pages := map[*testClient][]string{
        guest: {"/", "/login", "/nope"},
        user:  {"/dashboard/", "/dashboard/activity/2026/09", "/dashboard/month/2026/09", "/dashboard/month/2026/11", "/dashboard/year/2026", "/dashboard/settings", "/dashboard/data", "/dashboard/profile", "/dashboard/subscriptions", "/dashboard/assistant", "/logout", "/admin/"},
        admin: {"/admin/", "/admin/accounts", "/admin/accounts/demo", "/admin/settings", "/admin/accounts/demo/month/2026/10", "/admin/accounts/demo/year/2026"},
    }
    for client, paths := range pages {
        for _, path := range paths {
            status, _, body := client.get(path)
            if (status != http.StatusOK) && (status != http.StatusNotFound) && (status != http.StatusForbidden) {
                t.Errorf("%s: %d", path, status)
                continue
            }
            checkSkeleton(t, path, body)
        }
    }
}

func TestConversationPagesKeepTheHeadingOrder(t *testing.T) {
    site, _ := newAssistantSite(t, openAIText("# Soldes\n\n#### Détail\n\n| Compteur | Solde |\n| --- | --- |\n| CP | 12,5 |\n\n## Suite\n\n<script>alert(1)</script>"))
    demo := site.client().login("demo")
    demo.subscribe("mistral|mistral-small-latest", "mistral-key")
    path := demo.ask("", "Mes soldes ?")
    site.logs.waitFor("assistant answered")
    body := demo.page(path)
    checkSkeleton(t, path, body)
    if strings.Contains(body, "<script>alert") || !strings.Contains(body, "&lt;script&gt;alert(1)&lt;/script&gt;") || !strings.Contains(body, "<h5>Détail</h5>") {
        t.Fatalf("the answer must be escaped and its headings placed under the exchange:\n%s", body)
    }
}

func checkSkeleton(t *testing.T, path string, body string) {
    t.Helper()
    for _, element := range []string{`<html lang="fr"`, "<header>", "<main>", "<footer>", "</main>"} {
        if !strings.Contains(body, element) {
            t.Errorf("%s: missing %s", path, element)
        }
    }
    if strings.Contains(body, "<div") {
        t.Errorf("%s contains a div", path)
    }
    level := 0
    count := 0
    for _, match := range regexp.MustCompile(`<h([1-6])[ >]`).FindAllStringSubmatch(body, -1) {
        next, _ := strconv.Atoi(match[1])
        if next == 1 {
            count++
        }
        if next > (level + 1) {
            t.Errorf("%s: heading h%d after h%d", path, next, level)
        }
        level = next
    }
    if count != 1 {
        t.Errorf("%s: %d h1", path, count)
    }
}

func TestDayClassesExistInTheStyleSheet(t *testing.T) {
    site := newTestSite(t)
    user := site.client().login("demo")
    _, _, body := user.get("/dashboard/activity/2026/11")
    if status, _, _ := user.post("/dashboard/activity/2026/11", user.monthForm(body, map[string]string{"am:2026-11-03": "TP", "pm:2026-11-03": "TP"})); status != http.StatusSeeOther {
        t.Fatalf("save: %d", status)
    }
    data, err := fs.ReadFile(NewEmbeddedAssets().Static(), "css/components.css")
    if err != nil {
        t.Fatal(err)
    }
    _, _, colors := user.get("/dashboard/colors.css")
    classes := regexp.MustCompile(`class="([^"]*)"`)
    for _, path := range []string{"/dashboard/activity/2026/11", "/dashboard/month/2026/11", "/dashboard/year/2026"} {
        _, _, page := user.get(path)
        if !strings.Contains(page, "code-TP") {
            t.Errorf("%s: no class for the code TP", path)
        }
        for _, match := range classes.FindAllStringSubmatch(page, -1) {
            for _, name := range strings.Fields(match[1]) {
                sheet := string(data)
                if strings.HasPrefix(name, "code-") {
                    sheet = colors
                }
                if (strings.HasPrefix(name, "day-") || strings.HasPrefix(name, "code-")) && !strings.Contains(sheet, "." + name + " ") && !strings.Contains(sheet, "." + name + ",") {
                    t.Errorf("%s: class %s has no rule", path, name)
                }
            }
        }
    }
}

func TestColoursAndLettersFollowTheCodes(t *testing.T) {
    site := newTestSite(t)
    user := site.client().login("demo")
    _, _, page := user.get("/dashboard/settings")
    form := url.Values{"csrf": {user.token(page)}, "revision": {user.field(page, "revision")}, "configuration": {user.field(page, "configuration")}, "start": {"2026-09-01"}}
    for _, match := range regexp.MustCompile(`name="(color:[A-Za-z]+)" value="(#[0-9a-f]{6})"`).FindAllStringSubmatch(page, -1) {
        form.Set(match[1], match[2])
    }
    if !regexp.MustCompile(`<th scope="row">Temps de travail</th>(?s:.*?)<td><input type="color" name="color:TT" value="#[0-9a-f]{6}" aria-label="Couleur, TT : Temps de travail"></td>\s*</tr>`).MatchString(page) || !strings.Contains(page, `<th scope="col">Couleur</th>`) {
        t.Fatal("the colour of a code must sit in the last column of its counter")
    }
    form.Set("color:TT", "#c8b6e2")
    if status, _, _ := user.post("/dashboard/settings", form); status != http.StatusSeeOther {
        t.Fatalf("settings: %d", status)
    }
    if _, _, colors := user.get("/dashboard/colors.css"); !strings.Contains(colors, "--code-TT: #c8b6e2;") || !strings.Contains(colors, "--code-CP: #e01b24;") {
        t.Fatalf("colors.css: %s", colors)
    }
    _, _, body := user.get("/dashboard/activity/2026/11")
    if status, _, _ := user.post("/dashboard/activity/2026/11", user.monthForm(body, map[string]string{"am:2026-11-03": "TP"})); status != http.StatusSeeOther {
        t.Fatalf("save: %d", status)
    }
    if _, _, year := user.get("/dashboard/year/2026"); !regexp.MustCompile(`<td class="code-TP"><abbr title="[^"]*">TP</abbr></td>`).MatchString(year) || !strings.Contains(year, "<li class=\"code-TT\">TT : Temps de travail</li>") {
        t.Fatal("the year calendar must show the codes")
    }
    if _, _, year := user.get("/dashboard/year/2026"); (strings.Count(year, `<th scope="rowgroup" rowspan="2">`) != 12) || (strings.Count(year, `<th scope="row">Après-midi</th>`) != 12) || (strings.Count(year[strings.Index(year, `<section class="calendar">`):strings.Index(year, `<section class="summary">`)], "<td") != (12 * 2 * 31)) {
        t.Fatal("the year calendar must show two rows of 31 days per month, morning then afternoon")
    }
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
