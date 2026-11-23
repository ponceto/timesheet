/*
 * agent_test.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "testing"
)

// ---------------------------------------------------------------------------
// Agent tests
// ---------------------------------------------------------------------------

func TestAgentRunsTheToolsThenAnswers(t *testing.T) {
    tool := &echoTool{}
    provider := &scriptedProvider{replies: []Reply{
        {Calls: []ToolCall{{ID: "c1", Name: "read_balances", Arguments: json.RawMessage(`{}`)}, {ID: "c2", Name: "unknown", Arguments: json.RawMessage(`{}`)}}, InputTokens: 10, OutputTokens: 2},
        {Text: "Il vous reste 12,5 jours.", InputTokens: 30, OutputTokens: 8},
    }}
    reply, err := NewAgent(provider, []Tool{tool}, 4, 1024).Answer(context.Background(), "skill", []Message{{Role: RoleUser, Text: "Mon solde ?"}})
    if err != nil {
        t.Fatal(err)
    }
    if (reply.Text != "Il vous reste 12,5 jours.") || (reply.InputTokens != 40) || (reply.OutputTokens != 10) || (len(tool.calls) != 1) {
        t.Fatalf("got %+v %v", reply, tool.calls)
    }
    second := provider.requests[1].Messages
    if (len(second) != 4) || (second[2].Text != `{"CP":"12,5"}`) || !strings.Contains(second[3].Text, "outil inconnu") {
        t.Fatalf("got %+v", second)
    }
}

func TestAgentAsksForAFinalAnswerAfterTheLastRound(t *testing.T) {
    call := Reply{Text: "encore", Calls: []ToolCall{{ID: "c", Name: "read_balances", Arguments: json.RawMessage(`{}`)}}}
    provider := &scriptedProvider{replies: []Reply{call, call, call}}
    reply, err := NewAgent(provider, []Tool{&echoTool{}}, 2, 1024).Answer(context.Background(), "skill", []Message{{Role: RoleUser, Text: "?"}})
    if err != nil {
        t.Fatal(err)
    }
    if (len(provider.requests) != 3) || !strings.Contains(provider.requests[2].System, finalInstruction) || (reply.Text != "encore") || (reply.Calls != nil) {
        t.Fatalf("got %d requests, %+v", len(provider.requests), reply)
    }
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
