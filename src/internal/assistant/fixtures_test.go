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

package assistant

import (
    "context"
    "encoding/json"
    "io"
    "net/http"
    "net/http/httptest"
    "testing"

    "timesheet/internal/activity"
)

// ---------------------------------------------------------------------------
// fixtures
// ---------------------------------------------------------------------------

type scriptedProvider struct {
    replies  []Reply
    requests []Request
}

func (p *scriptedProvider) Ask(ctx context.Context, request Request) (Reply, error) {
    p.requests = append(p.requests, request)
    if len(p.replies) == 0 {
        return Reply{}, ErrNoAnswer
    }
    reply := p.replies[0]
    p.replies = p.replies[1:]
    return reply, nil
}

type echoTool struct {
    calls []string
}

func (t *echoTool) Spec() ToolSpec {
    return ToolSpec{Name: "read_balances", Description: "Soldes", Parameters: json.RawMessage(`{"type":"object","properties":{}}`)}
}

func (t *echoTool) Run(ctx context.Context, arguments json.RawMessage) string {
    t.calls = append(t.calls, string(arguments))
    return `{"CP":"12,5"}`
}

type capturedRequest struct {
    path    string
    headers http.Header
    body    map[string]any
}

func providerServer(t *testing.T, status int, answer string) (*httptest.Server, *capturedRequest) {
    t.Helper()
    captured := &capturedRequest{}
    server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        data, _ := io.ReadAll(r.Body)
        captured.path, captured.headers = r.URL.Path, r.Header.Clone()
        if err := json.Unmarshal(data, &captured.body); err != nil {
            t.Errorf("the request is not JSON: %v", err)
        }
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(status)
        io.WriteString(w, answer)
    }))
    t.Cleanup(server.Close)
    return server, captured
}

func endpointOf(server *httptest.Server, format activity.AIFormat, model string) Endpoint {
    return Endpoint{Format: format, BaseURL: server.URL + "/v1", Model: model, Key: "secret-key"}
}

func toolHistory() []Message {
    return []Message{
        {Role: RoleUser, Text: "Mon solde ?"},
        {Role: RoleAssistant, Text: "Je regarde.", Calls: []ToolCall{{ID: "c1", Name: "read_balances", Arguments: json.RawMessage(`{}`)}}},
        {Role: RoleTool, CallID: "c1", ToolName: "read_balances", Text: `{"CP":"12,5"}`},
    }
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
