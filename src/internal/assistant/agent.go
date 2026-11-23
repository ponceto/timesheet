/*
 * agent.go - Copyright (c) 2023-2026 - Olivier Poncet
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
// Agent declaration
// ---------------------------------------------------------------------------

type Agent struct {
    provider  Provider
    tools     []Tool
    maxRounds int64
    maxTokens int64
}

const finalInstruction = "Tu as atteint le nombre maximal d'appels d'outils : réponds maintenant à l'utilisateur avec ce que tu sais, sans appeler d'outil."

// ---------------------------------------------------------------------------
// Agent implementation
// ---------------------------------------------------------------------------

func NewAgent(provider Provider, tools []Tool, maxRounds int64, maxTokens int64) *Agent {
    return &Agent{provider: provider, tools: tools, maxRounds: maxRounds, maxTokens: maxTokens}
}

func (a *Agent) Answer(ctx context.Context, system string, history []Message) (Reply, error) {
    specs := []ToolSpec{}
    for _, tool := range a.tools {
        specs = append(specs, tool.Spec())
    }
    messages := append([]Message{}, history...)
    var input, output int64
    for round := int64(0); round <= a.maxRounds; round++ {
        prompt := system
        if round == a.maxRounds {
            prompt = system + "\n\n" + finalInstruction
        }
        reply, err := a.provider.Ask(ctx, Request{System: prompt, Messages: messages, Tools: specs, MaxTokens: a.maxTokens})
        if err != nil {
            return Reply{}, err
        }
        input, output = input + reply.InputTokens, output + reply.OutputTokens
        if (len(reply.Calls) == 0) || (round == a.maxRounds) {
            if reply.Text == "" {
                return Reply{}, ErrNoAnswer
            }
            reply.Calls, reply.InputTokens, reply.OutputTokens = nil, input, output
            return reply, nil
        }
        messages = append(messages, Message{Role: RoleAssistant, Text: reply.Text, Calls: reply.Calls})
        for _, call := range reply.Calls {
            messages = append(messages, Message{Role: RoleTool, CallID: call.ID, ToolName: call.Name, Text: a.run(ctx, call)})
        }
    }
    return Reply{}, ErrNoAnswer
}

func (a *Agent) run(ctx context.Context, call ToolCall) string {
    for _, tool := range a.tools {
        if tool.Spec().Name == call.Name {
            return tool.Run(ctx, call.Arguments)
        }
    }
    unknown, _ := json.Marshal(map[string]string{"error": "outil inconnu : " + call.Name})
    return string(unknown)
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
