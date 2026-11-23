/*
 * state_test.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "testing"
    "time"

    "timesheet/internal/account"
    "timesheet/internal/activity"
)

// ---------------------------------------------------------------------------
// State tests
// ---------------------------------------------------------------------------

func TestStateRefusesAnInvalidSubscriptionOrConversation(t *testing.T) {
    exists := map[account.Login]bool{"demo": true}
    now := time.Date(2026, 11, 8, 10, 0, 0, 0, time.UTC)
    valid := State{
        Subscriptions: []Subscription{{ID: "0a1b", Name: "Claude", Provider: "anthropic", Model: "claude-sonnet-5-5", Key: "c2VhbGVk", SharedWith: []account.Login{"demo"}, Created: now}},
        Conversations: []Conversation{{ID: "0c", Title: "Soldes", Subscription: SubscriptionRef{Owner: "root", ID: "0a1b"}, Updated: now, Exchanges: []Exchange{{Question: "Mon solde ?\nMerci", Answer: "| a |\n| - |", Asked: now, Answered: activity.NullableTime(now)}}}},
    }
    check := func(state State) error {
        return state.Validate(activity.NewDefaultConfiguration(), "root", func(login account.Login) bool { return exists[login] })
    }
    if err := check(valid); err != nil {
        t.Fatal(err)
    }
    cases := map[string]func(*State){
        "unknown provider": func(s *State) { s.Subscriptions[0].Provider = "other" },
        "shared to the owner": func(s *State) { s.Subscriptions[0].SharedWith = []account.Login{"root"} },
        "shared to nobody": func(s *State) { s.Subscriptions[0].SharedWith = []account.Login{"ghost"} },
        "empty key": func(s *State) { s.Subscriptions[0].Key = "" },
        "control character": func(s *State) { s.Conversations[0].Exchanges[0].Answer = "a\u0007b" },
        "duplicate conversation": func(s *State) { s.Conversations = append(s.Conversations, s.Conversations[0]) },
        "hidden own subscription": func(s *State) { s.Hidden = []SubscriptionRef{{Owner: "root", ID: "0a1b"}} },
    }
    for name, mutate := range cases {
        state := valid.Clone()
        mutate(&state)
        if err := check(state); !errors.Is(err, ErrInvalidState) {
            t.Errorf("%s: got %v", name, err)
        }
    }
}

func TestConversationTitleCutsAtAWord(t *testing.T) {
    title := (Conversation{}).TitleOf("Combien de jours de congés payés me reste-t-il au 31 mai, en comptant le mois en cours ?")
    if title != "Combien de jours de congés payés me reste-t-il au 31 mai,…" {
        t.Fatalf("got %q", title)
    }
    if (Conversation{}).TitleOf("   ") != untitled {
        t.Fatal("an empty question gives the default title")
    }
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
