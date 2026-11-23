/*
 * assistant_tool.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "context"
    "encoding/json"

    "timesheet/internal/assistant"
)

// ---------------------------------------------------------------------------
// AssistantTool declaration
// ---------------------------------------------------------------------------

type AssistantTool struct {
    spec assistant.ToolSpec
    run  func(arguments json.RawMessage) string
}

var _ assistant.Tool = AssistantTool{}

// ---------------------------------------------------------------------------
// AssistantTool implementation
// ---------------------------------------------------------------------------

func NewAssistantTool(name string, description string, parameters string, run func(arguments json.RawMessage) string) AssistantTool {
    return AssistantTool{spec: assistant.ToolSpec{Name: name, Description: description, Parameters: json.RawMessage(parameters)}, run: run}
}

func (t AssistantTool) Spec() assistant.ToolSpec {
    return t.spec
}

func (t AssistantTool) Run(ctx context.Context, arguments json.RawMessage) string {
    return t.run(arguments)
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
