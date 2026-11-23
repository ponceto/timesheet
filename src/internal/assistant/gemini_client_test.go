/*
 * gemini_client_test.go - Copyright (c) 2023-2026 - Olivier Poncet
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
// GeminiClient tests
// ---------------------------------------------------------------------------

func TestGeminiClientAnswersFunctionsByName(t *testing.T) {
    answer := `{"candidates":[{"content":{"parts":[{"functionCall":{"name":"read_year","args":{"year":2026}}}]}}],"usageMetadata":{"promptTokenCount":5,"candidatesTokenCount":3}}`
    server, captured := providerServer(t, http.StatusOK, answer)
    reply, err := endpointOf(server, activity.FormatGemini, "gemini-test").Provider(server.Client()).Ask(context.Background(), Request{System: "skill", Messages: toolHistory(), Tools: []ToolSpec{(&echoTool{}).Spec()}, MaxTokens: 512})
    if err != nil {
        t.Fatal(err)
    }
    if (captured.path != "/v1/models/gemini-test:generateContent") || (captured.headers.Get("x-goog-api-key") != "secret-key") {
        t.Fatalf("got %s %v", captured.path, captured.headers)
    }
    contents := captured.body["contents"].([]any)
    response := contents[2].(map[string]any)["parts"].([]any)[0].(map[string]any)["functionResponse"].(map[string]any)
    if (contents[1].(map[string]any)["role"] != "model") || (response["name"] != "read_balances") {
        t.Fatalf("got %v", contents)
    }
    if (len(reply.Calls) != 1) || (reply.Calls[0].Name != "read_year") || (reply.Model != "gemini-test") || (reply.InputTokens != 5) {
        t.Fatalf("got %+v", reply)
    }
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
