/*
 * auth_guard.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "mime"
    "net/http"

    "timesheet/internal/account"
)

// ---------------------------------------------------------------------------
// AuthGuard declaration
// ---------------------------------------------------------------------------

type AuthGuard struct {
    sessions   *Sessions
    repository Repository
    errors     *ErrorPages
}

type guardedHandler struct {
    guard   *AuthGuard
    profile account.Profile
    next    http.HandlerFunc
}

const (
    settingsPath       = "/dashboard/settings"
    profilePath        = "/dashboard/profile"
    maxMultipartMemory = 8 << 20
)

// ---------------------------------------------------------------------------
// AuthGuard implementation
// ---------------------------------------------------------------------------

func NewAuthGuard(sessions *Sessions, repository Repository, errors *ErrorPages) *AuthGuard {
    return &AuthGuard{sessions: sessions, repository: repository, errors: errors}
}

func (g *AuthGuard) Require(profile account.Profile, next http.HandlerFunc) http.Handler {
    return &guardedHandler{guard: g, profile: profile, next: next}
}

func (g *AuthGuard) Viewer(r *http.Request) *Viewer {
    cookie, err := r.Cookie(g.sessions.CookieName(SessionCookie))
    if err != nil {
        return nil
    }
    login, csrf, key, ok := g.sessions.Lookup(cookie.Value)
    if !ok {
        return nil
    }
    found, _, known := g.repository.Account(login)
    if !known || !found.State.Enabled() || g.sessions.Created(key).Before(found.Created) {
        g.sessions.Close(key)
        return nil
    }
    return &Viewer{Login: login, Name: found.Name, Profile: found.Profile, MustChange: found.MustChangePassword, SessionKey: key, CSRF: csrf}
}

func (g *AuthGuard) Optional(r *http.Request) *http.Request {
    if viewer := g.Viewer(r); viewer != nil {
        return viewer.Attach(r)
    }
    return r
}

func (g *AuthGuard) parse(r *http.Request) error {
    if kind, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); (err == nil) && (kind == "multipart/form-data") {
        return r.ParseMultipartForm(maxMultipartMemory)
    }
    return r.ParseForm()
}

func (g *AuthGuard) ValidToken(r *http.Request, expected string) bool {
    given := r.PostFormValue("csrf")
    return (expected != "") && (subtle.ConstantTimeCompare([]byte(given), []byte(expected)) == 1)
}

func (h *guardedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    viewer := h.guard.Viewer(r)
    if viewer == nil {
        http.Redirect(w, r, "/login", http.StatusSeeOther)
        return
    }
    r = viewer.Attach(r)
    if !viewer.Profile.AtLeast(h.profile) {
        h.guard.errors.Forbidden(w, r, ForbiddenProfile)
        return
    }
    exempt := (r.URL.Path == profilePath) || (r.URL.Path == "/dashboard/colors.css") || (r.URL.Path == "/logout")
    if viewer.MustChange && !exempt {
        http.Redirect(w, r, profilePath, http.StatusSeeOther)
        return
    }
    if r.Method == http.MethodPost {
        if err := h.guard.parse(r); err != nil {
            h.guard.errors.BadRequest(w, r)
            return
        }
        if !h.guard.ValidToken(r, viewer.CSRF) {
            h.guard.errors.Forbidden(w, r, ForbiddenToken)
            return
        }
    }
    h.next(w, r)
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
