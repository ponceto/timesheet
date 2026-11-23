/*
 * viewer.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "net/http"

    "timesheet/internal/account"
)

// ---------------------------------------------------------------------------
// Viewer declaration
// ---------------------------------------------------------------------------

type Viewer struct {
    Login      account.Login
    Name       string
    Profile    account.Profile
    MustChange bool
    SessionKey string
    CSRF       string
    Assistant  bool
}

type viewerKey struct{}

// ---------------------------------------------------------------------------
// Viewer implementation
// ---------------------------------------------------------------------------

func (v *Viewer) Attach(r *http.Request) *http.Request {
    NewAccessEntry().From(r).Identify(v.Login)
    return r.WithContext(context.WithValue(r.Context(), viewerKey{}, v))
}

func (v *Viewer) Detach(r *http.Request) *http.Request {
    return r.WithContext(context.WithValue(r.Context(), viewerKey{}, (*Viewer)(nil)))
}

func (v *Viewer) From(r *http.Request) *Viewer {
    if viewer, ok := r.Context().Value(viewerKey{}).(*Viewer); ok {
        return viewer
    }
    return nil
}

func (v *Viewer) IsAdmin() bool {
    return (v != nil) && (v.Profile == account.ProfileAdmin)
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
