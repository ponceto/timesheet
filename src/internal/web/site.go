/*
 * site.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "net/http"

    "timesheet/internal/account"
)

// ---------------------------------------------------------------------------
// Site declaration
// ---------------------------------------------------------------------------

type Site struct {
    handler  http.Handler
    router   *Router
    sessions *Sessions
}

type notFoundHandler struct {
    guard  *AuthGuard
    errors *ErrorPages
}

// ---------------------------------------------------------------------------
// Site implementation
// ---------------------------------------------------------------------------

func NewSite(deps SiteDependencies) (*Site, error) {
    assets := NewEmbeddedAssets()
    today := NewToday(deps.Clock, deps.Server.Server.Location())
    sessions := NewSessions(deps.Clock, deps.Random, deps.Origins.Secure(), deps.Server.Sessions)
    if deps.Sessions != nil {
        sessions.Restore(deps.Sessions, deps.Logger)
    }
    renderer, err := NewRenderer(assets.Templates(), today, sessions, deps.Logger)
    if err != nil {
        return nil, err
    }
    errors := NewErrorPages(renderer)
    guard := NewAuthGuard(sessions, deps.Repository, errors)
    loader := NewWorkspaceLoader(deps.Repository, today)
    address := NewClientAddress(deps.TrustedProxies, deps.LocalPrefixes)
    throttle := NewLoginThrottle(deps.Clock, deps.Server.Login)
    login, err := NewLoginController(guard, sessions, deps.Repository, throttle, address, renderer, errors, deps.Clock, deps.Random, deps.Iterations, deps.Server.Login.FormLifetime.Std(), deps.Logger)
    if err != nil {
        return nil, err
    }
    home := NewHomeController(guard, renderer)
    dashboard := NewDashboardController(loader, deps.Repository, renderer, errors, deps.Clock, deps.Random, deps.Logger)
    settings := NewSettingsController(loader, deps.Repository, renderer, errors, deps.Logger)
    profile := NewProfileController(throttle, address, loader, deps.Repository, sessions, renderer, errors, deps.Random, deps.Iterations, deps.Logger)
    data := NewDataController(loader, deps.Repository, renderer, errors, deps.Clock, deps.Random, deps.Logger)
    admin := NewAdminController(deps.Repository, sessions, today, renderer, errors, deps.Clock, deps.Random, deps.Iterations, deps.Logger)
    server := NewAdminServerController(deps.Server, deps.Version, renderer)
    configuration := NewAdminSettingsController(deps.Repository, renderer, errors, deps.Logger)
    user := account.ProfileUser
    root := account.ProfileAdmin

    router := NewRouter(&notFoundHandler{guard: guard, errors: errors}, errors)
    router.Handle(http.MethodGet, "/{$}", http.HandlerFunc(home.Show))
    router.Handle(http.MethodGet, "/login", http.HandlerFunc(login.Show))
    router.Handle(http.MethodPost, "/login", http.HandlerFunc(login.Submit))
    router.Handle(http.MethodGet, "/logout", http.HandlerFunc(login.ShowLogout))
    router.Handle(http.MethodPost, "/logout", guard.Require(user, login.Logout))
    router.Handle(http.MethodGet, "/dashboard/{$}", guard.Require(user, dashboard.Show))
    router.Handle(http.MethodGet, "/dashboard/activity", guard.Require(user, dashboard.CurrentActivity))
    router.Handle(http.MethodGet, "/dashboard/activity/{year}/{month}", guard.Require(user, dashboard.ShowActivity))
    router.Handle(http.MethodPost, "/dashboard/activity/{year}/{month}", guard.Require(user, dashboard.Save))
    router.Handle(http.MethodPost, "/dashboard/activity/{year}/{month}/validate", guard.Require(user, dashboard.Validate))
    router.Handle(http.MethodPost, "/dashboard/activity/{year}/{month}/reopen", guard.Require(user, dashboard.Reopen))
    router.Handle(http.MethodGet, "/dashboard/month", guard.Require(user, dashboard.CurrentMonth))
    router.Handle(http.MethodGet, "/dashboard/month/{year}/{month}", guard.Require(user, dashboard.ShowMonth))
    router.Handle(http.MethodGet, "/dashboard/year/{year}", guard.Require(user, dashboard.Year))
    router.Handle(http.MethodPost, "/dashboard/adjustments", guard.Require(user, dashboard.Adjust))
    router.Handle(http.MethodGet, "/dashboard/profile", guard.Require(user, profile.Show))
    router.Handle(http.MethodPost, "/dashboard/profile", guard.Require(user, profile.Submit))
    router.Handle(http.MethodGet, dataPath, guard.Require(user, data.Show))
    router.Handle(http.MethodGet, dataExportPath, guard.Require(user, data.Export))
    router.Handle(http.MethodPost, dataImportPath, guard.Require(user, data.Import))
    router.Handle(http.MethodPost, dataResetPath, guard.Require(user, data.Reset))
    router.Handle(http.MethodGet, "/dashboard/settings", guard.Require(user, settings.Show))
    router.Handle(http.MethodPost, "/dashboard/settings", guard.Require(user, settings.Submit))
    router.Handle(http.MethodGet, "/dashboard/colors.css", guard.Require(user, dashboard.Colors))
    router.Handle(http.MethodGet, "/admin/{$}", guard.Require(root, server.Show))
    router.Handle(http.MethodGet, "/admin/accounts", guard.Require(root, admin.List))
    router.Handle(http.MethodPost, "/admin/accounts", guard.Require(root, admin.Create))
    router.Handle(http.MethodGet, "/admin/accounts/{login}", guard.Require(root, admin.Show))
    router.Handle(http.MethodPost, "/admin/accounts/{login}", guard.Require(root, admin.Update))
    router.Handle(http.MethodGet, "/admin/accounts/{login}/dashboard", guard.Require(root, dashboard.ShowBalancesOther))
    router.Handle(http.MethodGet, "/admin/accounts/{login}/month/{year}/{month}", guard.Require(root, dashboard.ShowOther))
    router.Handle(http.MethodGet, "/admin/accounts/{login}/year/{year}", guard.Require(root, dashboard.YearOther))
    router.Handle(http.MethodGet, "/admin/settings", guard.Require(root, configuration.Show))
    router.Handle(http.MethodPost, "/admin/settings", guard.Require(root, configuration.Submit))
    router.Handle(http.MethodGet, "/static/{path...}", http.HandlerFunc(NewAssetServer(assets.Static(), errors).ServeStatic))

    var handler http.Handler = router
    handler = NewOriginGuard(handler, deps.Origins, errors)
    handler = NewRequestLimits(handler, errors, int(deps.Server.Server.MaxURLBytes), deps.Server.Server.MaxBodyBytes)
    handler = NewHostGuard(handler, deps.Origins, errors)
    handler = NewRecoveryHandler(handler, errors, deps.Logger)
    handler = NewSecurityHeaders(handler, deps.Origins.Secure(), deps.Server.Server.HSTSMaxAge.Std())
    handler = NewAccessLog(handler, address, deps.Clock, deps.Logger)
    return &Site{handler: handler, router: router, sessions: sessions}, nil
}

func (h *notFoundHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    h.errors.NotFound(w, h.guard.Optional(r))
}

func (s *Site) Handler() http.Handler {
    return s.handler
}

func (s *Site) Flush() {
    s.sessions.Flush()
}

func (s *Site) Routes() []Route {
    return s.router.Routes()
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
