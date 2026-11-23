/*
 * subscription_choice.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "strings"

    "timesheet/internal/account"
    "timesheet/internal/activity"
    "timesheet/internal/assistant"
)

// ---------------------------------------------------------------------------
// SubscriptionChoice declaration
// ---------------------------------------------------------------------------

type SubscriptionChoice struct {
    Ref   assistant.SubscriptionRef
    Value string
    Label string
}

// ---------------------------------------------------------------------------
// SubscriptionChoice implementation
// ---------------------------------------------------------------------------

func NewSubscriptionChoices(repository Repository, login account.Login) []SubscriptionChoice {
    config := repository.Configuration()
    choices := []SubscriptionChoice{}
    if state, found := repository.AssistantState(login); found {
        for _, subscription := range state.Subscriptions {
            choices = append(choices, NewSubscriptionChoice(config, assistant.SubscriptionRef{Owner: login, ID: subscription.ID}, subscription, ""))
        }
    }
    for _, shared := range repository.SharedSubscriptions(login) {
        if !shared.Hidden {
            choices = append(choices, NewSubscriptionChoice(config, shared.Ref(), shared.Subscription, shared.Owner))
        }
    }
    return choices
}

func NewSubscriptionChoice(config activity.Configuration, ref assistant.SubscriptionRef, subscription assistant.Subscription, owner account.Login) SubscriptionChoice {
    provider := subscription.Provider
    if known, found := config.Provider(subscription.Provider); found {
        provider = known.Label
    }
    label := subscription.Name + " (" + provider + ", " + subscription.Model + ")"
    if owner != "" {
        label += ", partagée par " + string(owner)
    }
    return SubscriptionChoice{Ref: ref, Value: string(ref.Owner) + ":" + ref.ID, Label: label}
}

func (c SubscriptionChoice) Find(choices []SubscriptionChoice, value string) (SubscriptionChoice, bool) {
    value = strings.TrimSpace(value)
    for _, choice := range choices {
        if (choice.Value == value) || ((value == "") && (choice.Value == choices[0].Value)) {
            return choice, true
        }
    }
    return SubscriptionChoice{}, false
}

func (c SubscriptionChoice) Usable(choices []SubscriptionChoice, ref assistant.SubscriptionRef) (SubscriptionChoice, bool) {
    for _, choice := range choices {
        if choice.Ref == ref {
            return choice, true
        }
    }
    return SubscriptionChoice{}, false
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
