/*
 * anthropic_client_test.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "net/http"
    "testing"

    "timesheet/internal/activity"
)

// ---------------------------------------------------------------------------
// AnthropicClient tests
// ---------------------------------------------------------------------------

func TestAnthropicClientGroupsTheToolResults(t *testing.T) {
    answer := `{"model":"claude-test","content":[{"type":"text","text":"Votre solde est de 12,5 jours."}],"usage":{"input_tokens":20,"output_tokens":9}}`
    server, captured := providerServer(t, http.StatusOK, answer)
    history := append(toolHistory(), Message{Role: RoleTool, CallID: "c3", ToolName: "read_month", Text: "{}"})
    reply, err := endpointOf(server, activity.FormatAnthropic, "claude-test").Provider(server.Client()).Ask(context.Background(), Request{System: "skill", Messages: history, MaxTokens: 512})
    if err != nil {
        t.Fatal(err)
    }
    if (captured.path != "/v1/messages") || (captured.headers.Get("x-api-key") != "secret-key") || (captured.headers.Get("anthropic-version") != anthropicVersion) || (captured.body["system"] != "skill") {
        t.Fatalf("got %s %v %v", captured.path, captured.headers, captured.body["system"])
    }
    messages := captured.body["messages"].([]any)
    last := messages[len(messages) - 1].(map[string]any)
    if (len(messages) != 3) || (last["role"] != "user") || (len(last["content"].([]any)) != 2) {
        t.Fatalf("the two tool results must travel in one user message: %v", messages)
    }
    if (reply.Text != "Votre solde est de 12,5 jours.") || (reply.OutputTokens != 9) {
        t.Fatalf("got %+v", reply)
    }
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
