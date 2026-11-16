/*
 * profile_controller.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "strconv"
    "strings"

    "timesheet/internal/account"
    "timesheet/internal/activity"
)

// ---------------------------------------------------------------------------
// ProfileController declaration
// ---------------------------------------------------------------------------

type ProfileController struct {
    throttle   *LoginThrottle
    address    *ClientAddress
    loader     *WorkspaceLoader
    repository Repository
    sessions   *Sessions
    renderer   *Renderer
    errors     *ErrorPages
    notices    *Notices
    random     io.Reader
    iterations int
    logger     *slog.Logger
}

type profileRecord struct {
    Login         string
    Name          string
    Profile       string
    Created       string
    CreatedDate   string
    LastLogin     string
    LastLoginDate string
    Sessions      int
    MustChange    bool
}

// ---------------------------------------------------------------------------
// ProfileController implementation
// ---------------------------------------------------------------------------

func NewProfileController(throttle *LoginThrottle, address *ClientAddress, loader *WorkspaceLoader, repository Repository, sessions *Sessions, renderer *Renderer, errors *ErrorPages, random io.Reader, iterations int, logger *slog.Logger) *ProfileController {
    return &ProfileController{throttle: throttle, address: address, loader: loader, repository: repository, sessions: sessions, renderer: renderer, errors: errors, notices: NewNotices(), random: random, iterations: iterations, logger: logger}
}

func (c *ProfileController) Show(w http.ResponseWriter, r *http.Request) {
    ws, found := c.loader.Own(r)
    if !found {
        c.errors.NotFound(w, r)
        return
    }
    c.render(w, r, ws, http.StatusOK, "")
}

func (c *ProfileController) render(w http.ResponseWriter, r *http.Request, ws *Workspace, status int, alert string) {
    c.page(w, r, ws, status, PageModel{Alert: alert})
}

func (c *ProfileController) page(w http.ResponseWriter, r *http.Request, ws *Workspace, status int, model PageModel) {
    owner := ws.Owner
    created := activity.NewDateOf(owner.Created.In(ws.Location))
    record := profileRecord{Login: string(owner.Login), Name: owner.Name, Profile: owner.Profile.Label(), Created: created.Numeric(), CreatedDate: created.String(), LastLogin: "jamais", Sessions: c.sessions.Count(owner.Login), MustChange: owner.MustChangePassword}
    if !owner.LastLogin.IsZero() {
        last := activity.NewDateOf(owner.LastLogin.In(ws.Location))
        record.LastLogin, record.LastLoginDate = last.Numeric(), last.String()
    }
    if owner.MustChangePassword && (model.Alert == "") {
        model.Warning = c.notices.Message("change-password")
    }
    model.Section, model.Data = "profile", record
    c.renderer.Render(w, r, status, "profile", model)
}

func (c *ProfileController) Submit(w http.ResponseWriter, r *http.Request) {
    ws, found := c.loader.Own(r)
    if !found {
        c.errors.NotFound(w, r)
        return
    }
    switch r.PostFormValue("action") {
        case "name":
            c.saveName(w, r, ws)
            return
        case "sessions":
            c.closeSessions(w, r, ws)
            return
        case "password":
        default:
            c.errors.BadRequest(w, r)
            return
    }
    current := r.PostFormValue("current")
    next := r.PostFormValue("password")
    key := c.address.Key(r)
    allowed, wait := c.throttle.Begin(key, string(ws.Owner.Login))
    if !allowed {
        w.Header().Set("Retry-After", strconv.Itoa(int(wait.Seconds()) + 1))
        c.errors.Status(w, r, http.StatusTooManyRequests)
        return
    }
    if !ws.Owner.Password.Verify(current) {
        c.throttle.End(key, string(ws.Owner.Login), false)
        c.logger.Warn("password change refused", "login", string(ws.Owner.Login))
        c.render(w, r, ws, http.StatusUnprocessableEntity, "Le mot de passe actuel est incorrect.")
        return
    }
    c.throttle.End(key, string(ws.Owner.Login), true)
    if next != r.PostFormValue("confirm") {
        c.render(w, r, ws, http.StatusUnprocessableEntity, "Les deux saisies du nouveau mot de passe diffèrent.")
        return
    }
    if (next == current) || (account.NewPasswordPolicy().Check(ws.Owner.Login, next) != nil) {
        c.render(w, r, ws, http.StatusUnprocessableEntity, "Le nouveau mot de passe doit différer de l'actuel et compter au moins 12 caractères dont 6 distincts, sans espace au début ni à la fin, et différer de l'identifiant.")
        return
    }
    hash, err := account.NewPasswordHash(next, c.iterations, c.random)
    if err == nil {
        err = c.repository.ChangePassword(ws.Owner.Login, ws.Owner.Password, hash)
    }
    if errors.Is(err, activity.ErrStaleRevision) {
        c.logger.Warn("password change refused", "login", string(ws.Owner.Login))
        c.render(w, r, ws, http.StatusConflict, "Le mot de passe a changé entre-temps : il n'a pas été modifié.")
        return
    }
    if err != nil {
        c.errors.Status(w, r, http.StatusInternalServerError)
        return
    }
    c.sessions.CloseAccount(ws.Owner.Login, "")
    id, err := c.sessions.Open(ws.Owner.Login, "")
    if err != nil {
        c.errors.Status(w, r, http.StatusInternalServerError)
        return
    }
    if current, _, still := c.repository.Account(ws.Owner.Login); !still || (current.Password != hash) || !current.State.Enabled() {
        c.sessions.Close(c.sessions.Hash(id))
        c.logger.Warn("password change refused", "login", string(ws.Owner.Login))
        c.render(w, r, ws, http.StatusConflict, "Le mot de passe a changé entre-temps : il n'a pas été modifié.")
        return
    }
    c.logger.Info("password changed", "login", string(ws.Owner.Login))
    http.SetCookie(w, c.sessions.Cookie(SessionCookie, id, 0))
    c.sessions.SetNotice(c.sessions.Hash(id), "password")
    http.Redirect(w, r, profilePath, http.StatusSeeOther)
}

func (c *ProfileController) closeSessions(w http.ResponseWriter, r *http.Request, ws *Workspace) {
    c.sessions.CloseAccount(ws.Owner.Login, "")
    c.logger.Info("all sessions closed", "login", string(ws.Owner.Login))
    http.SetCookie(w, c.sessions.Cookie(SessionCookie, "", -1))
    c.renderer.Render(w, (&Viewer{}).From(r).Detach(r), http.StatusOK, "logged_out", PageModel{Section: "logout"})
}

func (c *ProfileController) saveName(w http.ResponseWriter, r *http.Request, ws *Workspace) {
    name := strings.Join(strings.Fields(r.PostFormValue("name")), " ")
    if err := c.repository.SetName(ws.Owner.Login, name); err != nil {
        if errors.Is(err, account.ErrInvalidName) {
            c.render(w, r, ws, http.StatusUnprocessableEntity, "Le nom compte au plus 60 caractères, sur une ligne.")
            return
        }
        c.logger.Error("cannot save the name", "error", err.Error())
        c.errors.Status(w, r, http.StatusInternalServerError)
        return
    }
    c.logger.Info("name changed", "login", string(ws.Owner.Login), "by", string(ws.Owner.Login))
    c.renderer.Redirect(w, r, profilePath, "name")
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
