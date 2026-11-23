/*
 * subscription_controller.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "encoding/json"
    "errors"
    "io"
    "log/slog"
    "net/http"
    "slices"
    "strconv"
    "strings"

    "timesheet/internal/account"
    "timesheet/internal/activity"
    "timesheet/internal/assistant"
)

// ---------------------------------------------------------------------------
// SubscriptionController declaration
// ---------------------------------------------------------------------------

type SubscriptionController struct {
    loader     *WorkspaceLoader
    repository Repository
    renderer   *Renderer
    errors     *ErrorPages
    clock      activity.Clock
    random     io.Reader
    logger     *slog.Logger
}

type subscriptionsRecord struct {
    Own    []ownSubscriptionRecord
    Shared []sharedSubscriptionRecord
    Groups []providerGroup
    Full   bool
}

type ownSubscriptionRecord struct {
    ID       string
    Name     string
    Provider string
    Model    string
    Default  bool
    Shared   []string
    Groups   []providerGroup
}

type sharedSubscriptionRecord struct {
    Owner    string
    ID       string
    Label    string
    Provider string
    Model    string
    Hidden   bool
}

type providerGroup struct {
    Label   string
    Options []modelOption
}

type modelOption struct {
    Value    string
    Label    string
    Selected bool
}

const (
    choiceSeparator     = "|"
    subscriptionsPath   = "/dashboard/subscriptions"
    minLoginQuery       = 2
    maxLoginQuery       = 32
    maxLoginSuggestions = 10
    unknownModel        = "Choisissez un fournisseur et un modèle de la liste, ou un autre modèle de 1 à 100 caractères imprimables sans espace."
    invalidName         = "Le nom compte au plus 60 caractères, sur une ligne."
    invalidKey          = "Indiquez la clef d'API : une ligne de 512 caractères au plus."
    tooManyOwn          = "Vous avez atteint le nombre maximal de souscriptions."
    unknownShare        = "Partage impossible : indiquez l'identifiant d'un autre compte existant, qui n'a pas déjà accès à cette souscription."
    tooManyShares       = "Cette souscription est déjà partagée avec le nombre maximal de comptes."
    staleProfile        = "Les données ont changé entre-temps (autre onglet ?). Rechargez la page avant de recommencer."
)

// ---------------------------------------------------------------------------
// SubscriptionController implementation
// ---------------------------------------------------------------------------

func NewSubscriptionController(loader *WorkspaceLoader, repository Repository, renderer *Renderer, errors *ErrorPages, clock activity.Clock, random io.Reader, logger *slog.Logger) *SubscriptionController {
    return &SubscriptionController{loader: loader, repository: repository, renderer: renderer, errors: errors, clock: clock, random: random, logger: logger}
}

func (c *SubscriptionController) Show(w http.ResponseWriter, r *http.Request) {
    ws, found := c.loader.Own(r)
    if !found {
        c.errors.NotFound(w, r)
        return
    }
    c.render(w, r, ws, http.StatusOK, "")
}

func (c *SubscriptionController) render(w http.ResponseWriter, r *http.Request, ws *Workspace, status int, alert string) {
    c.renderer.Render(w, r, status, "subscriptions", PageModel{Section: "subscriptions", Alert: alert, Data: c.Record(ws)})
}

func (c *SubscriptionController) Submit(w http.ResponseWriter, r *http.Request) {
    ws, found := c.loader.Own(r)
    if !found {
        c.errors.NotFound(w, r)
        return
    }
    outcome := c.apply(r, ws)
    switch {
        case outcome.Alert != "":
            c.render(w, r, ws, outcome.Status, outcome.Alert)
        case outcome.Status == http.StatusBadRequest:
            c.errors.BadRequest(w, r)
        case outcome.Status == http.StatusNotFound:
            c.errors.NotFound(w, r)
        case outcome.Status != 0:
            c.errors.Status(w, r, outcome.Status)
        default:
            c.renderer.Redirect(w, r, subscriptionsPath, outcome.Notice)
    }
}

func (c *SubscriptionController) Logins(w http.ResponseWriter, r *http.Request) {
    viewer := (&Viewer{}).From(r)
    query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
    logins := []string{}
    if (viewer != nil) && (len(query) >= minLoginQuery) && (len(query) <= maxLoginQuery) {
        for _, item := range c.repository.Accounts() {
            if (item.Login != viewer.Login) && item.State.Enabled() && strings.Contains(string(item.Login), query) && (len(logins) < maxLoginSuggestions) {
                logins = append(logins, string(item.Login))
            }
        }
    }
    slices.Sort(logins)
    data, _ := json.Marshal(logins)
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.Header().Set("Content-Length", strconv.Itoa(len(data)))
    w.WriteHeader(http.StatusOK)
    w.Write(data)
}

func (c *SubscriptionController) Record(ws *Workspace) subscriptionsRecord {
    state, _ := c.repository.AssistantState(ws.Owner.Login)
    record := subscriptionsRecord{Groups: c.groups(ws.Config, "", ""), Full: len(state.Subscriptions) >= assistant.MaxSubscriptions}
    for i, subscription := range state.Subscriptions {
        item := ownSubscriptionRecord{ID: subscription.ID, Name: subscription.Name, Provider: c.label(ws.Config, subscription.Provider), Model: subscription.Model, Default: i == 0, Groups: c.groups(ws.Config, subscription.Provider, subscription.Model)}
        for _, login := range subscription.SharedWith {
            item.Shared = append(item.Shared, string(login))
        }
        record.Own = append(record.Own, item)
    }
    for _, shared := range c.repository.SharedSubscriptions(ws.Owner.Login) {
        owner := string(shared.Owner)
        if shared.OwnerName != "" {
            owner = shared.OwnerName + " (" + owner + ")"
        }
        record.Shared = append(record.Shared, sharedSubscriptionRecord{Owner: owner, ID: string(shared.Owner) + ":" + shared.Subscription.ID, Label: shared.Subscription.Name, Provider: c.label(ws.Config, shared.Subscription.Provider), Model: shared.Subscription.Model, Hidden: shared.Hidden})
    }
    return record
}

func (c *SubscriptionController) label(config activity.Configuration, name string) string {
    if provider, found := config.Provider(name); found {
        return provider.Label
    }
    return name
}

func (c *SubscriptionController) groups(config activity.Configuration, selectedProvider string, selectedModel string) []providerGroup {
    groups := []providerGroup{}
    for _, provider := range config.Providers() {
        group := providerGroup{Label: provider.Label}
        models := slices.Clone(provider.Models)
        if !slices.Contains(models, provider.DefaultModel) {
            models = append([]string{provider.DefaultModel}, models...)
        }
        for _, model := range models {
            selected := (provider.Name == selectedProvider) && (model == selectedModel)
            if selectedProvider == "" {
                selected = (len(groups) == 0) && (model == provider.DefaultModel)
            }
            group.Options = append(group.Options, modelOption{Value: provider.Name + choiceSeparator + model, Label: model, Selected: selected})
        }
        groups = append(groups, group)
    }
    return groups
}

func (c *SubscriptionController) apply(r *http.Request, ws *Workspace) SubscriptionOutcome {
    state, _ := c.repository.AssistantState(ws.Owner.Login)
    next := state.Clone()
    keys := map[string]string{}
    var outcome SubscriptionOutcome
    switch action := r.PostFormValue("action"); action {
        case "subscription-add":
            outcome = c.add(r, ws, &next, keys)
        case "subscription-edit":
            outcome = c.edit(r, ws, &next, keys)
        case "subscription-delete":
            outcome = c.own(r, &next, c.delete)
        case "subscription-default":
            outcome = c.own(r, &next, c.prefer)
        case "subscription-share":
            outcome = c.share(r, ws, &next)
        case "subscription-unshare":
            outcome = c.unshare(r, &next)
        case "subscription-hide", "subscription-show":
            outcome = c.hide(r, ws, &next, action == "subscription-hide")
        default:
            return SubscriptionOutcome{Status: http.StatusBadRequest}
    }
    if outcome.Status != 0 {
        return outcome
    }
    err := c.repository.SaveAssistantState(ws.Owner.Login, state.Revision, next, keys)
    switch {
        case errors.Is(err, activity.ErrStaleRevision):
            return SubscriptionOutcome{Status: http.StatusConflict, Alert: staleProfile}
        case errors.Is(err, assistant.ErrInvalidKey):
            return SubscriptionOutcome{Status: http.StatusUnprocessableEntity, Alert: invalidKey}
        case errors.Is(err, assistant.ErrInvalidState):
            return SubscriptionOutcome{Status: http.StatusUnprocessableEntity, Alert: unknownShare}
        case err != nil:
            c.logger.Error("cannot save the subscriptions", "login", string(ws.Owner.Login), "error", err.Error())
            return SubscriptionOutcome{Status: http.StatusInternalServerError}
    }
    c.logger.Info("subscriptions changed", "login", string(ws.Owner.Login), "change", outcome.Notice)
    return outcome
}

func (c *SubscriptionController) fields(r *http.Request, ws *Workspace, subscription *assistant.Subscription) string {
    provider, model, _ := strings.Cut(r.PostFormValue("model"), choiceSeparator)
    if custom := strings.TrimSpace(r.PostFormValue("custom_model")); custom != "" {
        model = custom
    }
    known, found := ws.Config.Provider(provider)
    if !found || !known.ValidModel(model) {
        return unknownModel
    }
    name := strings.Join(strings.Fields(r.PostFormValue("name")), " ")
    if name == "" {
        name = known.Label + " " + model
        if runes := []rune(name); len(runes) > assistant.MaxSubscriptionName {
            name = strings.TrimSpace(string(runes[:assistant.MaxSubscriptionName]))
        }
    }
    if !(assistant.State{}).ValidLabel(name, assistant.MaxSubscriptionName) {
        return invalidName
    }
    subscription.Name, subscription.Provider, subscription.Model = name, provider, model
    return ""
}

func (c *SubscriptionController) key(r *http.Request, required bool) (string, bool) {
    key := strings.TrimSpace(r.PostFormValue("key"))
    if key == "" {
        return "", !required
    }
    return key, (len(key) <= assistant.MaxKeyLength) && activity.NewTextPolicy().Accepts(key, assistant.MaxKeyLength)
}

func (c *SubscriptionController) add(r *http.Request, ws *Workspace, state *assistant.State, keys map[string]string) SubscriptionOutcome {
    if len(state.Subscriptions) >= assistant.MaxSubscriptions {
        return SubscriptionOutcome{Status: http.StatusUnprocessableEntity, Alert: tooManyOwn}
    }
    subscription := assistant.Subscription{Created: c.clock.Now().UTC()}
    if alert := c.fields(r, ws, &subscription); alert != "" {
        return SubscriptionOutcome{Status: http.StatusUnprocessableEntity, Alert: alert}
    }
    key, valid := c.key(r, true)
    if !valid {
        return SubscriptionOutcome{Status: http.StatusUnprocessableEntity, Alert: invalidKey}
    }
    id, err := activity.NewIdentifierGenerator(c.random).Next()
    if err != nil {
        return SubscriptionOutcome{Status: http.StatusInternalServerError}
    }
    subscription.ID = id
    state.Subscriptions = append(state.Subscriptions, subscription)
    keys[id] = key
    return SubscriptionOutcome{Notice: "subscription"}
}

func (c *SubscriptionController) edit(r *http.Request, ws *Workspace, state *assistant.State, keys map[string]string) SubscriptionOutcome {
    index, found := state.Subscription(r.PostFormValue("id"))
    if !found {
        return SubscriptionOutcome{Status: http.StatusNotFound}
    }
    if alert := c.fields(r, ws, &state.Subscriptions[index]); alert != "" {
        return SubscriptionOutcome{Status: http.StatusUnprocessableEntity, Alert: alert}
    }
    key, valid := c.key(r, false)
    if !valid {
        return SubscriptionOutcome{Status: http.StatusUnprocessableEntity, Alert: invalidKey}
    }
    if key != "" {
        keys[state.Subscriptions[index].ID] = key
    }
    return SubscriptionOutcome{Notice: "subscription"}
}

func (c *SubscriptionController) own(r *http.Request, state *assistant.State, change func(*assistant.State, int) string) SubscriptionOutcome {
    index, found := state.Subscription(r.PostFormValue("id"))
    if !found {
        return SubscriptionOutcome{Status: http.StatusNotFound}
    }
    return SubscriptionOutcome{Notice: change(state, index)}
}

func (c *SubscriptionController) delete(state *assistant.State, index int) string {
    state.Subscriptions = slices.Delete(state.Subscriptions, index, index + 1)
    return "subscription-deleted"
}

func (c *SubscriptionController) prefer(state *assistant.State, index int) string {
    preferred := state.Subscriptions[index]
    state.Subscriptions = append([]assistant.Subscription{preferred}, slices.Delete(state.Subscriptions, index, index + 1)...)
    return "subscription"
}

func (c *SubscriptionController) share(r *http.Request, ws *Workspace, state *assistant.State) SubscriptionOutcome {
    index, found := state.Subscription(r.PostFormValue("id"))
    if !found {
        return SubscriptionOutcome{Status: http.StatusNotFound}
    }
    login := account.Login(strings.TrimSpace(r.PostFormValue("login")))
    subscription := &state.Subscriptions[index]
    if len(subscription.SharedWith) >= assistant.MaxShares {
        return SubscriptionOutcome{Status: http.StatusUnprocessableEntity, Alert: tooManyShares}
    }
    if _, _, exists := c.repository.Account(login); !login.Valid() || !exists || (login == ws.Owner.Login) || subscription.SharedTo(login) {
        return SubscriptionOutcome{Status: http.StatusUnprocessableEntity, Alert: unknownShare}
    }
    subscription.SharedWith = append(subscription.SharedWith, login)
    return SubscriptionOutcome{Notice: "shared"}
}

func (c *SubscriptionController) unshare(r *http.Request, state *assistant.State) SubscriptionOutcome {
    index, found := state.Subscription(r.PostFormValue("id"))
    login := account.Login(r.PostFormValue("login"))
    if !found || !state.Subscriptions[index].SharedTo(login) {
        return SubscriptionOutcome{Status: http.StatusNotFound}
    }
    state.Subscriptions[index].SharedWith = slices.DeleteFunc(state.Subscriptions[index].SharedWith, login.Equal)
    return SubscriptionOutcome{Notice: "unshared"}
}

func (c *SubscriptionController) hide(r *http.Request, ws *Workspace, state *assistant.State, hidden bool) SubscriptionOutcome {
    owner, id, _ := strings.Cut(r.PostFormValue("id"), ":")
    ref := assistant.SubscriptionRef{Owner: account.Login(owner), ID: id}
    shared := false
    for _, item := range c.repository.SharedSubscriptions(ws.Owner.Login) {
        shared = shared || (item.Ref() == ref)
    }
    if !shared {
        return SubscriptionOutcome{Status: http.StatusNotFound}
    }
    state.Hidden = slices.DeleteFunc(state.Hidden, ref.Equal)
    if hidden {
        state.Hidden = append(state.Hidden, ref)
        return SubscriptionOutcome{Notice: "subscription-hidden"}
    }
    return SubscriptionOutcome{Notice: "subscription-shown"}
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
