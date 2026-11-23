/*
 * state.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "errors"
    "fmt"
    "slices"
    "strings"
    "unicode/utf8"

    "timesheet/internal/account"
    "timesheet/internal/activity"
)

// ---------------------------------------------------------------------------
// State declaration
// ---------------------------------------------------------------------------

type State struct {
    Revision      int64             `json:"revision"`
    Subscriptions []Subscription    `json:"subscriptions"`
    Hidden        []SubscriptionRef `json:"hidden"`
    Conversations []Conversation    `json:"conversations"`
}

const maxHidden = 100

var (
    ErrInvalidState    = errors.New("assistant: invalid state")
    ErrUnknownProvider = errors.New("assistant: unknown provider")
    ErrInvalidModel    = errors.New("assistant: invalid model")
    ErrInvalidName     = errors.New("assistant: invalid subscription name")
    ErrInvalidKey      = errors.New("assistant: invalid API key")
    ErrTooMany         = errors.New("assistant: too many subscriptions or shares")
    ErrUnknownShare    = errors.New("assistant: unknown or invalid account to share with")
    ErrUnusable        = errors.New("assistant: subscription unknown or not usable")
)

// ---------------------------------------------------------------------------
// State implementation
// ---------------------------------------------------------------------------

func (s State) Validate(config activity.Configuration, owner account.Login, exists func(account.Login) bool) error {
    if len(s.Subscriptions) > MaxSubscriptions {
        return s.fail("subscriptions", "at most %d subscriptions expected", MaxSubscriptions)
    }
    identifiers := map[string]bool{}
    for i, subscription := range s.Subscriptions {
        path := fmt.Sprintf("subscriptions[%d]", i)
        if err := s.validateSubscription(config, owner, subscription, exists); err != nil {
            return s.fail(path, "%v", err)
        }
        if identifiers[subscription.ID] {
            return s.fail(path + ".id", "duplicate identifier")
        }
        identifiers[subscription.ID] = true
    }
    if len(s.Hidden) > maxHidden {
        return s.fail("hidden", "at most %d hidden subscriptions expected", maxHidden)
    }
    for i, ref := range s.Hidden {
        if !ref.Owner.Valid() || (ref.Owner == owner) || !s.ValidIdentifier(ref.ID) {
            return s.fail(fmt.Sprintf("hidden[%d]", i), "the subscription of another account expected")
        }
    }
    if len(s.Conversations) > activity.MaxConversations {
        return s.fail("conversations", "at most %d conversations expected", activity.MaxConversations)
    }
    conversations := map[string]bool{}
    for i, conversation := range s.Conversations {
        path := fmt.Sprintf("conversations[%d]", i)
        if err := s.validateConversation(conversation); err != nil {
            return s.fail(path, "%v", err)
        }
        if conversations[conversation.ID] {
            return s.fail(path + ".id", "duplicate identifier")
        }
        conversations[conversation.ID] = true
    }
    return nil
}

func (s State) validateSubscription(config activity.Configuration, owner account.Login, subscription Subscription, exists func(account.Login) bool) error {
    provider, known := config.Provider(subscription.Provider)
    switch {
        case !s.ValidIdentifier(subscription.ID):
            return errors.New("id: invalid identifier")
        case !s.ValidLabel(subscription.Name, MaxSubscriptionName):
            return ErrInvalidName
        case !known:
            return ErrUnknownProvider
        case !provider.ValidModel(subscription.Model):
            return ErrInvalidModel
        case (subscription.Key == "") || (len(subscription.Key) > (2 * MaxKeyLength)):
            return ErrInvalidKey
        case subscription.Created.IsZero():
            return errors.New("created: a time expected")
        case len(subscription.SharedWith) > MaxShares:
            return ErrTooMany
    }
    seen := map[account.Login]bool{}
    for _, login := range subscription.SharedWith {
        if !login.Valid() || (login == owner) || seen[login] || !exists(login) {
            return ErrUnknownShare
        }
        seen[login] = true
    }
    return nil
}

func (s State) validateConversation(conversation Conversation) error {
    switch {
        case !s.ValidIdentifier(conversation.ID):
            return errors.New("id: invalid identifier")
        case !s.ValidLabel(conversation.Title, MaxTitleLength):
            return errors.New("title: invalid title")
        case !conversation.Subscription.Owner.Valid() || !s.ValidIdentifier(conversation.Subscription.ID):
            return errors.New("subscription: invalid reference")
        case conversation.Updated.IsZero():
            return errors.New("updated: a time expected")
        case len(conversation.Exchanges) > activity.MaxExchanges:
            return errors.New("exchanges: too many exchanges")
    }
    for i, exchange := range conversation.Exchanges {
        if !s.ValidText(exchange.Question, activity.MaxQuestionBytes) || !s.ValidText(exchange.Answer, MaxAnswerBytes) || !s.ValidText(exchange.Failure, MaxFailureBytes) || !s.ValidOptionalLabel(exchange.Provider, activity.MaxAIModelLength) || !s.ValidOptionalLabel(exchange.Model, activity.MaxAIModelLength) || exchange.Asked.IsZero() {
            return fmt.Errorf("exchanges[%d]: invalid exchange", i)
        }
    }
    if pending := conversation.Pending; pending != nil {
        if (pending.Tool == "") || !s.ValidLabel(pending.Tool, 40) || (len(pending.Arguments) > MaxArgumentBytes) || !utf8.ValidString(pending.Arguments) || (len(pending.Summary) > MaxSummaryLines) || pending.Proposed.IsZero() {
            return errors.New("pending: invalid action")
        }
        for _, line := range pending.Summary {
            if !s.ValidLabel(line, MaxSummaryLine) {
                return errors.New("pending.summary: invalid line")
            }
        }
    }
    return nil
}

func (s State) ValidIdentifier(id string) bool {
    if (len(id) < 1) || (len(id) > 32) {
        return false
    }
    for _, letter := range []byte(id) {
        if ((letter < 'a') || (letter > 'f')) && ((letter < '0') || (letter > '9')) {
            return false
        }
    }
    return true
}

func (s State) ValidLabel(label string, max int) bool {
    return (label != "") && s.ValidOptionalLabel(label, max)
}

func (s State) ValidOptionalLabel(label string, max int) bool {
    return (strings.TrimSpace(label) == label) && activity.NewTextPolicy().Accepts(label, max)
}

func (s State) ValidText(text string, maxBytes int) bool {
    if !utf8.ValidString(text) || (len(text) > maxBytes) {
        return false
    }
    policy := activity.NewTextPolicy()
    for _, line := range strings.Split(text, "\n") {
        if !policy.Accepts(strings.ReplaceAll(line, "\t", "    "), maxBytes) {
            return false
        }
    }
    return true
}

func (s State) fail(path string, format string, args ...any) error {
    return fmt.Errorf("%w: %s: %s", ErrInvalidState, path, fmt.Sprintf(format, args...))
}

func (s State) Subscription(id string) (int, bool) {
    for i, subscription := range s.Subscriptions {
        if subscription.ID == id {
            return i, true
        }
    }
    return -1, false
}

func (s State) Conversation(id string) (int, bool) {
    for i, conversation := range s.Conversations {
        if conversation.ID == id {
            return i, true
        }
    }
    return -1, false
}

func (s State) IsHidden(ref SubscriptionRef) bool {
    return slices.Contains(s.Hidden, ref)
}

func (s State) WithoutShare(login account.Login) State {
    next := s.Clone()
    for i, subscription := range next.Subscriptions {
        next.Subscriptions[i].SharedWith = slices.DeleteFunc(subscription.SharedWith, login.Equal)
    }
    hidden := []SubscriptionRef{}
    for _, ref := range next.Hidden {
        if ref.Owner != login {
            hidden = append(hidden, ref)
        }
    }
    next.Hidden = hidden
    return next
}

func (s State) Clone() State {
    clone := s
    clone.Subscriptions = make([]Subscription, 0, len(s.Subscriptions))
    for _, subscription := range s.Subscriptions {
        clone.Subscriptions = append(clone.Subscriptions, subscription.Clone())
    }
    clone.Hidden = append([]SubscriptionRef{}, s.Hidden...)
    clone.Conversations = make([]Conversation, 0, len(s.Conversations))
    for _, conversation := range s.Conversations {
        clone.Conversations = append(clone.Conversations, conversation.Clone())
    }
    return clone
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
