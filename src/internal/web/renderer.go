/*
 * renderer.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "bytes"
    "fmt"
    "html/template"
    "io/fs"
    "log/slog"
    "net/http"
    "path"
    "strconv"
    "strings"
    "unicode"
)

// ---------------------------------------------------------------------------
// Renderer declaration
// ---------------------------------------------------------------------------

type Renderer struct {
    views    map[string]*template.Template
    families map[string]string
    clock    yearSource
    sessions *Sessions
    notices  *Notices
    logger   *slog.Logger
}

type yearSource interface {
    Year() int
}

const layoutFile = "layout.html"

// ---------------------------------------------------------------------------
// Renderer implementation
// ---------------------------------------------------------------------------

func NewRenderer(templates fs.FS, clock yearSource, sessions *Sessions, logger *slog.Logger) (*Renderer, error) {
    r := &Renderer{views: map[string]*template.Template{}, families: map[string]string{}, clock: clock, sessions: sessions, notices: NewNotices(), logger: logger}
    files, err := fs.Glob(templates, "*.html")
    if err != nil {
        return nil, err
    }
    for _, file := range files {
        if file == layoutFile {
            continue
        }
        view := strings.TrimSuffix(path.Base(file), ".html")
        for _, family := range r.familiesOf(view) {
            set, err := template.ParseFS(templates, layoutFile, "partials/" + family + ".html", file)
            if err != nil {
                return nil, fmt.Errorf("template %s: %w", file, err)
            }
            r.views[view + "@" + family] = set
        }
        r.families[view] = r.familiesOf(view)[0]
    }
    return r, nil
}

func (r *Renderer) familiesOf(view string) []string {
    switch {
        case (view == "error") || (view == "forbidden") || (view == "not_found"):
            return []string{FamilyPublic, FamilyDashboard, FamilyAdmin}
        case strings.HasPrefix(view, "admin_"):
            return []string{FamilyAdmin}
        case (view == "home") || (view == "login") || (view == "logout") || (view == "logged_out"):
            return []string{FamilyPublic}
    }
    return []string{FamilyDashboard, FamilyAdmin}
}

func (r *Renderer) Redirect(w http.ResponseWriter, req *http.Request, path string, notice string) {
    if viewer := (&Viewer{}).From(req); viewer != nil {
        r.sessions.SetNotice(viewer.SessionKey, notice)
    }
    http.Redirect(w, req, path, http.StatusSeeOther)
}

func (r *Renderer) Render(w http.ResponseWriter, req *http.Request, status int, view string, model PageModel) {
    viewer := (&Viewer{}).From(req)
    family := r.families[view]
    if (family != FamilyPublic) || (len(r.familiesOf(view)) > 1) {
        family = r.family(req, viewer, view)
    }
    tmpl, ok := r.views[view + "@" + family]
    if !ok {
        r.fail(w, fmt.Errorf("unknown view %q", view))
        return
    }
    model.Chrome = Chrome{Family: family, Section: model.Section, Path: req.URL.Path, Year: r.clock.Year()}
    if (viewer != nil) && (model.Notice == "") && (model.Warning == "") {
        if code := r.sessions.TakeNotice(viewer.SessionKey); r.notices.IsWarning(code) {
            model.Warning = r.notices.Message(code)
        } else {
            model.Notice = r.notices.Message(code)
        }
    }
    if viewer != nil {
        model.Chrome.Connected = true
        model.Chrome.Admin = viewer.IsAdmin()
        model.Chrome.Login = string(viewer.Login)
        model.Chrome.Name = viewer.Name
        if viewer.Name == "" {
            model.Chrome.Name = string(viewer.Login)
        }
        model.Chrome.Initials = r.initials(model.Chrome.Name)
        model.Chrome.ProfileLabel = viewer.Profile.Label()
        model.Chrome.CSRF = viewer.CSRF
        model.Chrome.Assistant = viewer.Assistant
        if model.CSRF == "" {
            model.CSRF = viewer.CSRF
        }
    }
    var buffer bytes.Buffer
    if err := tmpl.ExecuteTemplate(&buffer, "layout", model); err != nil {
        r.fail(w, err)
        return
    }
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    w.Header().Set("Content-Length", strconv.Itoa(buffer.Len()))
    w.WriteHeader(status)
    w.Write(buffer.Bytes())
}

func (r *Renderer) family(req *http.Request, viewer *Viewer, view string) string {
    families := r.familiesOf(view)
    admin := strings.HasPrefix(path.Clean(req.URL.Path) + "/", "/admin/")
    for _, family := range families {
        switch {
            case admin && viewer.IsAdmin() && (family == FamilyAdmin):
                return family
            case (!admin || !viewer.IsAdmin()) && (viewer != nil) && (family == FamilyDashboard):
                return family
            case (viewer == nil) && (family == FamilyPublic):
                return family
        }
    }
    return families[0]
}

func (r *Renderer) initials(label string) string {
    words := [][]rune{}
    for _, word := range strings.Fields(label) {
        if letters := r.alphanumerics(word); len(letters) > 0 {
            words = append(words, letters)
        }
    }
    switch {
        case len(words) == 0:
            return ""
        case len(words) == 1:
            return string(words[0][:min(2, len(words[0]))])
    }
    return string([]rune{words[0][0], words[len(words) - 1][0]})
}

func (r *Renderer) alphanumerics(word string) []rune {
    letters := []rune{}
    for _, letter := range word {
        if unicode.IsLetter(letter) || unicode.IsDigit(letter) {
            letters = append(letters, unicode.ToUpper(letter))
        }
    }
    return letters
}

func (r *Renderer) fail(w http.ResponseWriter, err error) {
    r.logger.Error("template rendering failed", "error", err.Error())
    w.Header().Set("Content-Type", "text/plain; charset=utf-8")
    w.WriteHeader(http.StatusInternalServerError)
    w.Write([]byte("Erreur interne.\n"))
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
