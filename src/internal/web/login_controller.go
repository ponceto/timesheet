/*
 * login_controller.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "crypto/subtle"
    "io"
    "log/slog"
    "net/http"
    "regexp"
    "strconv"
    "time"

    "timesheet/internal/account"
    "timesheet/internal/activity"
)

// ---------------------------------------------------------------------------
// LoginController declaration
// ---------------------------------------------------------------------------

type LoginController struct {
    guard      *AuthGuard
    sessions   *Sessions
    repository Repository
    throttle   *LoginThrottle
    address    *ClientAddress
    renderer   *Renderer
    errors     *ErrorPages
    clock      activity.Clock
    decoy      account.PasswordHash
    lifetime   int
    logger     *slog.Logger
}

type loginRecord struct {
    Login string
}

const LoginCookie = "ts_login"

// ---------------------------------------------------------------------------
// LoginController implementation
// ---------------------------------------------------------------------------

func NewLoginController(guard *AuthGuard, sessions *Sessions, repository Repository, throttle *LoginThrottle, address *ClientAddress, renderer *Renderer, errors *ErrorPages, clock activity.Clock, random io.Reader, iterations int, lifetime time.Duration, logger *slog.Logger) (*LoginController, error) {
    decoy, err := account.NewPasswordHash("decoy password never used", iterations, random)
    if err != nil {
        return nil, err
    }
    return &LoginController{guard: guard, sessions: sessions, repository: repository, throttle: throttle, address: address, renderer: renderer, errors: errors, clock: clock, decoy: decoy, lifetime: int(lifetime / time.Second), logger: logger}, nil
}

func (c *LoginController) Show(w http.ResponseWriter, r *http.Request) {
    if c.guard.Viewer(r) != nil {
        http.Redirect(w, r, "/dashboard/", http.StatusSeeOther)
        return
    }
    c.form(w, r, http.StatusOK, "", "")
}

func (c *LoginController) form(w http.ResponseWriter, r *http.Request, status int, login string, alert string) {
    token := ""
    if cookie, err := r.Cookie(c.sessions.CookieName(LoginCookie)); (err == nil) && regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`).MatchString(cookie.Value) {
        token = cookie.Value
    } else if token, err = c.sessions.Token(); err != nil {
        c.errors.Status(w, r, http.StatusInternalServerError)
        return
    }
    http.SetCookie(w, c.sessions.Cookie(LoginCookie, token, c.lifetime))
    c.renderer.Render(w, r, status, "login", PageModel{Section: "login", CSRF: token, Alert: alert, Data: loginRecord{Login: login}})
}

func (c *LoginController) Submit(w http.ResponseWriter, r *http.Request) {
    if err := r.ParseForm(); err != nil {
        c.errors.BadRequest(w, r)
        return
    }
    cookie, err := r.Cookie(c.sessions.CookieName(LoginCookie))
    if (err != nil) || (cookie.Value == "") || (subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(r.PostFormValue("csrf"))) != 1) {
        c.errors.Forbidden(w, r, ForbiddenToken)
        return
    }
    text := r.PostFormValue("login")
    password := r.PostFormValue("password")
    login, valid := account.NewLogin(text)
    key := c.address.Key(r)
    allowed, wait := c.throttle.Begin(key, string(login))
    if !allowed {
        c.logger.Warn("login throttled", "address", key.String())
        w.Header().Set("Retry-After", strconv.Itoa(int(wait.Seconds()) + 1))
        c.errors.Status(w, r, http.StatusTooManyRequests)
        return
    }
    found, _, known := account.Account{}, activity.Calendar{}, false
    if valid == nil {
        found, _, known = c.repository.Account(login)
    }
    hash := c.decoy
    if known {
        hash = found.Password
    }
    if !hash.Verify(password) || !known || !found.State.Enabled() {
        c.throttle.End(key, string(login), false)
        c.logger.Warn("login failed", "address", key.String())
        c.form(w, r, http.StatusUnauthorized, "", "Identifiant ou mot de passe incorrect.")
        return
    }
    c.throttle.End(key, string(login), true)
    previous := ""
    if old, err := r.Cookie(c.sessions.CookieName(SessionCookie)); err == nil {
        previous = old.Value
    }
    id, err := c.sessions.Open(login, previous)
    if err != nil {
        c.errors.Status(w, r, http.StatusInternalServerError)
        return
    }
    if current, _, still := c.repository.Account(login); !still || (current.Password != found.Password) || !current.State.Enabled() {
        c.sessions.Close(c.sessions.Hash(id))
        c.logger.Warn("login failed", "address", key.String())
        c.form(w, r, http.StatusUnauthorized, "", "Identifiant ou mot de passe incorrect.")
        return
    }
    c.repository.TouchLogin(login, c.clock.Now())
    c.logger.Info("login", "login", string(login))
    NewAccessEntry().From(r).Identify(login)
    http.SetCookie(w, c.sessions.Cookie(LoginCookie, "", -1))
    http.SetCookie(w, c.sessions.Cookie(SessionCookie, id, 0))
    http.Redirect(w, r, "/dashboard/", http.StatusSeeOther)
}

func (c *LoginController) ShowLogout(w http.ResponseWriter, r *http.Request) {
    viewer := c.guard.Viewer(r)
    if viewer == nil {
        http.Redirect(w, r, "/login", http.StatusSeeOther)
        return
    }
    c.renderer.Render(w, viewer.Attach(r), http.StatusOK, "logout", PageModel{Section: "logout"})
}

func (c *LoginController) Logout(w http.ResponseWriter, r *http.Request) {
    viewer := (&Viewer{}).From(r)
    c.sessions.Close(viewer.SessionKey)
    c.logger.Info("logout", "login", string(viewer.Login))
    http.SetCookie(w, c.sessions.Cookie(SessionCookie, "", -1))
    c.renderer.Render(w, viewer.Detach(r), http.StatusOK, "logged_out", PageModel{Section: "logout"})
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
