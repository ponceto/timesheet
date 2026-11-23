/*
 * openai_client_test.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "errors"
    "net/http"
    "testing"

    "timesheet/internal/activity"
)

// ---------------------------------------------------------------------------
// OpenAIClient tests
// ---------------------------------------------------------------------------

func TestOpenAIClientSendsTheToolsAndReadsTheCalls(t *testing.T) {
    answer := `{"model":"gpt-test","choices":[{"message":{"content":null,"tool_calls":[{"id":"c2","type":"function","function":{"name":"read_month","arguments":"{\"month\":\"2026-11\"}"}}]}}],"usage":{"prompt_tokens":11,"completion_tokens":7}}`
    server, captured := providerServer(t, http.StatusOK, answer)
    provider := endpointOf(server, activity.FormatOpenAI, "gpt-test").Provider(server.Client())
    reply, err := provider.Ask(context.Background(), Request{System: "skill", Messages: toolHistory(), Tools: []ToolSpec{(&echoTool{}).Spec()}, MaxTokens: 512})
    if err != nil {
        t.Fatal(err)
    }
    if (captured.path != "/v1/chat/completions") || (captured.headers.Get("Authorization") != "Bearer secret-key") {
        t.Fatalf("got %s %v", captured.path, captured.headers)
    }
    messages := captured.body["messages"].([]any)
    if (len(messages) != 4) || (messages[0].(map[string]any)["role"] != "system") || (messages[3].(map[string]any)["tool_call_id"] != "c1") {
        t.Fatalf("got %v", messages)
    }
    if (len(reply.Calls) != 1) || (reply.Calls[0].Name != "read_month") || (string(reply.Calls[0].Arguments) != `{"month":"2026-11"}`) || (reply.InputTokens != 11) || (reply.Model != "gpt-test") {
        t.Fatalf("got %+v", reply)
    }
}

func TestOpenAIClientReportsARefusalWithoutItsBody(t *testing.T) {
    server, _ := providerServer(t, http.StatusUnauthorized, `{"error":"bad key secret-key"}`)
    _, err := endpointOf(server, activity.FormatOpenAI, "m").Provider(server.Client()).Ask(context.Background(), Request{Messages: []Message{{Role: RoleUser, Text: "?"}}})
    refused := &ProviderError{}
    if !errors.As(err, &refused) || (refused.Status != http.StatusUnauthorized) || (err.Error() != "assistant: the provider refused the request with status 401") {
        t.Fatalf("got %v", err)
    }
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
