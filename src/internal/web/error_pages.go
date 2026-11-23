/*
 * error_pages.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "strings"
)

// ---------------------------------------------------------------------------
// ErrorPages declaration
// ---------------------------------------------------------------------------

type ErrorPages struct {
    renderer *Renderer
}

type errorRecord struct {
    Status int
    Title  string
    Text   string
}

const (
    ForbiddenOrigin  = "Cette requête vient d'une autre origine et a été refusée."
    ForbiddenProfile = "Cette page est réservée aux administrateurs."
    ForbiddenToken   = "Le formulaire a expiré ou n'est pas valide. Rechargez la page et recommencez."
)

// ---------------------------------------------------------------------------
// ErrorPages implementation
// ---------------------------------------------------------------------------

func NewErrorPages(renderer *Renderer) *ErrorPages {
    return &ErrorPages{renderer: renderer}
}

func (e *ErrorPages) NotFound(w http.ResponseWriter, r *http.Request) {
    e.renderer.Render(w, r, http.StatusNotFound, "not_found", PageModel{})
}

func (e *ErrorPages) Forbidden(w http.ResponseWriter, r *http.Request, message string) {
    e.renderer.Render(w, r, http.StatusForbidden, "forbidden", PageModel{Alert: message})
}

func (e *ErrorPages) MethodNotAllowed(w http.ResponseWriter, r *http.Request, allowed []string) {
    w.Header().Set("Allow", strings.Join(allowed, ", "))
    e.Status(w, r, http.StatusMethodNotAllowed)
}

func (e *ErrorPages) BadRequest(w http.ResponseWriter, r *http.Request) {
    e.Status(w, r, http.StatusBadRequest)
}

func (e *ErrorPages) Status(w http.ResponseWriter, r *http.Request, status int) {
    e.renderer.Render(w, r, status, "error", PageModel{Data: errorRecord{Status: status, Title: e.title(status), Text: e.text(status)}})
}

func (e *ErrorPages) title(status int) string {
    switch status {
        case http.StatusBadRequest:
            return "Requête invalide"
        case http.StatusMethodNotAllowed:
            return "Méthode non autorisée"
        case http.StatusRequestURITooLong:
            return "Adresse trop longue"
        case http.StatusMisdirectedRequest:
            return "Hôte inconnu"
        case http.StatusTooManyRequests:
            return "Trop de tentatives"
        case http.StatusGone:
            return "Lien expiré"
    }
    return "Erreur interne"
}

func (e *ErrorPages) text(status int) string {
    switch status {
        case http.StatusBadRequest:
            return "La requête ne peut pas être traitée."
        case http.StatusMethodNotAllowed:
            return "Cette adresse n'accepte pas cette méthode."
        case http.StatusRequestURITooLong:
            return "L'adresse demandée dépasse la taille autorisée."
        case http.StatusMisdirectedRequest:
            return "Ce serveur ne répond pas pour ce nom d'hôte."
        case http.StatusTooManyRequests:
            return "Patientez quelques instants avant de réessayer."
        case http.StatusGone:
            return "Ce fichier n'est plus disponible : demandez un nouvel export à l'assistant."
    }
    return "Une erreur inattendue s'est produite. Réessayez plus tard."
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
