/*
 * anthropic_client.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "strings"
)

// ---------------------------------------------------------------------------
// AnthropicClient declaration
// ---------------------------------------------------------------------------

type AnthropicClient struct {
    transport *ProviderTransport
    endpoint  Endpoint
}

type anthropicAnswer struct {
    Model   string `json:"model"`
    Content []struct {
        Type  string          `json:"type"`
        Text  string          `json:"text"`
        ID    string          `json:"id"`
        Name  string          `json:"name"`
        Input json.RawMessage `json:"input"`
    } `json:"content"`
    Usage struct {
        InputTokens  int64 `json:"input_tokens"`
        OutputTokens int64 `json:"output_tokens"`
    } `json:"usage"`
}

const anthropicVersion = "2023-06-01"

// ---------------------------------------------------------------------------
// AnthropicClient implementation
// ---------------------------------------------------------------------------

func NewAnthropicClient(transport *ProviderTransport, endpoint Endpoint) *AnthropicClient {
    return &AnthropicClient{transport: transport, endpoint: endpoint}
}

func (c *AnthropicClient) Ask(ctx context.Context, request Request) (Reply, error) {
    payload := map[string]any{"model": c.endpoint.Model, "system": request.System, "max_tokens": request.MaxTokens, "messages": c.messages(request.Messages)}
    if len(request.Tools) > 0 {
        tools := []map[string]any{}
        for _, tool := range request.Tools {
            tools = append(tools, map[string]any{"name": tool.Name, "description": tool.Description, "input_schema": tool.Parameters})
        }
        payload["tools"] = tools
    }
    answer := anthropicAnswer{}
    headers := map[string]string{"x-api-key": c.endpoint.Key, "anthropic-version": anthropicVersion}
    if err := c.transport.Post(ctx, c.endpoint.BaseURL + "/messages", headers, payload, &answer); err != nil {
        return Reply{}, err
    }
    reply := Reply{Model: answer.Model, InputTokens: answer.Usage.InputTokens, OutputTokens: answer.Usage.OutputTokens}
    texts := []string{}
    for _, block := range answer.Content {
        switch block.Type {
            case "text":
                texts = append(texts, block.Text)
            case "tool_use":
                arguments := block.Input
                if !json.Valid(arguments) {
                    arguments = json.RawMessage("{}")
                }
                reply.Calls = append(reply.Calls, ToolCall{ID: block.ID, Name: block.Name, Arguments: arguments})
        }
    }
    reply.Text = strings.Join(texts, "\n\n")
    if (reply.Text == "") && (len(reply.Calls) == 0) {
        return Reply{}, ErrNoAnswer
    }
    return reply, nil
}

func (c *AnthropicClient) messages(history []Message) []map[string]any {
    messages := []map[string]any{}
    for _, message := range history {
        switch message.Role {
            case RoleTool:
                messages = c.result(messages, map[string]any{"type": "tool_result", "tool_use_id": message.CallID, "content": message.Text})
            case RoleAssistant:
                blocks := []map[string]any{}
                if message.Text != "" {
                    blocks = append(blocks, map[string]any{"type": "text", "text": message.Text})
                }
                for _, call := range message.Calls {
                    blocks = append(blocks, map[string]any{"type": "tool_use", "id": call.ID, "name": call.Name, "input": call.Arguments})
                }
                messages = append(messages, map[string]any{"role": "assistant", "content": blocks})
            default:
                messages = append(messages, map[string]any{"role": "user", "content": message.Text})
        }
    }
    return messages
}

func (c *AnthropicClient) result(messages []map[string]any, result map[string]any) []map[string]any {
    if last := len(messages) - 1; last >= 0 {
        if results, grouped := messages[last]["content"].([]map[string]any); grouped && (messages[last]["role"] == "user") {
            messages[last]["content"] = append(results, result)
            return messages
        }
    }
    return append(messages, map[string]any{"role": "user", "content": []map[string]any{result}})
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
