/*
 * dashboard_controller.go - Copyright (c) 2023-2026 - Olivier Poncet
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

    "timesheet/internal/activity"
)

// ---------------------------------------------------------------------------
// DashboardController declaration
// ---------------------------------------------------------------------------

type DashboardController struct {
    loader     *WorkspaceLoader
    repository Repository
    renderer   *Renderer
    errors     *ErrorPages
    clock      activity.Clock
    random     io.Reader
    logger     *slog.Logger
    dashboards *DashboardView
    months     *MonthView
    years      *YearView
    editor     *MonthEditor
}

const (
    MaxDays        = 366
    PlanningMonths = 24
)

// ---------------------------------------------------------------------------
// DashboardController implementation
// ---------------------------------------------------------------------------

func NewDashboardController(loader *WorkspaceLoader, repository Repository, renderer *Renderer, errors *ErrorPages, clock activity.Clock, random io.Reader, logger *slog.Logger) *DashboardController {
    return &DashboardController{loader: loader, repository: repository, renderer: renderer, errors: errors, clock: clock, random: random, logger: logger, dashboards: NewDashboardView(), months: NewMonthView(), years: NewYearView(), editor: NewMonthEditor(random)}
}

func (c *DashboardController) Show(w http.ResponseWriter, r *http.Request) {
    ws, ok := c.loader.Own(r)
    if !ok {
        c.errors.NotFound(w, r)
        return
    }
    c.renderer.Render(w, r, http.StatusOK, "dashboard", PageModel{Section: "dashboard", Data: c.dashboards.Build(ws)})
}

func (c *DashboardController) ShowBalancesOther(w http.ResponseWriter, r *http.Request) {
    ws, ok := c.loader.Other(r)
    if !ok {
        c.errors.NotFound(w, r)
        return
    }
    c.renderer.Render(w, r, http.StatusOK, "dashboard", PageModel{Section: "accounts", Data: c.dashboards.Build(ws)})
}

func (c *DashboardController) ShowActivity(w http.ResponseWriter, r *http.Request) {
    ws, m, ok := c.workspace(w, r, false)
    if ok {
        c.render(w, r, ws, PageActivity, http.StatusOK, m, nil, nil, "")
    }
}

func (c *DashboardController) ShowMonth(w http.ResponseWriter, r *http.Request) {
    ws, m, ok := c.workspace(w, r, false)
    if ok {
        c.render(w, r, ws, PageMonth, http.StatusOK, m, nil, nil, "")
    }
}

func (c *DashboardController) ShowOther(w http.ResponseWriter, r *http.Request) {
    ws, m, ok := c.workspace(w, r, true)
    if ok {
        c.render(w, r, ws, PageMonth, http.StatusOK, m, nil, nil, "")
    }
}

func (c *DashboardController) workspace(w http.ResponseWriter, r *http.Request, other bool) (*Workspace, activity.Month, bool) {
    ws, found := c.loader.Own(r)
    if other {
        ws, found = c.loader.Other(r)
    }
    m, valid := c.loader.Month(r)
    if !found || !valid {
        c.errors.NotFound(w, r)
        return nil, activity.Month{}, false
    }
    return ws, m, true
}

func (c *DashboardController) render(w http.ResponseWriter, r *http.Request, ws *Workspace, page string, status int, m activity.Month, submitted map[activity.Date]activity.DayEntry, problems map[activity.Date][]string, alert string) {
    section := page
    if ws.ReadOnly {
        section = "accounts"
    }
    model := PageModel{Section: section, Alert: alert, Data: c.months.Build(ws, page, m, submitted, problems)}
    c.renderer.Render(w, r, status, page, model)
}

func (c *DashboardController) CurrentActivity(w http.ResponseWriter, r *http.Request) {
    c.current(w, r, PageActivity)
}

func (c *DashboardController) CurrentMonth(w http.ResponseWriter, r *http.Request) {
    c.current(w, r, PageMonth)
}

func (c *DashboardController) current(w http.ResponseWriter, r *http.Request, page string) {
    ws, found := c.loader.Own(r)
    if !found {
        c.errors.NotFound(w, r)
        return
    }
    http.Redirect(w, r, ws.PagePath(page, ws.Today.Month()), http.StatusSeeOther)
}

func (c *DashboardController) Save(w http.ResponseWriter, r *http.Request) {
    ws, m, ok := c.workspace(w, r, false)
    if !ok {
        return
    }
    if !c.editor.Editable(ws, m) {
        c.render(w, r, ws, PageActivity, http.StatusConflict, m, nil, nil, lockedMonth)
        return
    }
    if !c.upToDate(w, r, ws, m) {
        return
    }
    submitted := map[activity.Date]activity.DayEntry{}
    calendar := ws.Calendar.Clone()
    for d := m.First(); !d.After(m.Last()); d = d.AddDays(1) {
        if !calendar.Tracks(d) {
            continue
        }
        key := d.String()
        if _, posted := r.PostForm["comment:" + key]; !posted {
            c.errors.BadRequest(w, r)
            return
        }
        entry := activity.DayEntry{AM: r.PostFormValue("am:" + key), PM: r.PostFormValue("pm:" + key), Comment: strings.TrimSpace(r.PostFormValue("comment:" + key))}
        if whole := r.PostFormValue("day:" + key); whole != "" {
            entry.AM, entry.PM = whole, whole
        }
        submitted[d] = entry
    }
    if problems := c.editor.Apply(ws, &calendar, submitted); len(problems) > 0 {
        c.render(w, r, ws, PageActivity, http.StatusUnprocessableEntity, m, submitted, problems, invalidEntries)
        return
    }
    if !c.save(w, r, ws, PageActivity, calendar, m) {
        return
    }
    c.logger.Info("month saved", "login", string(ws.Owner.Login), "month", m.String())
    c.renderer.Redirect(w, r, ws.PagePath(PageActivity, m), "saved")
}

func (c *DashboardController) upToDate(w http.ResponseWriter, r *http.Request, ws *Workspace, m activity.Month) bool {
    revision, err := strconv.ParseInt(r.PostFormValue("revision"), 10, 64)
    if err != nil {
        c.errors.BadRequest(w, r)
        return false
    }
    if revision != ws.Calendar.Revision {
        c.render(w, r, ws, PageActivity, http.StatusConflict, m, nil, nil, "Les données ont changé entre-temps (autre onglet ?). Rechargez la page avant de recommencer.")
        return false
    }
    return true
}

func (c *DashboardController) save(w http.ResponseWriter, r *http.Request, ws *Workspace, page string, calendar activity.Calendar, m activity.Month) bool {
    revision, err := strconv.ParseInt(r.PostFormValue("revision"), 10, 64)
    if err != nil {
        c.errors.BadRequest(w, r)
        return false
    }
    err = activity.ErrStaleRevision
    if revision == ws.Calendar.Revision {
        err = c.repository.SaveCalendar(ws.Owner.Login, ws.Calendar.Revision, calendar)
    }
    if err != nil {
        if errors.Is(err, activity.ErrStaleRevision) {
            if fresh, found := c.loader.Own(r); found {
                c.render(w, r, fresh, page, http.StatusConflict, m, nil, nil, "Les données ont changé entre-temps (autre onglet ?). Rechargez la page avant de recommencer.")
                return false
            }
            http.Redirect(w, r, "/login", http.StatusSeeOther)
            return false
        }
        if errors.Is(err, activity.ErrCalendarTooLarge) {
            c.render(w, r, ws, page, http.StatusRequestEntityTooLarge, m, nil, nil, "Les données de ce compte atteignent la taille maximale autorisée.")
            return false
        }
        c.logger.Error("cannot save the calendar", "error", err.Error())
        c.errors.Status(w, r, http.StatusInternalServerError)
        return false
    }
    return true
}

func (c *DashboardController) Validate(w http.ResponseWriter, r *http.Request) {
    ws, m, ok := c.workspace(w, r, false)
    if !ok {
        return
    }
    if !c.upToDate(w, r, ws, m) {
        return
    }
    closing := activity.NewMonthClosing(ws.Book, activity.NewIdentifierGenerator(c.random))
    calendar, err := closing.Validate(m, c.clock.Now().UTC(), ws.Today)
    if err != nil {
        c.render(w, r, ws, PageActivity, http.StatusConflict, m, nil, nil, "Validation impossible : " + c.months.reason(err))
        return
    }
    if !c.save(w, r, ws, PageActivity, calendar, m) {
        return
    }
    c.logger.Info("month validated", "login", string(ws.Owner.Login), "month", m.String())
    c.renderer.Redirect(w, r, ws.PagePath(PageActivity, m), "validated")
}

func (c *DashboardController) Reopen(w http.ResponseWriter, r *http.Request) {
    ws, m, ok := c.workspace(w, r, false)
    if !ok {
        return
    }
    if !c.upToDate(w, r, ws, m) {
        return
    }
    closing := activity.NewMonthClosing(ws.Book, activity.NewIdentifierGenerator(c.random))
    calendar, err := closing.Reopen(m, r.PostFormValue("reason"), c.clock.Now().UTC())
    if err != nil {
        alert := "Seul le dernier mois validé peut être rouvert."
        if errors.Is(err, activity.ErrReasonRequired) {
            alert = "Indiquez le motif de la réouverture (une ligne, 200 caractères au plus)."
        }
        c.render(w, r, ws, PageActivity, http.StatusConflict, m, nil, nil, alert)
        return
    }
    if !c.save(w, r, ws, PageActivity, calendar, m) {
        return
    }
    c.logger.Info("month reopened", "login", string(ws.Owner.Login), "month", m.String())
    c.renderer.Redirect(w, r, ws.PagePath(PageActivity, m), "reopened")
}

func (c *DashboardController) Adjust(w http.ResponseWriter, r *http.Request) {
    ws, found := c.loader.Own(r)
    if !found {
        c.errors.NotFound(w, r)
        return
    }
    calendar := ws.Calendar.Clone()
    var m activity.Month
    var alert string
    notice := "adjusted"
    switch r.PostFormValue("action") {
        case "add":
            m, alert = c.editor.AddAdjustment(ws, &calendar, r.PostFormValue("date"), r.PostFormValue("counter"), r.PostFormValue("amount"), r.PostFormValue("reason"))
        case "delete":
            m, alert = c.editor.DeleteAdjustment(&calendar, r.PostFormValue("id"))
            notice = "unadjusted"
        default:
            c.errors.BadRequest(w, r)
            return
    }
    if m.IsZero() {
        m = ws.Today.Month()
    }
    if !c.upToDate(w, r, ws, m) {
        return
    }
    if alert != "" {
        c.render(w, r, ws, PageActivity, http.StatusUnprocessableEntity, m, nil, nil, alert)
        return
    }
    if !c.save(w, r, ws, PageActivity, calendar, m) {
        return
    }
    c.logger.Info("adjustments changed", "login", string(ws.Owner.Login), "month", m.String(), "change", notice)
    c.renderer.Redirect(w, r, ws.PagePath(PageActivity, m), notice)
}

func (c *DashboardController) Year(w http.ResponseWriter, r *http.Request) {
    c.year(w, r, false)
}

func (c *DashboardController) YearOther(w http.ResponseWriter, r *http.Request) {
    c.year(w, r, true)
}

func (c *DashboardController) year(w http.ResponseWriter, r *http.Request, other bool) {
    ws, found := c.loader.Own(r)
    if other {
        ws, found = c.loader.Other(r)
    }
    year, valid := c.loader.Year(r)
    if !found || !valid {
        c.errors.NotFound(w, r)
        return
    }
    section := "year"
    if other {
        section = "accounts"
    }
    c.renderer.Render(w, r, http.StatusOK, "year", PageModel{Section: section, Data: c.years.Build(ws, year)})
}

func (c *DashboardController) Colors(w http.ResponseWriter, r *http.Request) {
    ws, found := c.loader.Own(r)
    if !found {
        c.errors.NotFound(w, r)
        return
    }
    colors := ws.Calendar.Settings.Colors
    var css strings.Builder
    css.WriteString(":root {\n")
    for _, code := range ws.Config.Activities {
        color := colors.OfCode(code)
        css.WriteString("    --code-" + code.ID + ": " + string(color) + ";\n")
        css.WriteString("    --code-" + code.ID + "-ink: " + string(color.Ink()) + ";\n")
    }
    css.WriteString("}\n")
    for _, code := range ws.Config.Activities {
        css.WriteString(".code-" + code.ID + " {\n    background: var(--code-" + code.ID + ");\n    color: var(--code-" + code.ID + "-ink);\n}\n")
    }
    w.Header().Set("Content-Type", "text/css; charset=utf-8")
    io.WriteString(w, css.String())
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
