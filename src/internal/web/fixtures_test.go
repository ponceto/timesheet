/*
 * fixtures_test.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "bytes"
    "crypto/rand"
    "encoding/json"
    "html"
    "io"
    "log/slog"
    "mime/multipart"
    "net/http"
    "net/http/cookiejar"
    "net/http/httptest"
    "net/url"
    "path/filepath"
    "regexp"
    "slices"
    "strings"
    "sync"
    "testing"
    "time"

    "timesheet/internal/account"
    "timesheet/internal/activity"
    "timesheet/internal/store"
)

// ---------------------------------------------------------------------------
// fixtures
// ---------------------------------------------------------------------------

const testPassword = "correct horse battery"

type testSite struct {
    t        *testing.T
    server   *httptest.Server
    database *store.Database
    logs     *testJournal
    clock    *activity.ManualClock
    site     *Site
}

type testJournal struct {
    mu     sync.Mutex
    buffer bytes.Buffer
}

type testClient struct {
    site   *testSite
    client *http.Client
}

func newTestSite(t *testing.T) *testSite {
    t.Helper()
    return newTestSiteWith(t, nil)
}

func newTestSiteWith(t *testing.T, client *http.Client) *testSite {
    t.Helper()
    logs := &testJournal{}
    logger := slog.New(slog.NewTextHandler(logs, nil))
    path := filepath.Join(t.TempDir(), "timesheet.json")
    database := store.NewDatabase(path, logger, activity.NewDefaultDatabaseSettings().MaxBytes, activity.NewDefaultLimitSettings())
    if err := database.Create(activity.NewDefaultConfiguration()); err != nil {
        t.Fatal(err)
    }
    clock := activity.NewManualClock(time.Date(2026, time.November, 8, 10, 0, 0, 0, time.UTC))
    for login, profile := range map[string]account.Profile{"root": account.ProfileAdmin, "demo": account.ProfileUser} {
        hash, _ := account.NewPasswordHash(testPassword, account.MinIterations, rand.Reader)
        calendar := activity.NewCalendar(activity.NewMonth(2026, time.September).First())
        if err := database.CreateAccount(account.Account{Login: account.Login(login), Profile: profile, State: account.StateEnabled, Password: hash, Created: clock.Now()}, calendar); err != nil {
            t.Fatal(err)
        }
    }
    s := &testSite{t: t, database: database, logs: logs, clock: clock}
    s.server = httptest.NewUnstartedServer(nil)
    origins, err := NewPublicOrigins([]string{"http://" + s.server.Listener.Addr().String()})
    if err != nil {
        t.Fatal(err)
    }
    server := activity.NewDefaultServerConfiguration()
    server.Database.Path = "/srv/timesheet/timesheet-database.json"
    s.site, err = NewSite(SiteDependencies{Repository: database, Server: server, Clock: clock, Random: rand.Reader, Logger: logger, Origins: origins, Iterations: account.MinIterations, Version: "test", Client: client, Downloads: store.NewDownloadStore(path + activity.DownloadsSuffix, logger)})
    if err != nil {
        t.Fatal(err)
    }
    s.server.Config.Handler = s.site.Handler()
    s.server.Start()
    t.Cleanup(s.site.Stop)
    t.Cleanup(s.server.Close)
    return s
}

type fakeProvider struct {
    mu       sync.Mutex
    server   *httptest.Server
    replies  []string
    requests []string
}

func newAssistantSite(t *testing.T, replies ...string) (*testSite, *fakeProvider) {
    t.Helper()
    provider := &fakeProvider{replies: replies}
    provider.server = httptest.NewTLSServer(http.HandlerFunc(provider.serve))
    t.Cleanup(provider.server.Close)
    s := newTestSiteWith(t, provider.server.Client())
    config := s.database.Configuration()
    for i := range config.AIProviders {
        config.AIProviders[i].BaseURL = provider.server.URL + "/v1"
    }
    if err := s.database.SaveConfiguration(config.Revision, config); err != nil {
        t.Fatal(err)
    }
    return s, provider
}

func (p *fakeProvider) serve(w http.ResponseWriter, r *http.Request) {
    p.mu.Lock()
    defer p.mu.Unlock()
    var body bytes.Buffer
    body.ReadFrom(r.Body)
    p.requests = append(p.requests, r.URL.Path + " " + r.Header.Get("Authorization") + " " + body.String())
    if len(p.replies) == 0 {
        w.WriteHeader(http.StatusServiceUnavailable)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    io.WriteString(w, p.replies[0])
    p.replies = p.replies[1:]
}

func (p *fakeProvider) seen() []string {
    p.mu.Lock()
    defer p.mu.Unlock()
    return slices.Clone(p.requests)
}

func openAIText(text string) string {
    content, _ := json.Marshal(text)
    return `{"model":"fake-model","choices":[{"message":{"content":` + string(content) + `}}],"usage":{"prompt_tokens":3,"completion_tokens":2}}`
}

func openAICall(name string, arguments string) string {
    encoded, _ := json.Marshal(arguments)
    return `{"model":"fake-model","choices":[{"message":{"content":null,"tool_calls":[{"id":"call-` + name + `","type":"function","function":{"name":"` + name + `","arguments":` + string(encoded) + `}}]}}]}`
}

func (c *testClient) subscribe(model string, key string) {
    c.site.t.Helper()
    body := c.page("/dashboard/subscriptions")
    status, location, page := c.post("/dashboard/subscriptions", url.Values{"csrf": {c.token(body)}, "action": {"subscription-add"}, "model": {model}, "key": {key}, "name": {""}})
    if (status != http.StatusSeeOther) || (location != "/dashboard/subscriptions") {
        c.site.t.Fatalf("subscribe: %d %s %s", status, location, page)
    }
}

func (c *testClient) ask(conversation string, question string) string {
    c.site.t.Helper()
    body := c.page("/dashboard/assistant")
    status, location, page := c.post("/dashboard/assistant", url.Values{"csrf": {c.token(body)}, "action": {"ask"}, "conversation": {conversation}, "question": {question}})
    if (status != http.StatusSeeOther) || !strings.HasPrefix(location, "/dashboard/assistant/") {
        c.site.t.Fatalf("ask: %d %s %s", status, location, page)
    }
    return location
}

func (j *testJournal) Write(data []byte) (int, error) {
    j.mu.Lock()
    defer j.mu.Unlock()
    return j.buffer.Write(data)
}

func (j *testJournal) String() string {
    j.mu.Lock()
    defer j.mu.Unlock()
    return j.buffer.String()
}

func (j *testJournal) waitFor(text string) string {
    for attempt := 0; attempt < 400; attempt++ {
        if journal := j.String(); strings.Contains(journal, text) {
            return journal
        }
        time.Sleep(5 * time.Millisecond)
    }
    return j.String()
}

func (s *testSite) client() *testClient {
    jar, _ := cookiejar.New(nil)
    return &testClient{site: s, client: &http.Client{Jar: jar, CheckRedirect: s.noRedirect}}
}

func (s *testSite) noRedirect(req *http.Request, via []*http.Request) error {
    return http.ErrUseLastResponse
}

func (c *testClient) get(path string) (int, string, string) {
    c.site.t.Helper()
    response, err := c.client.Get(c.site.server.URL + path)
    if err != nil {
        c.site.t.Fatal(err)
    }
    defer response.Body.Close()
    var body bytes.Buffer
    body.ReadFrom(response.Body)
    return response.StatusCode, response.Header.Get("Location"), body.String()
}

func (c *testClient) post(path string, form url.Values) (int, string, string) {
    c.site.t.Helper()
    request, _ := http.NewRequest(http.MethodPost, c.site.server.URL + path, strings.NewReader(form.Encode()))
    request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    request.Header.Set("Origin", c.site.server.URL)
    response, err := c.client.Do(request)
    if err != nil {
        c.site.t.Fatal(err)
    }
    defer response.Body.Close()
    var body bytes.Buffer
    body.ReadFrom(response.Body)
    return response.StatusCode, response.Header.Get("Location"), body.String()
}

func (c *testClient) upload(path string, fields url.Values, name string, content string) (int, string, string) {
    c.site.t.Helper()
    var payload bytes.Buffer
    writer := multipart.NewWriter(&payload)
    for key, values := range fields {
        for _, value := range values {
            writer.WriteField(key, value)
        }
    }
    if name != "" {
        part, _ := writer.CreateFormFile("file", name)
        part.Write([]byte(content))
    }
    writer.Close()
    request, _ := http.NewRequest(http.MethodPost, c.site.server.URL + path, &payload)
    request.Header.Set("Content-Type", writer.FormDataContentType())
    request.Header.Set("Origin", c.site.server.URL)
    response, err := c.client.Do(request)
    if err != nil {
        c.site.t.Fatal(err)
    }
    defer response.Body.Close()
    var body bytes.Buffer
    body.ReadFrom(response.Body)
    return response.StatusCode, response.Header.Get("Location"), body.String()
}

func (c *testClient) download(path string) (int, http.Header, string) {
    c.site.t.Helper()
    response, err := c.client.Get(c.site.server.URL + path)
    if err != nil {
        c.site.t.Fatal(err)
    }
    defer response.Body.Close()
    var body bytes.Buffer
    body.ReadFrom(response.Body)
    return response.StatusCode, response.Header, body.String()
}

type memoryArchive struct {
    records []account.SessionRecord
    saves   int
}

func (a *memoryArchive) LoadSessions() ([]account.SessionRecord, error) {
    return a.records, nil
}

func (a *memoryArchive) SaveSessions(records []account.SessionRecord) error {
    a.records = records
    a.saves++
    return nil
}

func (c *testClient) page(path string) string {
    _, _, body := c.get(path)
    return body
}

func (c *testClient) token(body string) string {
    match := regexp.MustCompile(`name="csrf" value="([^"]+)"`).FindStringSubmatch(body)
    if match == nil {
        c.site.t.Fatal("no csrf token in the page")
    }
    return match[1]
}

func (c *testClient) field(body string, name string) string {
    match := regexp.MustCompile(`name="` + regexp.QuoteMeta(name) + `" value="([^"]*)"`).FindStringSubmatch(body)
    if match == nil {
        c.site.t.Fatalf("no field %s in the page", name)
    }
    return match[1]
}

func (c *testClient) login(login string) *testClient {
    c.site.t.Helper()
    _, _, body := c.get("/login")
    status, location, _ := c.post("/login", url.Values{"csrf": {c.token(body)}, "login": {login}, "password": {testPassword}})
    if (status != http.StatusSeeOther) || (location != "/dashboard/") {
        c.site.t.Fatalf("login %s: %d %s", login, status, location)
    }
    return c
}

func (c *testClient) monthForm(body string, overrides map[string]string) url.Values {
    form := url.Values{"csrf": {c.token(body)}, "revision": {c.field(body, "revision")}}
    selects := regexp.MustCompile(`(?s)<select name="((?:am|pm):[0-9-]+)"[^>]*>(.*?)</select>`)
    for _, match := range selects.FindAllStringSubmatch(body, -1) {
        selected := regexp.MustCompile(`<option value="([^"]*)" selected`).FindStringSubmatch(match[2])
        if selected != nil {
            form.Set(match[1], selected[1])
        }
    }
    comments := regexp.MustCompile(`<input type="text" name="(comment:[0-9-]+)" value="([^"]*)"`)
    for _, match := range comments.FindAllStringSubmatch(body, -1) {
        form.Set(match[1], html.UnescapeString(match[2]))
    }
    for name, value := range overrides {
        form.Set(name, value)
    }
    return form
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
