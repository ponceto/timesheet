/*
 * admin_controller.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "time"

    "timesheet/internal/account"
    "timesheet/internal/activity"
)

// ---------------------------------------------------------------------------
// AdminController declaration
// ---------------------------------------------------------------------------

type AdminController struct {
    repository Repository
    sessions   *Sessions
    today      *Today
    renderer   *Renderer
    errors     *ErrorPages
    clock      activity.Clock
    random     io.Reader
    iterations int
    logger     *slog.Logger
    downloads  Downloads
}

type accountsRecord struct {
    Accounts []accountRecord
    Login    string
    Profile  string
}

type accountRecord struct {
    Login         string
    Name          string
    Disabled      bool
    State         string
    Profile       string
    Admin         bool
    Created       string
    CreatedDate   string
    Identity      string
    LastLogin     string
    LastLoginDate string
    Path          string
    Balances      string
    Dashboard     string
    Year          string
    Self          bool
    Sessions      int
}

// ---------------------------------------------------------------------------
// AdminController implementation
// ---------------------------------------------------------------------------

func NewAdminController(repository Repository, sessions *Sessions, today *Today, renderer *Renderer, errors *ErrorPages, clock activity.Clock, random io.Reader, iterations int, logger *slog.Logger, downloads Downloads) *AdminController {
    return &AdminController{repository: repository, sessions: sessions, today: today, renderer: renderer, errors: errors, clock: clock, random: random, iterations: iterations, logger: logger, downloads: downloads}
}

func (c *AdminController) record(viewer *Viewer, item account.Account) accountRecord {
    base := "/admin/accounts/" + string(item.Login)
    today := c.today.Date()
    year := strconv.Itoa(today.Year())
    location := c.today.Location()
    created := activity.NewDateOf(item.Created.In(location))
    record := accountRecord{Login: string(item.Login), Name: item.Name, Disabled: !item.State.Enabled(), State: item.State.Label(), Profile: item.Profile.Label(), Admin: item.Profile == account.ProfileAdmin, Created: created.Numeric(), CreatedDate: created.String(), Identity: c.identity(item), LastLogin: "jamais", Path: base, Balances: base + "/dashboard", Dashboard: base + "/month/" + today.Month().Path(), Year: base + "/year/" + year, Self: item.Login == viewer.Login, Sessions: c.sessions.Count(item.Login)}
    if !item.LastLogin.IsZero() {
        last := activity.NewDateOf(item.LastLogin.In(location))
        record.LastLogin, record.LastLoginDate = last.Numeric(), last.String()
    }
    return record
}

func (c *AdminController) List(w http.ResponseWriter, r *http.Request) {
    c.list(w, r, http.StatusOK, "", "", "")
}

func (c *AdminController) list(w http.ResponseWriter, r *http.Request, status int, alert string, login string, profile string) {
    viewer := (&Viewer{}).From(r)
    record := accountsRecord{Login: login, Profile: profile}
    for _, item := range c.repository.Accounts() {
        record.Accounts = append(record.Accounts, c.record(viewer, item))
    }
    c.renderer.Render(w, r, status, "admin_accounts", PageModel{Section: "accounts", Alert: alert, Data: record})
}

func (c *AdminController) Create(w http.ResponseWriter, r *http.Request) {
    text := strings.TrimSpace(r.PostFormValue("login"))
    login, err := account.NewLogin(text)
    if err != nil {
        c.list(w, r, http.StatusUnprocessableEntity, "Identifiant invalide : 2 à 32 caractères parmi a-z, 0-9, point, tiret et souligné, commençant par une lettre ou un chiffre.", text, r.PostFormValue("profile"))
        return
    }
    profile, err := account.NewProfile(r.PostFormValue("profile"))
    if err != nil {
        c.list(w, r, http.StatusUnprocessableEntity, "Profil invalide.", text, "")
        return
    }
    password := r.PostFormValue("password")
    if account.NewPasswordPolicy().Check(login, password) != nil {
        c.list(w, r, http.StatusUnprocessableEntity, "Mot de passe provisoire trop faible : au moins 12 caractères dont 6 distincts, sans espace au début ni à la fin, différent de l'identifiant.", text, string(profile))
        return
    }
    hash, err := account.NewPasswordHash(password, c.iterations, c.random)
    if err != nil {
        c.errors.Status(w, r, http.StatusInternalServerError)
        return
    }
    created := account.Account{Login: login, Profile: profile, State: account.StateEnabled, Password: hash, MustChangePassword: true, Created: c.clock.Now().UTC().Truncate(time.Second)}
    if err := c.repository.CreateAccount(created, activity.NewCalendar(c.today.Date().Month().First())); err != nil {
        alert := "Création impossible."
        c.logger.Warn("account creation refused", "login", string(login), "error", err.Error())
        switch {
            case errors.Is(err, account.ErrDuplicateLogin):
                alert = "Cet identifiant est déjà pris."
            case errors.Is(err, account.ErrTooManyAccounts):
                alert = "Le nombre maximal de comptes est atteint."
        }
        c.list(w, r, http.StatusUnprocessableEntity, alert, text, string(profile))
        return
    }
    c.logger.Info("account created", "login", string(login), "profile", string(profile), "by", string((&Viewer{}).From(r).Login))
    c.renderer.Redirect(w, r, "/admin/accounts", "created")
}

func (c *AdminController) Show(w http.ResponseWriter, r *http.Request) {
    c.show(w, r, http.StatusOK, "")
}

func (c *AdminController) show(w http.ResponseWriter, r *http.Request, status int, alert string) {
    if r.PostFormValue("return") == "list" {
        c.list(w, r, status, alert, "", "")
        return
    }
    item, _, found := c.repository.Account(account.Login(r.PathValue("login")))
    if !found {
        c.errors.NotFound(w, r)
        return
    }
    c.renderer.Render(w, r, status, "admin_account", PageModel{Section: "accounts", Alert: alert, Data: c.record((&Viewer{}).From(r), item)})
}

func (c *AdminController) Update(w http.ResponseWriter, r *http.Request) {
    viewer := (&Viewer{}).From(r)
    login := account.Login(r.PathValue("login"))
    item, _, found := c.repository.Account(login)
    if !found {
        c.errors.NotFound(w, r)
        return
    }
    if r.PostFormValue("created") != c.identity(item) {
        c.show(w, r, http.StatusConflict, "Ce compte a été supprimé puis recréé entre-temps. Rechargez la page avant de recommencer.")
        return
    }
    switch r.PostFormValue("action") {
        case "profile":
            profile, err := account.NewProfile(r.PostFormValue("profile"))
            if (err != nil) || (login == viewer.Login) {
                c.show(w, r, http.StatusUnprocessableEntity, "Profil invalide, ou changement de son propre profil.")
                return
            }
            if err := c.repository.SetProfile(login, profile); err != nil {
                c.show(w, r, http.StatusConflict, c.message(err))
                return
            }
            c.sessions.CloseAccount(login, "")
            c.logger.Info("profile changed", "login", string(login), "profile", string(profile), "by", string(viewer.Login))
            c.renderer.Redirect(w, r, "/admin/accounts/" + string(login), "profile")
        case "reset":
            if login == viewer.Login {
                c.show(w, r, http.StatusUnprocessableEntity, "Changez votre propre mot de passe depuis votre profil.")
                return
            }
            password := r.PostFormValue("password")
            if account.NewPasswordPolicy().Check(login, password) != nil {
                c.show(w, r, http.StatusUnprocessableEntity, "Mot de passe provisoire trop faible : au moins 12 caractères dont 6 distincts, sans espace au début ni à la fin, différent de l'identifiant.")
                return
            }
            hash, err := account.NewPasswordHash(password, c.iterations, c.random)
            if err != nil {
                c.errors.Status(w, r, http.StatusInternalServerError)
                return
            }
            if err := c.repository.SetPassword(login, hash, true); err != nil {
                c.show(w, r, http.StatusConflict, c.message(err))
                return
            }
            c.sessions.CloseAccount(login, "")
            c.logger.Info("password reset", "login", string(login), "by", string(viewer.Login))
            c.renderer.Redirect(w, r, "/admin/accounts/" + string(login), "reset")
        case "name":
            name := strings.Join(strings.Fields(r.PostFormValue("name")), " ")
            if err := c.repository.SetName(login, name); err != nil {
                if errors.Is(err, account.ErrInvalidName) {
                    c.show(w, r, http.StatusUnprocessableEntity, "Le nom compte au plus 60 caractères, sur une ligne.")
                    return
                }
                c.show(w, r, http.StatusConflict, c.message(err))
                return
            }
            c.logger.Info("name changed", "login", string(login), "by", string(viewer.Login))
            c.renderer.Redirect(w, r, "/admin/accounts/" + string(login), "name")
        case "disable", "enable":
            state, notice := account.StateEnabled, "enabled"
            if r.PostFormValue("action") == "disable" {
                state, notice = account.StateDisabled, "disabled"
            }
            if login == viewer.Login {
                c.show(w, r, http.StatusUnprocessableEntity, "Un administrateur ne peut pas désactiver son propre compte.")
                return
            }
            if err := c.repository.SetState(login, state); err != nil {
                c.show(w, r, http.StatusConflict, c.message(err))
                return
            }
            if !state.Enabled() {
                c.sessions.CloseAccount(login, "")
            }
            c.logger.Info("account " + notice, "login", string(login), "by", string(viewer.Login))
            c.renderer.Redirect(w, r, c.target(r, login), notice)
        case "sessions":
            c.sessions.CloseAccount(login, "")
            c.logger.Info("sessions closed", "login", string(login), "by", string(viewer.Login))
            if login == viewer.Login {
                http.SetCookie(w, c.sessions.Cookie(SessionCookie, "", -1))
                c.renderer.Render(w, viewer.Detach(r), http.StatusOK, "logged_out", PageModel{Section: "logout"})
                return
            }
            c.renderer.Redirect(w, r, c.target(r, login), "sessions")
        case "delete":
            if (login == viewer.Login) || (r.PostFormValue("confirm") != string(item.Login)) {
                c.show(w, r, http.StatusUnprocessableEntity, "Retapez exactement l'identifiant pour confirmer ; un administrateur ne peut pas supprimer son propre compte.")
                return
            }
            if err := c.repository.DeleteAccount(login); err != nil {
                c.show(w, r, http.StatusConflict, c.message(err))
                return
            }
            c.sessions.CloseAccount(login, "")
            c.downloads.RemoveAccount(login)
            c.logger.Info("account deleted", "login", string(login), "by", string(viewer.Login))
            c.renderer.Redirect(w, r, "/admin/accounts", "deleted")
        default:
            c.errors.BadRequest(w, r)
    }
}

func (c *AdminController) identity(item account.Account) string {
    return strconv.FormatInt(item.Created.Unix(), 10)
}

func (c *AdminController) target(r *http.Request, login account.Login) string {
    if r.PostFormValue("return") == "list" {
        return "/admin/accounts"
    }
    return "/admin/accounts/" + string(login)
}

func (c *AdminController) message(err error) string {
    if errors.Is(err, account.ErrLastAdmin) {
        return "Le dernier administrateur actif ne peut être ni supprimé, ni rétrogradé, ni désactivé."
    }
    return "Opération impossible."
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
