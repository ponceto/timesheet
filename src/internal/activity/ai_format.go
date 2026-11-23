/*
 * ai_format.go - Copyright (c) 2023-2026 - Olivier Poncet
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

package activity

// ---------------------------------------------------------------------------
// AIFormat declaration
// ---------------------------------------------------------------------------

type AIFormat string

const (
    FormatOpenAI    AIFormat = "openai"
    FormatAnthropic AIFormat = "anthropic"
    FormatGemini    AIFormat = "gemini"
)

// ---------------------------------------------------------------------------
// AIFormat implementation
// ---------------------------------------------------------------------------

func (f AIFormat) Valid() bool {
    return (f == FormatOpenAI) || (f == FormatAnthropic) || (f == FormatGemini)
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
