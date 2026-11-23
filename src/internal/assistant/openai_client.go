/*
 * openai_client.go - Copyright (c) 2023-2026 - Olivier Poncet
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
)

// ---------------------------------------------------------------------------
// OpenAIClient declaration
// ---------------------------------------------------------------------------

type OpenAIClient struct {
    transport *ProviderTransport
    endpoint  Endpoint
}

type openAIAnswer struct {
    Model   string `json:"model"`
    Choices []struct {
        Message struct {
            Content   *string `json:"content"`
            ToolCalls []struct {
                ID       string `json:"id"`
                Function struct {
                    Name      string `json:"name"`
                    Arguments string `json:"arguments"`
                } `json:"function"`
            } `json:"tool_calls"`
        } `json:"message"`
    } `json:"choices"`
    Usage struct {
        PromptTokens     int64 `json:"prompt_tokens"`
        CompletionTokens int64 `json:"completion_tokens"`
    } `json:"usage"`
}

// ---------------------------------------------------------------------------
// OpenAIClient implementation
// ---------------------------------------------------------------------------

func NewOpenAIClient(transport *ProviderTransport, endpoint Endpoint) *OpenAIClient {
    return &OpenAIClient{transport: transport, endpoint: endpoint}
}

func (c *OpenAIClient) Ask(ctx context.Context, request Request) (Reply, error) {
    messages := []map[string]any{{"role": "system", "content": request.System}}
    for _, message := range request.Messages {
        messages = append(messages, c.message(message))
    }
    payload := map[string]any{"model": c.endpoint.Model, "max_tokens": request.MaxTokens, "messages": messages}
    if len(request.Tools) > 0 {
        tools := []map[string]any{}
        for _, tool := range request.Tools {
            tools = append(tools, map[string]any{"type": "function", "function": map[string]any{"name": tool.Name, "description": tool.Description, "parameters": tool.Parameters}})
        }
        payload["tools"] = tools
    }
    answer := openAIAnswer{}
    headers := map[string]string{"Authorization": "Bearer " + c.endpoint.Key}
    if err := c.transport.Post(ctx, c.endpoint.BaseURL + "/chat/completions", headers, payload, &answer); err != nil {
        return Reply{}, err
    }
    if len(answer.Choices) == 0 {
        return Reply{}, ErrNoAnswer
    }
    reply := Reply{Model: answer.Model, InputTokens: answer.Usage.PromptTokens, OutputTokens: answer.Usage.CompletionTokens}
    choice := answer.Choices[0].Message
    if choice.Content != nil {
        reply.Text = *choice.Content
    }
    for _, call := range choice.ToolCalls {
        arguments := json.RawMessage(call.Function.Arguments)
        if !json.Valid(arguments) {
            arguments = json.RawMessage("{}")
        }
        reply.Calls = append(reply.Calls, ToolCall{ID: call.ID, Name: call.Function.Name, Arguments: arguments})
    }
    return reply, nil
}

func (c *OpenAIClient) message(message Message) map[string]any {
    switch message.Role {
        case RoleTool:
            return map[string]any{"role": "tool", "tool_call_id": message.CallID, "content": message.Text}
        case RoleAssistant:
            if len(message.Calls) == 0 {
                return map[string]any{"role": "assistant", "content": message.Text}
            }
            calls := []map[string]any{}
            for _, call := range message.Calls {
                calls = append(calls, map[string]any{"id": call.ID, "type": "function", "function": map[string]any{"name": call.Name, "arguments": string(call.Arguments)}})
            }
            return map[string]any{"role": "assistant", "content": message.Text, "tool_calls": calls}
    }
    return map[string]any{"role": "user", "content": message.Text}
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
