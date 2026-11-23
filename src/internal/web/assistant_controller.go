/*
 * assistant_controller.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "errors"
    "io"
    "log/slog"
    "net/http"
    "slices"
    "strconv"
    "strings"
    "time"

    "timesheet/internal/activity"
    "timesheet/internal/assistant"
)

// ---------------------------------------------------------------------------
// AssistantController declaration
// ---------------------------------------------------------------------------

type AssistantController struct {
    loader     *WorkspaceLoader
    repository Repository
    renderer   *Renderer
    errors     *ErrorPages
    runner     *AssistantRunner
    planner    *ActionPlanner
    settings   activity.AssistantSettings
    clock      activity.Clock
    random     io.Reader
    logger     *slog.Logger
    markdown   *assistant.Markdown
    downloads  Downloads
}

type assistantRecord struct {
    Choices       []SubscriptionChoice
    Conversations []conversationItem
    MaxQuestion   int64
    Question      string
}

type conversationItem struct {
    ID          string
    Path        string
    Title       string
    Updated     string
    UpdatedDate string
    Waiting     bool
}

type conversationRecord struct {
    ID           string
    Title        string
    Subscription string
    Usable       bool
    Exchanges    []exchangeRecord
    Pending      *pendingRecord
    Running      bool
    MaxQuestion  int64
    Question     string
}

type exchangeRecord struct {
    Number    int
    Question  []string
    Answer    []assistant.Block
    Failure   string
    Waiting   bool
    Asked     string
    AskedDate string
    Model     string
}

type pendingRecord struct {
    Summary  []string
    Proposed string
}

const (
    assistantPath      = "/dashboard/assistant"
    answerHeadingLevel = 4
    staleAssistant     = "Les données ont changé entre-temps (autre onglet ?). Rechargez la page avant de recommencer."
    busyAssistant      = "Une réponse est déjà en cours de préparation : attendez-la avant de poser une autre question."
    noSubscription     = "Aucune souscription IA utilisable : ajoutez-en une dans « Mes souscriptions »."
    staleAction        = "L'action ne correspond plus à l'état de vos données : elle n'a pas été appliquée. Annulez-la et demandez une nouvelle proposition."
    timestampLayout    = "02/01/2006 15:04"
)

// ---------------------------------------------------------------------------
// AssistantController implementation
// ---------------------------------------------------------------------------

func NewAssistantController(loader *WorkspaceLoader, repository Repository, renderer *Renderer, errors *ErrorPages, runner *AssistantRunner, planner *ActionPlanner, settings activity.AssistantSettings, clock activity.Clock, random io.Reader, logger *slog.Logger, downloads Downloads) *AssistantController {
    return &AssistantController{loader: loader, repository: repository, renderer: renderer, errors: errors, runner: runner, planner: planner, settings: settings, clock: clock, random: random, logger: logger, markdown: assistant.NewMarkdown(answerHeadingLevel), downloads: downloads}
}

func (c *AssistantController) Show(w http.ResponseWriter, r *http.Request) {
    ws, found := c.loader.Own(r)
    if !found {
        c.errors.NotFound(w, r)
        return
    }
    c.list(w, r, ws, http.StatusOK, "", "")
}

func (c *AssistantController) list(w http.ResponseWriter, r *http.Request, ws *Workspace, status int, alert string, question string) {
    state, _ := c.repository.AssistantState(ws.Owner.Login)
    record := assistantRecord{Choices: NewSubscriptionChoices(c.repository, ws.Owner.Login), MaxQuestion: c.settings.QuestionBytes, Question: question}
    conversations := slices.Clone(state.Conversations)
    slices.SortFunc(conversations, c.recent)
    for _, conversation := range conversations {
        updated := conversation.Updated.In(ws.Location)
        record.Conversations = append(record.Conversations, conversationItem{ID: conversation.ID, Path: assistantPath + "/" + conversation.ID, Title: conversation.Title, Updated: updated.Format(timestampLayout), UpdatedDate: updated.Format(time.RFC3339), Waiting: conversation.Waiting() && c.runner.Running(ws.Owner.Login)})
    }
    if (len(record.Choices) == 0) && (alert == "") {
        alert = noSubscription
    }
    c.renderer.Render(w, r, status, "assistant", PageModel{Section: "assistant", Alert: alert, Data: record})
}

func (c *AssistantController) recent(a assistant.Conversation, b assistant.Conversation) int {
    return b.Updated.Compare(a.Updated)
}

func (c *AssistantController) ShowConversation(w http.ResponseWriter, r *http.Request) {
    ws, found := c.loader.Own(r)
    if !found {
        c.errors.NotFound(w, r)
        return
    }
    state, _ := c.repository.AssistantState(ws.Owner.Login)
    index, known := state.Conversation(r.PathValue("id"))
    if !known {
        c.errors.NotFound(w, r)
        return
    }
    c.conversation(w, r, ws, state.Conversations[index], http.StatusOK, "", "")
}

func (c *AssistantController) conversation(w http.ResponseWriter, r *http.Request, ws *Workspace, conversation assistant.Conversation, status int, alert string, question string) {
    choices := NewSubscriptionChoices(c.repository, ws.Owner.Login)
    choice, usable := (SubscriptionChoice{}).Usable(choices, conversation.Subscription)
    record := conversationRecord{ID: conversation.ID, Title: conversation.Title, Subscription: choice.Label, Usable: usable, Running: c.runner.Running(ws.Owner.Login) && conversation.Waiting(), MaxQuestion: c.settings.QuestionBytes, Question: question}
    if !usable {
        record.Subscription = "souscription supprimée ou qui n'est plus partagée"
    }
    for i, exchange := range conversation.Exchanges {
        asked := exchange.Asked.In(ws.Location)
        item := exchangeRecord{Number: i + 1, Question: strings.Split(exchange.Question, "\n"), Answer: c.markdown.Parse(exchange.Answer), Failure: exchange.Failure, Waiting: exchange.Waiting(), Asked: asked.Format(timestampLayout), AskedDate: asked.Format(time.RFC3339), Model: exchange.Model}
        if item.Waiting && !record.Running {
            item.Waiting, item.Failure = false, interrupted
        }
        record.Exchanges = append(record.Exchanges, item)
    }
    if pending := conversation.Pending; pending != nil {
        record.Pending = &pendingRecord{Summary: pending.Summary, Proposed: pending.Proposed.In(ws.Location).Format(timestampLayout)}
    }
    c.renderer.Render(w, r, status, "assistant_conversation", PageModel{Section: "assistant", Alert: alert, Refresh: record.Running, Data: record})
}

func (c *AssistantController) Download(w http.ResponseWriter, r *http.Request) {
    ws, found := c.loader.Own(r)
    if !found {
        c.errors.NotFound(w, r)
        return
    }
    download, err := c.downloads.Open(ws.Owner.Login, r.PathValue("id"), c.clock.Now(), c.settings.DownloadTTL.Std())
    switch {
        case errors.Is(err, assistant.ErrDownloadExpired):
            c.errors.Status(w, r, http.StatusGone)
            return
        case errors.Is(err, assistant.ErrDownloadMissing):
            c.errors.NotFound(w, r)
            return
        case err != nil:
            c.logger.Error("cannot read the download", "error", err.Error())
            c.errors.Status(w, r, http.StatusInternalServerError)
            return
    }
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.Header().Set("Content-Disposition", `attachment; filename="` + ws.ExportName(activity.NewDateOf(download.Created.In(ws.Location))) + `"`)
    w.Header().Set("Cache-Control", "no-store")
    w.Header().Set("Content-Length", strconv.Itoa(len(download.Data)))
    w.WriteHeader(http.StatusOK)
    w.Write(download.Data)
    c.logger.Info("assistant download", "login", string(ws.Owner.Login))
}

func (c *AssistantController) Submit(w http.ResponseWriter, r *http.Request) {
    ws, found := c.loader.Own(r)
    if !found {
        c.errors.NotFound(w, r)
        return
    }
    state, _ := c.repository.AssistantState(ws.Owner.Login)
    id := r.PostFormValue("conversation")
    index, known := state.Conversation(id)
    if (id != "") && !known {
        c.errors.NotFound(w, r)
        return
    }
    switch r.PostFormValue("action") {
        case "ask":
            c.ask(w, r, ws, state, index)
        case "confirm":
            c.confirm(w, r, ws, state, index)
        case "cancel":
            c.cancel(w, r, ws, state, index)
        case "delete":
            c.remove(w, r, ws, state, index)
        case "delete-all":
            c.removeAll(w, r, ws, state)
        default:
            c.errors.BadRequest(w, r)
    }
}

func (c *AssistantController) refuse(w http.ResponseWriter, r *http.Request, ws *Workspace, state assistant.State, index int, status int, alert string, question string) {
    if index < 0 {
        c.list(w, r, ws, status, alert, question)
        return
    }
    c.conversation(w, r, ws, state.Conversations[index], status, alert, question)
}

func (c *AssistantController) ask(w http.ResponseWriter, r *http.Request, ws *Workspace, state assistant.State, index int) {
    question := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(r.PostFormValue("question"), "\r\n", "\n"), "\r", "\n"))
    switch {
        case question == "":
            c.refuse(w, r, ws, state, index, http.StatusUnprocessableEntity, "Écrivez une question.", question)
            return
        case (int64(len(question)) > c.settings.QuestionBytes) || !state.ValidText(question, activity.MaxQuestionBytes):
            c.refuse(w, r, ws, state, index, http.StatusUnprocessableEntity, "La question est trop longue ou contient des caractères non autorisés.", question)
            return
    }
    choices := NewSubscriptionChoices(c.repository, ws.Owner.Login)
    var ref assistant.SubscriptionRef
    if index < 0 {
        choice, usable := (SubscriptionChoice{}).Find(choices, r.PostFormValue("subscription"))
        if !usable {
            c.refuse(w, r, ws, state, index, http.StatusConflict, noSubscription, question)
            return
        }
        ref = choice.Ref
    } else if _, usable := (SubscriptionChoice{}).Usable(choices, state.Conversations[index].Subscription); usable {
        ref = state.Conversations[index].Subscription
    } else {
        c.refuse(w, r, ws, state, index, http.StatusConflict, unusable, question)
        return
    }
    if !c.runner.Begin(ws.Owner.Login) {
        c.refuse(w, r, ws, state, index, http.StatusConflict, busyAssistant, question)
        return
    }
    now := c.clock.Now().UTC()
    next := state.Clone()
    if index < 0 {
        id, err := activity.NewIdentifierGenerator(c.random).Next()
        if err != nil {
            c.runner.Abandon(ws.Owner.Login)
            c.errors.Status(w, r, http.StatusInternalServerError)
            return
        }
        for int64(len(next.Conversations)) >= c.settings.Conversations {
            next.Conversations = c.withoutOldest(next.Conversations)
        }
        next.Conversations = append(next.Conversations, assistant.Conversation{ID: id, Title: (assistant.Conversation{}).TitleOf(question), Subscription: ref})
        index = len(next.Conversations) - 1
    }
    conversation := &next.Conversations[index]
    if last := len(conversation.Exchanges) - 1; (last >= 0) && conversation.Exchanges[last].Waiting() {
        conversation.Exchanges[last].Failure, conversation.Exchanges[last].Answered = interrupted, activity.NullableTime(now)
    }
    if int64(len(conversation.Exchanges)) >= c.settings.Exchanges {
        conversation.Exchanges = slices.Clone(conversation.Exchanges[int64(len(conversation.Exchanges)) - c.settings.Exchanges + 1:])
    }
    conversation.Exchanges = append(conversation.Exchanges, assistant.Exchange{Question: question, Asked: now})
    conversation.Updated = now
    if !c.save(w, r, ws, state, index, next, question) {
        c.runner.Abandon(ws.Owner.Login)
        return
    }
    c.runner.Start(ws.Owner.Login, conversation.ID)
    c.logger.Info("assistant asked", "login", string(ws.Owner.Login))
    http.Redirect(w, r, assistantPath + "/" + conversation.ID, http.StatusSeeOther)
}

func (c *AssistantController) withoutOldest(conversations []assistant.Conversation) []assistant.Conversation {
    oldest := 0
    for i, conversation := range conversations {
        if conversation.Updated.Before(conversations[oldest].Updated) {
            oldest = i
        }
    }
    return slices.Delete(slices.Clone(conversations), oldest, oldest + 1)
}

func (c *AssistantController) save(w http.ResponseWriter, r *http.Request, ws *Workspace, state assistant.State, index int, next assistant.State, question string) bool {
    err := c.repository.SaveAssistantState(ws.Owner.Login, state.Revision, next, nil)
    switch {
        case err == nil:
            return true
        case errors.Is(err, activity.ErrStaleRevision):
            fresh, _ := c.repository.AssistantState(ws.Owner.Login)
            if index >= 0 {
                if index, found := fresh.Conversation(state.Conversations[index].ID); found {
                    c.conversation(w, r, ws, fresh.Conversations[index], http.StatusConflict, staleAssistant, question)
                    return false
                }
            }
            c.list(w, r, ws, http.StatusConflict, staleAssistant, question)
            return false
    }
    c.logger.Error("cannot save the conversations", "login", string(ws.Owner.Login), "error", err.Error())
    c.errors.Status(w, r, http.StatusInternalServerError)
    return false
}

func (c *AssistantController) confirm(w http.ResponseWriter, r *http.Request, ws *Workspace, state assistant.State, index int) {
    if (index < 0) || (state.Conversations[index].Pending == nil) {
        c.errors.NotFound(w, r)
        return
    }
    pending := state.Conversations[index].Pending
    if pending.Revision != ws.Calendar.Revision {
        c.conversation(w, r, ws, state.Conversations[index], http.StatusConflict, staleAction, "")
        return
    }
    plan, refusal := c.planner.Plan(ws, pending.Tool, pending.Arguments)
    if refusal != "" {
        c.conversation(w, r, ws, state.Conversations[index], http.StatusConflict, "L'action n'a pas été appliquée : " + refusal, "")
        return
    }
    err := c.repository.SaveCalendar(ws.Owner.Login, pending.Revision, plan.Calendar)
    switch {
        case errors.Is(err, activity.ErrStaleRevision):
            c.conversation(w, r, ws, state.Conversations[index], http.StatusConflict, staleAction, "")
            return
        case errors.Is(err, activity.ErrCalendarTooLarge):
            c.conversation(w, r, ws, state.Conversations[index], http.StatusRequestEntityTooLarge, "Les données de ce compte atteignent la taille maximale autorisée.", "")
            return
        case err != nil:
            c.logger.Error("cannot save the calendar", "error", err.Error())
            c.errors.Status(w, r, http.StatusInternalServerError)
            return
    }
    c.logger.Info(plan.Event, "login", string(ws.Owner.Login), "month", plan.Month.String(), "by", "assistant")
    c.close(w, r, ws, state.Conversations[index].ID, "Je confirme l'action proposée.", plan.Done, plan.Notice)
}

func (c *AssistantController) cancel(w http.ResponseWriter, r *http.Request, ws *Workspace, state assistant.State, index int) {
    if (index < 0) || (state.Conversations[index].Pending == nil) {
        c.errors.NotFound(w, r)
        return
    }
    c.close(w, r, ws, state.Conversations[index].ID, "J'annule l'action proposée.", "L'action a été annulée : rien n'a été écrit.", "cancelled")
}

func (c *AssistantController) close(w http.ResponseWriter, r *http.Request, ws *Workspace, id string, question string, answer string, notice string) {
    now := c.clock.Now().UTC()
    for range saveAttempts {
        state, _ := c.repository.AssistantState(ws.Owner.Login)
        index, found := state.Conversation(id)
        if !found {
            break
        }
        conversation := &state.Conversations[index]
        conversation.Pending, conversation.Updated = nil, now
        if int64(len(conversation.Exchanges)) >= c.settings.Exchanges {
            conversation.Exchanges = slices.Clone(conversation.Exchanges[1:])
        }
        conversation.Exchanges = append(conversation.Exchanges, assistant.Exchange{Question: question, Answer: answer, Asked: now, Answered: activity.NullableTime(now)})
        err := c.repository.SaveAssistantState(ws.Owner.Login, state.Revision, state, nil)
        if err == nil {
            break
        }
        if !errors.Is(err, activity.ErrStaleRevision) {
            c.logger.Error("cannot save the conversations", "login", string(ws.Owner.Login), "error", err.Error())
            c.errors.Status(w, r, http.StatusInternalServerError)
            return
        }
    }
    c.renderer.Redirect(w, r, assistantPath + "/" + id, notice)
}

func (c *AssistantController) remove(w http.ResponseWriter, r *http.Request, ws *Workspace, state assistant.State, index int) {
    if index < 0 {
        c.errors.NotFound(w, r)
        return
    }
    next := state.Clone()
    next.Conversations = slices.Delete(next.Conversations, index, index + 1)
    if !c.save(w, r, ws, state, index, next, "") {
        return
    }
    c.logger.Info("conversation deleted", "login", string(ws.Owner.Login))
    c.renderer.Redirect(w, r, assistantPath, "conversation-deleted")
}

func (c *AssistantController) removeAll(w http.ResponseWriter, r *http.Request, ws *Workspace, state assistant.State) {
    next := state.Clone()
    next.Conversations = nil
    if !c.save(w, r, ws, state, -1, next, "") {
        return
    }
    c.logger.Info("conversations deleted", "login", string(ws.Owner.Login))
    c.renderer.Redirect(w, r, assistantPath, "conversations-deleted")
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
