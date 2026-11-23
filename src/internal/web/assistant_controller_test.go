/*
 * assistant_controller_test.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "log/slog"
    "net/http"
    "net/url"
    "path/filepath"
    "regexp"
    "strings"
    "testing"
    "time"

    "timesheet/internal/activity"
    "timesheet/internal/store"
)

// ---------------------------------------------------------------------------
// AssistantController tests
// ---------------------------------------------------------------------------

func TestAssistantWithoutSubscriptionPointsToTheProfile(t *testing.T) {
    site := newTestSite(t)
    demo := site.client().login("demo")
    status, _, body := demo.get("/dashboard/assistant")
    if (status != http.StatusOK) || !strings.Contains(body, "ajoutez-en une dans « Mes souscriptions »") || strings.Contains(body, `name="question"`) {
        t.Fatalf("got %d", status)
    }
}

func TestAssistantAnswersWithATableThenWritesOnlyOnConfirmation(t *testing.T) {
    site, provider := newAssistantSite(t,
        openAICall("read_month", `{"month":"2026-11"}`),
        openAICall("propose_days", `{"month":"2026-11","days":[{"date":"2026-11-03","am":"TT","pm":"CP","comment":"pont"}]}`),
        openAIText("Voici la proposition :\n\n| Jour | Matin | Après-midi |\n| --- | --- | --- |\n| mardi 3 | TT | CP |\n\nConfirmez-la sous ma réponse."),
    )
    demo := site.client().login("demo")
    demo.subscribe("mistral|mistral-small-latest", "mistral-key")
    path := demo.ask("", "Pose-moi un CP mardi après-midi")
    if journal := site.logs.waitFor("assistant answered"); !strings.Contains(journal, "proposal=true") || strings.Contains(journal, "mistral-key") || strings.Contains(journal, "Pose-moi") {
        t.Fatalf("got %s", journal)
    }
    requests := provider.seen()
    if (len(requests) != 3) || !strings.Contains(requests[0], "/v1/chat/completions Bearer mistral-key") || !strings.Contains(requests[1], `"tool_call_id":"call-read_month"`) {
        t.Fatalf("got %v", requests)
    }
    page := demo.page(path)
    for _, text := range []string{"<caption>Tableau 1</caption>", `<th scope="col">Matin</th>`, "<h2>Action à confirmer</h2>", "mardi 03/11/2026 : matin TT (Temps de travail), après-midi CP (Congé payé)"} {
        if !strings.Contains(page, text) {
            t.Fatalf("missing %q in\n%s", text, page)
        }
    }
    if _, calendar, _ := site.database.Account("demo"); len(calendar.Days) != 0 {
        t.Fatal("a proposal writes nothing")
    }
    id := strings.TrimPrefix(path, "/dashboard/assistant/")
    status, location, _ := demo.post("/dashboard/assistant", url.Values{"csrf": {demo.token(page)}, "action": {"confirm"}, "conversation": {id}})
    if (status != http.StatusSeeOther) || (location != path) {
        t.Fatalf("confirm: %d %s", status, location)
    }
    _, calendar, _ := site.database.Account("demo")
    if entry := calendar.Days[activity.NewDate(2026, time.November, 3)]; (entry.AM != "TT") || (entry.PM != "CP") || (entry.Comment != "pont") {
        t.Fatalf("got %+v", entry)
    }
    after := demo.page(path)
    if strings.Contains(after, "<h2>Action à confirmer</h2>") || !strings.Contains(after, "Le mois a été enregistré.") || !strings.Contains(after, "Les demi-journées de novembre 2026 ont été enregistrées.") {
        t.Fatalf("the confirmation closes the action:\n%s", after)
    }
    if status, _, _ := demo.post("/dashboard/assistant", url.Values{"csrf": {demo.token(page)}, "action": {"confirm"}, "conversation": {id}}); status != http.StatusNotFound {
        t.Fatalf("an action is confirmed once: %d", status)
    }
    root := site.client().login("root")
    if status, _, _ := root.get(path); status != http.StatusNotFound {
        t.Fatalf("a conversation belongs to its account: %d", status)
    }
}

func TestAssistantRefusesAnActionOverTakenByAnEdit(t *testing.T) {
    site, _ := newAssistantSite(t,
        openAICall("propose_adjustment", `{"date":"2026-11-05","counter":"CP","amount":"1,5","reason":"Report accordé"}`),
        openAIText("Je propose un ajustement."),
    )
    demo := site.client().login("demo")
    demo.subscribe("mistral|mistral-small-latest", "mistral-key")
    path := demo.ask("", "Ajoute 1,5 jour de CP")
    site.logs.waitFor("assistant answered")
    page := demo.page(path)
    if !strings.Contains(page, "Ajustement de &#43;1,50 jour sur le compteur CP") {
        t.Fatalf("got\n%s", page)
    }
    activityPage := demo.page("/dashboard/activity/2026/11")
    if status, _, _ := demo.post("/dashboard/activity/2026/11", demo.monthForm(activityPage, map[string]string{"am:2026-11-02": "TT"})); status != http.StatusSeeOther {
        t.Fatalf("save: %d", status)
    }
    id := strings.TrimPrefix(path, "/dashboard/assistant/")
    status, _, body := demo.post("/dashboard/assistant", url.Values{"csrf": {demo.token(page)}, "action": {"confirm"}, "conversation": {id}})
    if (status != http.StatusConflict) || !strings.Contains(body, "ne correspond plus") {
        t.Fatalf("got %d", status)
    }
    if status, location, _ := demo.post("/dashboard/assistant", url.Values{"csrf": {demo.token(page)}, "action": {"cancel"}, "conversation": {id}}); (status != http.StatusSeeOther) || (location != path) {
        t.Fatalf("cancel: %d", status)
    }
    if _, calendar, _ := site.database.Account("demo"); len(calendar.Adjustments) != 0 {
        t.Fatal("a refused or cancelled action writes nothing")
    }
    if !strings.Contains(demo.page(path), "rien n&#39;a été écrit") {
        t.Fatal("the cancellation is noted")
    }
}

func TestAssistantReportsAProviderFailure(t *testing.T) {
    site, _ := newAssistantSite(t)
    demo := site.client().login("demo")
    demo.subscribe("mistral|mistral-small-latest", "mistral-key")
    path := demo.ask("", "Bonjour")
    site.logs.waitFor("assistant failed")
    if page := demo.page(path); !strings.Contains(page, "Le fournisseur a refusé la requête (statut 503)") {
        t.Fatalf("got\n%s", page)
    }
    list := demo.page("/dashboard/assistant")
    id := strings.TrimPrefix(path, "/dashboard/assistant/")
    if !strings.Contains(list, `<th scope="col">Actions</th>`) || !strings.Contains(list, `name="conversation" value="` + id + `"`) {
        t.Fatalf("each conversation of the list has its delete button:\n%s", list)
    }
    second := demo.ask("", "Encore")
    site.logs.waitFor("assistant failed")
    if status, location, _ := demo.post("/dashboard/assistant", url.Values{"csrf": {demo.token(list)}, "action": {"delete"}, "conversation": {id}}); (status != http.StatusSeeOther) || (location != "/dashboard/assistant") {
        t.Fatalf("delete one: %d %s", status, location)
    }
    if status, _, _ := demo.get(path); status != http.StatusNotFound {
        t.Fatalf("the deleted conversation is gone: %d", status)
    }
    if status, _, _ := demo.get(second); status != http.StatusOK {
        t.Fatalf("the other conversation stays: %d", status)
    }
    status, location, _ := demo.post("/dashboard/assistant", url.Values{"csrf": {demo.token(demo.page(path))}, "action": {"delete-all"}})
    if (status != http.StatusSeeOther) || (location != "/dashboard/assistant") || !strings.Contains(demo.page("/dashboard/assistant"), "Aucune conversation.") {
        t.Fatalf("delete all: %d %s", status, location)
    }
}

func TestAssistantGivesATemporaryExportToItsAccountOnly(t *testing.T) {
    site, provider := newAssistantSite(t,
        openAICall("export_data", `{}`),
        openAIText("Voici votre export."),
    )
    demo := site.client().login("demo")
    demo.subscribe("mistral|mistral-small-latest", "mistral-key")
    path := demo.ask("", "Exporte mes données")
    if journal := site.logs.waitFor("assistant answered"); !strings.Contains(journal, `msg="data exported" login=demo by=assistant`) {
        t.Fatalf("got %s", journal)
    }
    requests := provider.seen()
    link := regexp.MustCompile(`/dashboard/assistant/downloads/[0-9a-f]{32}`).FindString(requests[len(requests) - 1])
    if link == "" {
        t.Fatalf("the tool gives the link to the model: %v", requests)
    }
    status, headers, body := demo.download(link)
    if (status != http.StatusOK) || !strings.HasPrefix(headers.Get("Content-Disposition"), `attachment; filename="timesheet-demo-2026-11-08.json"`) || (headers.Get("Cache-Control") != "no-store") || !strings.Contains(body, `"days"`) {
        t.Fatalf("got %d %v %s", status, headers, body)
    }
    root := site.client().login("root")
    if status, _, _ := root.get(link); status != http.StatusNotFound {
        t.Fatalf("a download belongs to its account: %d", status)
    }
    site.clock.Advance(31 * time.Minute)
    if status, _, page := demo.get(link); (status != http.StatusGone) || !strings.Contains(page, "Lien expiré") {
        t.Fatalf("got %d", status)
    }
    if status, _, _ := demo.get(link); status != http.StatusNotFound {
        t.Fatalf("an expired download is removed: %d", status)
    }
    if !strings.Contains(demo.page(path), "Voici votre export.") {
        t.Fatal("the answer is shown")
    }
}

func TestAnswersLinkOnlyToTheDownloadsOfTheAssistant(t *testing.T) {
    site, _ := newAssistantSite(t, openAIText("[export](/dashboard/assistant/downloads/0123456789abcdef0123456789abcdef) et [ailleurs](/admin/accounts)"))
    demo := site.client().login("demo")
    demo.subscribe("mistral|mistral-small-latest", "mistral-key")
    path := demo.ask("", "Un lien ?")
    site.logs.waitFor("assistant answered")
    page := demo.page(path)
    if !strings.Contains(page, `<a href="/dashboard/assistant/downloads/0123456789abcdef0123456789abcdef" download>export</a>`) || strings.Contains(page, `href="/admin/accounts"`) {
        t.Fatalf("got\n%s", page)
    }
}

func TestDownloadSweeperRemovesTheExpiredExports(t *testing.T) {
    site := newTestSite(t)
    downloads := store.NewDownloadStore(filepath.Join(t.TempDir(), "ts.json.downloads"), slog.New(slog.NewTextHandler(site.logs, nil)))
    clock := activity.NewManualClock(time.Now())
    if _, err := downloads.Save("demo", []byte("{}"), clock.Now()); err != nil {
        t.Fatal(err)
    }
    sweeper := NewDownloadSweeper(downloads, clock, activity.NewDefaultAssistantSettings(), slog.New(slog.NewTextHandler(site.logs, nil)))
    if sweeper.Sweep() != 0 {
        t.Fatal("a fresh export is kept")
    }
    clock.Advance(31 * time.Minute)
    if (sweeper.Sweep() != 1) || !strings.Contains(site.logs.String(), `msg="downloads removed" count=1`) {
        t.Fatalf("got %s", site.logs.String())
    }
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
