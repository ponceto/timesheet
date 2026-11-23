/*
 * gemini_client.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "net/url"
    "strconv"
    "strings"
)

// ---------------------------------------------------------------------------
// GeminiClient declaration
// ---------------------------------------------------------------------------

type GeminiClient struct {
    transport *ProviderTransport
    endpoint  Endpoint
}

type geminiAnswer struct {
    Candidates []struct {
        Content struct {
            Parts []struct {
                Text         string `json:"text"`
                FunctionCall *struct {
                    Name string          `json:"name"`
                    Args json.RawMessage `json:"args"`
                } `json:"functionCall"`
            } `json:"parts"`
        } `json:"content"`
    } `json:"candidates"`
    UsageMetadata struct {
        PromptTokenCount     int64 `json:"promptTokenCount"`
        CandidatesTokenCount int64 `json:"candidatesTokenCount"`
    } `json:"usageMetadata"`
}

// ---------------------------------------------------------------------------
// GeminiClient implementation
// ---------------------------------------------------------------------------

func NewGeminiClient(transport *ProviderTransport, endpoint Endpoint) *GeminiClient {
    return &GeminiClient{transport: transport, endpoint: endpoint}
}

func (c *GeminiClient) Ask(ctx context.Context, request Request) (Reply, error) {
    payload := map[string]any{
        "systemInstruction": map[string]any{"parts": []map[string]any{{"text": request.System}}},
        "contents":          c.contents(request.Messages),
        "generationConfig":  map[string]any{"maxOutputTokens": request.MaxTokens},
    }
    if len(request.Tools) > 0 {
        declarations := []map[string]any{}
        for _, tool := range request.Tools {
            declarations = append(declarations, map[string]any{"name": tool.Name, "description": tool.Description, "parameters": tool.Parameters})
        }
        payload["tools"] = []map[string]any{{"functionDeclarations": declarations}}
    }
    answer := geminiAnswer{}
    headers := map[string]string{"x-goog-api-key": c.endpoint.Key}
    target := c.endpoint.BaseURL + "/models/" + url.PathEscape(c.endpoint.Model) + ":generateContent"
    if err := c.transport.Post(ctx, target, headers, payload, &answer); err != nil {
        return Reply{}, err
    }
    if len(answer.Candidates) == 0 {
        return Reply{}, ErrNoAnswer
    }
    reply := Reply{Model: c.endpoint.Model, InputTokens: answer.UsageMetadata.PromptTokenCount, OutputTokens: answer.UsageMetadata.CandidatesTokenCount}
    texts := []string{}
    for i, part := range answer.Candidates[0].Content.Parts {
        if part.FunctionCall != nil {
            arguments := part.FunctionCall.Args
            if !json.Valid(arguments) {
                arguments = json.RawMessage("{}")
            }
            reply.Calls = append(reply.Calls, ToolCall{ID: "call-" + strconv.Itoa(i), Name: part.FunctionCall.Name, Arguments: arguments})
            continue
        }
        if part.Text != "" {
            texts = append(texts, part.Text)
        }
    }
    reply.Text = strings.Join(texts, "")
    if (reply.Text == "") && (len(reply.Calls) == 0) {
        return Reply{}, ErrNoAnswer
    }
    return reply, nil
}

func (c *GeminiClient) contents(history []Message) []map[string]any {
    contents := []map[string]any{}
    for _, message := range history {
        switch message.Role {
            case RoleTool:
                contents = c.result(contents, map[string]any{"functionResponse": map[string]any{"name": message.ToolName, "response": c.response(message.Text)}})
            case RoleAssistant:
                parts := []map[string]any{}
                if message.Text != "" {
                    parts = append(parts, map[string]any{"text": message.Text})
                }
                for _, call := range message.Calls {
                    parts = append(parts, map[string]any{"functionCall": map[string]any{"name": call.Name, "args": call.Arguments}})
                }
                contents = append(contents, map[string]any{"role": "model", "parts": parts})
            default:
                contents = append(contents, map[string]any{"role": "user", "parts": []map[string]any{{"text": message.Text}}})
        }
    }
    return contents
}

func (c *GeminiClient) result(contents []map[string]any, response map[string]any) []map[string]any {
    if last := len(contents) - 1; last >= 0 {
        if parts, grouped := contents[last]["parts"].([]map[string]any); grouped && (contents[last]["role"] == "user") && (len(parts) > 0) && (parts[0]["functionResponse"] != nil) {
            contents[last]["parts"] = append(parts, response)
            return contents
        }
    }
    return append(contents, map[string]any{"role": "user", "parts": []map[string]any{response}})
}

func (c *GeminiClient) response(text string) any {
    object := map[string]any{}
    if json.Unmarshal([]byte(text), &object) == nil {
        return object
    }
    return map[string]any{"result": text}
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
