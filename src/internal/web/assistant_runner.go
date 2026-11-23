/*
 * assistant_runner.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "errors"
    "log/slog"
    "net/http"
    "strconv"
    "strings"
    "sync"

    "timesheet/internal/account"
    "timesheet/internal/activity"
    "timesheet/internal/assistant"
)

// ---------------------------------------------------------------------------
// AssistantRunner declaration
// ---------------------------------------------------------------------------

type AssistantRunner struct {
    mu         sync.Mutex
    busy       map[account.Login]bool
    ctx        context.Context
    cancel     context.CancelFunc
    wait       sync.WaitGroup
    repository Repository
    loader     *WorkspaceLoader
    client     *http.Client
    settings   activity.AssistantSettings
    planner    *ActionPlanner
    downloads  Downloads
    skill      *assistant.Skill
    clock      activity.Clock
    logger     *slog.Logger
}

const (
    saveAttempts    = 3
    interrupted     = "La réponse a été interrompue : posez la question à nouveau."
    unusable        = "La souscription de cette conversation n'est plus utilisable : elle a été supprimée, n'est plus partagée ou sa clef est illisible."
    unreachable     = "Le fournisseur est injoignable ou sa réponse est illisible."
    late            = "Le fournisseur n'a pas répondu à temps."
    unknownProvider = "Le fournisseur de cette souscription n'existe plus dans la configuration."
)

// ---------------------------------------------------------------------------
// AssistantRunner implementation
// ---------------------------------------------------------------------------

func NewAssistantRunner(repository Repository, loader *WorkspaceLoader, client *http.Client, settings activity.AssistantSettings, planner *ActionPlanner, downloads Downloads, clock activity.Clock, logger *slog.Logger) *AssistantRunner {
    ctx, cancel := context.WithCancel(context.Background())
    return &AssistantRunner{busy: map[account.Login]bool{}, ctx: ctx, cancel: cancel, repository: repository, loader: loader, client: client, settings: settings, planner: planner, downloads: downloads, skill: assistant.NewSkill(), clock: clock, logger: logger}
}

func (r *AssistantRunner) Begin(login account.Login) bool {
    r.mu.Lock()
    defer r.mu.Unlock()
    if r.busy[login] || (r.ctx.Err() != nil) {
        return false
    }
    r.busy[login] = true
    r.wait.Add(1)
    return true
}

func (r *AssistantRunner) Abandon(login account.Login) {
    r.mu.Lock()
    defer r.mu.Unlock()
    delete(r.busy, login)
    r.wait.Done()
}

func (r *AssistantRunner) Running(login account.Login) bool {
    r.mu.Lock()
    defer r.mu.Unlock()
    return r.busy[login]
}

func (r *AssistantRunner) Start(login account.Login, conversation string) {
    go r.run(login, conversation)
}

func (r *AssistantRunner) Stop() {
    r.cancel()
    r.wait.Wait()
}

func (r *AssistantRunner) run(login account.Login, id string) {
    defer r.Abandon(login)
    state, _ := r.repository.AssistantState(login)
    index, found := state.Conversation(id)
    if !found || !state.Conversations[index].Waiting() {
        return
    }
    conversation := state.Conversations[index]
    subscription, key, err := r.repository.SubscriptionKey(login, conversation.Subscription)
    if err != nil {
        r.finish(login, id, assistant.Exchange{Failure: unusable}, nil)
        return
    }
    provider, known := r.repository.Configuration().Provider(subscription.Provider)
    if !known {
        r.finish(login, id, assistant.Exchange{Failure: unknownProvider}, nil)
        return
    }
    ws, found := r.loader.load(&Viewer{Login: login}, login, "/dashboard", false)
    if !found {
        return
    }
    box := NewToolBox(ws, r.planner, r.clock.Now().UTC()).WithExport(NewToolExport(r.downloads, r.settings.DownloadTTL.Std(), r.logger))
    endpoint := assistant.Endpoint{Format: provider.Format, BaseURL: provider.BaseURL, Model: subscription.Model, Key: key}
    agent := assistant.NewAgent(endpoint.Provider(r.client), box.Tools(), r.settings.MaxRounds, r.settings.MaxTokens)
    reply, err := agent.Answer(r.ctx, r.system(ws), r.history(conversation))
    outcome := assistant.Exchange{Provider: provider.Name, Model: subscription.Model}
    if err != nil {
        outcome.Failure = r.failure(err)
        r.logger.Warn("assistant failed", "login", string(login), "provider", provider.Name, "model", subscription.Model, "error", err.Error())
        r.finish(login, id, outcome, nil)
        return
    }
    if reply.Model != "" {
        outcome.Model = reply.Model
    }
    outcome.Answer = r.answer(reply.Text)
    r.logger.Info("assistant answered", "login", string(login), "provider", provider.Name, "model", outcome.Model, "input_tokens", reply.InputTokens, "output_tokens", reply.OutputTokens, "proposal", box.Pending() != nil)
    r.finish(login, id, outcome, box.Pending())
}

func (r *AssistantRunner) history(conversation assistant.Conversation) []assistant.Message {
    exchanges := conversation.Exchanges
    if limit := int(r.settings.Exchanges); len(exchanges) > limit {
        exchanges = exchanges[len(exchanges) - limit:]
    }
    messages := []assistant.Message{}
    for i, exchange := range exchanges {
        if (i < (len(exchanges) - 1)) && (exchange.Answer == "") {
            continue
        }
        messages = append(messages, assistant.Message{Role: assistant.RoleUser, Text: exchange.Question})
        if i < (len(exchanges) - 1) {
            messages = append(messages, assistant.Message{Role: assistant.RoleAssistant, Text: exchange.Answer})
        }
    }
    return messages
}

func (r *AssistantRunner) system(ws *Workspace) string {
    lines := []string{r.skill.Text(), "", "## Context", "", "- Today: " + ws.Today.Label() + " (" + ws.Today.String() + ")", "- User: " + string(ws.Owner.Login)}
    if ws.Owner.Name != "" {
        lines = append(lines, "- Name: " + ws.Owner.Name)
    }
    lines = append(lines, "- Tracking start: " + ws.Calendar.Settings.Start.String(), "- Planning horizon: " + ws.Horizon().String(), "- Activity codes:")
    for _, code := range ws.Config.Activities {
        if !code.Archived {
            lines = append(lines, "  - `" + code.ID + "`: " + code.Label)
        }
    }
    return strings.Join(lines, "\n")
}

func (r *AssistantRunner) answer(text string) string {
    text = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n"))
    var cleaned strings.Builder
    policy := activity.NewTextPolicy()
    for _, letter := range text {
        switch {
            case letter == '\n':
                cleaned.WriteRune(letter)
            case letter == '\t':
                cleaned.WriteString("    ")
            case policy.Accepts(string(letter), 1):
                cleaned.WriteRune(letter)
        }
    }
    text = cleaned.String()
    if len(text) > assistant.MaxAnswerBytes {
        text = strings.ToValidUTF8(text[:assistant.MaxAnswerBytes - 4], "") + "\n\n…"
    }
    return text
}

func (r *AssistantRunner) failure(err error) string {
    refused := &assistant.ProviderError{}
    switch {
        case errors.As(err, &refused) && ((refused.Status == http.StatusUnauthorized) || (refused.Status == http.StatusForbidden)):
            return "Le fournisseur a refusé la clef d'API (statut " + strconv.Itoa(refused.Status) + ") : vérifiez la souscription dans « Mes souscriptions »."
        case errors.As(err, &refused) && (refused.Status == http.StatusTooManyRequests):
            return "Le quota ou le débit autorisé par le fournisseur est dépassé (statut 429) : réessayez plus tard."
        case errors.As(err, &refused):
            return "Le fournisseur a refusé la requête (statut " + strconv.Itoa(refused.Status) + ") : vérifiez le modèle de la souscription."
        case errors.Is(err, context.Canceled):
            return interrupted
        case errors.Is(err, context.DeadlineExceeded):
            return late
    }
    var timeout interface{ Timeout() bool }
    if errors.As(err, &timeout) && timeout.Timeout() {
        return late
    }
    return unreachable
}

func (r *AssistantRunner) finish(login account.Login, id string, outcome assistant.Exchange, pending *assistant.PendingAction) {
    now := r.clock.Now().UTC()
    for range saveAttempts {
        state, _ := r.repository.AssistantState(login)
        index, found := state.Conversation(id)
        if !found || !state.Conversations[index].Waiting() {
            return
        }
        conversation := &state.Conversations[index]
        last := &conversation.Exchanges[len(conversation.Exchanges) - 1]
        last.Answer, last.Failure, last.Provider, last.Model, last.Answered = outcome.Answer, outcome.Failure, outcome.Provider, outcome.Model, activity.NullableTime(now)
        conversation.Updated = now
        if pending != nil {
            conversation.Pending = pending
        }
        err := r.repository.SaveAssistantState(login, state.Revision, state, nil)
        if !errors.Is(err, activity.ErrStaleRevision) {
            if err != nil {
                r.logger.Error("cannot save the answer", "login", string(login), "error", err.Error())
            }
            return
        }
    }
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
