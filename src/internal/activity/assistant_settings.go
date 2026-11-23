/*
 * assistant_settings.go - Copyright (c) 2023-2026 - Olivier Poncet
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

import (
    "time"
)

// ---------------------------------------------------------------------------
// AssistantSettings declaration
// ---------------------------------------------------------------------------

type AssistantSettings struct {
    Timeout       Duration `json:"timeout"`
    MaxRounds     int64    `json:"max_rounds"`
    MaxTokens     int64    `json:"max_tokens"`
    Conversations int64    `json:"conversations"`
    Exchanges     int64    `json:"exchanges"`
    QuestionBytes int64    `json:"question_bytes"`
    DownloadTTL   Duration `json:"download_ttl"`
    DownloadSweep Duration `json:"download_sweep"`
}

const (
    minAssistantTimeout = 5 * time.Second
    maxAssistantTimeout = 5 * time.Minute
    maxAssistantRounds  = 8
    minAssistantTokens  = 256
    maxAssistantTokens  = 32000
    MaxConversations    = 200
    MaxExchanges        = 100
    minQuestionBytes    = 100
    MaxQuestionBytes    = 16000
    minDownloadTTL      = time.Minute
    maxDownloadTTL      = 24 * time.Hour
    minDownloadSweep    = 10 * time.Second
    maxDownloadSweep    = time.Hour
)

// ---------------------------------------------------------------------------
// AssistantSettings implementation
// ---------------------------------------------------------------------------

func NewDefaultAssistantSettings() AssistantSettings {
    return AssistantSettings{Timeout: Duration(time.Minute), MaxRounds: 4, MaxTokens: 4096, Conversations: 50, Exchanges: 20, QuestionBytes: 4000, DownloadTTL: Duration(30 * time.Minute), DownloadSweep: Duration(5 * time.Minute)}
}

func (s AssistantSettings) Validate() error {
    check := NewSettingCheck("assistant")
    if err := check.Duration("timeout", s.Timeout, minAssistantTimeout, maxAssistantTimeout); err != nil {
        return err
    }
    if err := check.Integer("max_rounds", s.MaxRounds, 1, maxAssistantRounds); err != nil {
        return err
    }
    if err := check.Integer("max_tokens", s.MaxTokens, minAssistantTokens, maxAssistantTokens); err != nil {
        return err
    }
    if err := check.Integer("conversations", s.Conversations, 1, MaxConversations); err != nil {
        return err
    }
    if err := check.Integer("exchanges", s.Exchanges, 1, MaxExchanges); err != nil {
        return err
    }
    if err := check.Integer("question_bytes", s.QuestionBytes, minQuestionBytes, MaxQuestionBytes); err != nil {
        return err
    }
    if err := check.Duration("download_ttl", s.DownloadTTL, minDownloadTTL, maxDownloadTTL); err != nil {
        return err
    }
    return check.Duration("download_sweep", s.DownloadSweep, minDownloadSweep, maxDownloadSweep)
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
