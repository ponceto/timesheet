/*
 * tool_proposal.go - Copyright (c) 2023-2026 - Olivier Poncet
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

import "encoding/json"

// ---------------------------------------------------------------------------
// ToolProposal declaration
// ---------------------------------------------------------------------------

type ToolProposal struct {
    box  *ToolBox
    tool string
}

// ---------------------------------------------------------------------------
// ToolProposal implementation
// ---------------------------------------------------------------------------

func NewToolProposal(box *ToolBox, tool string) *ToolProposal {
    return &ToolProposal{box: box, tool: tool}
}

func (p *ToolProposal) Run(arguments json.RawMessage) string {
    return p.box.record(p.tool, arguments)
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
