/*
 * conversation.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "strings"
    "time"
    "unicode/utf8"
)

// ---------------------------------------------------------------------------
// Conversation declaration
// ---------------------------------------------------------------------------

type Conversation struct {
    ID           string          `json:"id"`
    Title        string          `json:"title"`
    Subscription SubscriptionRef `json:"subscription"`
    Updated      time.Time       `json:"updated"`
    Exchanges    []Exchange      `json:"exchanges"`
    Pending      *PendingAction  `json:"pending"`
}

const (
    MaxTitleLength = 60
    minTitleCut    = 20
    untitled       = "Nouvelle conversation"
)

// ---------------------------------------------------------------------------
// Conversation implementation
// ---------------------------------------------------------------------------

func (c Conversation) TitleOf(question string) string {
    words := strings.Join(strings.Fields(question), " ")
    if words == "" {
        return untitled
    }
    if utf8.RuneCountInString(words) <= MaxTitleLength {
        return words
    }
    runes := []rune(words)[:MaxTitleLength - 1]
    cut := strings.LastIndex(string(runes), " ")
    if cut < minTitleCut {
        return string(runes) + "…"
    }
    return string(runes)[:cut] + "…"
}

func (c Conversation) Waiting() bool {
    return (len(c.Exchanges) > 0) && c.Exchanges[len(c.Exchanges) - 1].Waiting()
}

func (c Conversation) Clone() Conversation {
    clone := c
    clone.Exchanges = append([]Exchange{}, c.Exchanges...)
    if c.Pending != nil {
        pending := c.Pending.Clone()
        clone.Pending = &pending
    }
    return clone
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
