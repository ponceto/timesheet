/*
 * data_controller.go - Copyright (c) 2023-2026 - Olivier Poncet
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
// DataController declaration
// ---------------------------------------------------------------------------

type DataController struct {
    loader     *WorkspaceLoader
    repository Repository
    renderer   *Renderer
    errors     *ErrorPages
    months     *MonthView
    clock      activity.Clock
    random     io.Reader
    logger     *slog.Logger
}

type dataRecord struct {
    Revision   int64
    ImportPath string
    ExportPath string
    ResetPath  string
}

const (
    dataPath       = "/dashboard/data"
    dataExportPath = "/dashboard/data/export"
    dataImportPath = "/dashboard/data/import"
    dataResetPath  = "/dashboard/data/reset"
    actionCheck    = "check"
    actionImport   = "import"
)

// ---------------------------------------------------------------------------
// DataController implementation
// ---------------------------------------------------------------------------

func NewDataController(loader *WorkspaceLoader, repository Repository, renderer *Renderer, errors *ErrorPages, clock activity.Clock, random io.Reader, logger *slog.Logger) *DataController {
    return &DataController{loader: loader, repository: repository, renderer: renderer, errors: errors, months: NewMonthView(), clock: clock, random: random, logger: logger}
}

func (c *DataController) Show(w http.ResponseWriter, r *http.Request) {
    ws, found := c.loader.Own(r)
    if !found {
        c.errors.NotFound(w, r)
        return
    }
    c.page(w, r, ws, http.StatusOK, PageModel{})
}

func (c *DataController) page(w http.ResponseWriter, r *http.Request, ws *Workspace, status int, model PageModel) {
    model.Section, model.Data = "data", dataRecord{Revision: ws.Calendar.Revision, ImportPath: dataImportPath, ExportPath: dataExportPath, ResetPath: dataResetPath}
    c.renderer.Render(w, r, status, "data", model)
}

func (c *DataController) Export(w http.ResponseWriter, r *http.Request) {
    ws, found := c.loader.Own(r)
    if !found {
        c.errors.NotFound(w, r)
        return
    }
    data, err := activity.NewImportFileOf(ws.Calendar, ws.Config).Encode()
    if err != nil {
        c.logger.Error("cannot export the data", "error", err.Error())
        c.errors.Status(w, r, http.StatusInternalServerError)
        return
    }
    name := "timesheet-" + string(ws.Owner.Login) + "-" + ws.Today.String() + ".json"
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.Header().Set("Content-Disposition", `attachment; filename="` + name + `"`)
    w.Header().Set("Cache-Control", "no-store")
    w.Header().Set("Content-Length", strconv.Itoa(len(data)))
    w.WriteHeader(http.StatusOK)
    w.Write(data)
    c.logger.Info("data exported", "login", string(ws.Owner.Login))
}

func (c *DataController) Import(w http.ResponseWriter, r *http.Request) {
    ws, found := c.loader.Own(r)
    if !found {
        c.errors.NotFound(w, r)
        return
    }
    action := r.PostFormValue("action")
    revision, err := strconv.ParseInt(r.PostFormValue("revision"), 10, 64)
    if ((action != actionCheck) && (action != actionImport)) || (err != nil) {
        c.errors.BadRequest(w, r)
        return
    }
    if revision != ws.Calendar.Revision {
        c.store(w, r, ws, activity.ErrStaleRevision)
        return
    }
    file, _, err := r.FormFile("file")
    if err != nil {
        c.page(w, r, ws, http.StatusUnprocessableEntity, PageModel{Alert: "Choisissez un fichier à importer."})
        return
    }
    defer file.Close()
    data, err := io.ReadAll(io.LimitReader(file, activity.MaxImportSize + 1))
    if err != nil {
        c.errors.BadRequest(w, r)
        return
    }
    imported, err := activity.NewImportFile(data)
    if err != nil {
        c.page(w, r, ws, http.StatusUnprocessableEntity, PageModel{Alert: "Le fichier n'est pas un import valide : un objet JSON au format timesheet-import, version 1, sans champ inconnu."})
        return
    }
    options := activity.NewImportOptions()
    options.Replace, options.Validate = r.PostFormValue("replace") == "on", r.PostFormValue("validate") == "on"
    importer := activity.NewImporter(ws.Config, c.clock, activity.NewIdentifierGenerator(c.random), ws.Location)
    report, err := importer.Run(ws.Calendar, imported, options, ws.Today, ws.Horizon())
    if err != nil {
        c.logger.Error("cannot import the data", "error", err.Error())
        c.errors.Status(w, r, http.StatusInternalServerError)
        return
    }
    model := c.report(report)
    if len(report.Problems) > 0 {
        model.Alert, model.Details = "L'import est refusé : rien n'a été enregistré. Corrigez le fichier puis recommencez.", nil
        c.page(w, r, ws, http.StatusUnprocessableEntity, model)
        return
    }
    if action == actionCheck {
        if !c.store(w, r, ws, c.repository.CheckCalendar(ws.Owner.Login, revision, report.Calendar)) {
            return
        }
        model.Notice = "Vérification réussie : le fichier peut être importé. Rien n'a été enregistré."
        c.page(w, r, ws, http.StatusOK, model)
        return
    }
    if !c.store(w, r, ws, c.repository.SaveCalendar(ws.Owner.Login, revision, report.Calendar)) {
        return
    }
    c.logger.Info("import saved", "login", string(ws.Owner.Login), "days", len(report.Dates), "adjustments", report.Adjustments, "validated", len(report.Validated))
    c.renderer.Redirect(w, r, dataPath, "imported")
}

func (c *DataController) Reset(w http.ResponseWriter, r *http.Request) {
    ws, found := c.loader.Own(r)
    if !found {
        c.errors.NotFound(w, r)
        return
    }
    revision, err := strconv.ParseInt(r.PostFormValue("revision"), 10, 64)
    if err != nil {
        c.errors.BadRequest(w, r)
        return
    }
    if revision != ws.Calendar.Revision {
        c.store(w, r, ws, activity.ErrStaleRevision)
        return
    }
    if r.PostFormValue("confirm") != string(ws.Owner.Login) {
        c.page(w, r, ws, http.StatusUnprocessableEntity, PageModel{Alert: "Retapez exactement votre identifiant pour confirmer la suppression de vos données."})
        return
    }
    if !c.store(w, r, ws, c.repository.EraseCalendar(ws.Owner.Login, revision, ws.Today.Month().First())) {
        return
    }
    c.logger.Info("data erased", "login", string(ws.Owner.Login))
    c.renderer.Redirect(w, r, dataPath, "erased")
}

func (c *DataController) store(w http.ResponseWriter, r *http.Request, ws *Workspace, err error) bool {
    switch {
        case err == nil:
            return true
        case errors.Is(err, activity.ErrStaleRevision):
            c.page(w, r, ws, http.StatusConflict, PageModel{Alert: "Les données ont changé entre-temps (autre onglet ?). Rechargez la page avant de recommencer."})
        case errors.Is(err, activity.ErrCalendarTooLarge):
            c.page(w, r, ws, http.StatusRequestEntityTooLarge, PageModel{Alert: "Les données de ce compte atteignent la taille maximale autorisée."})
        default:
            c.logger.Error("cannot save the calendar", "error", err.Error())
            c.errors.Status(w, r, http.StatusInternalServerError)
    }
    return false
}

func (c *DataController) report(report activity.ImportReport) PageModel {
    model := PageModel{Details: []string{}, Warnings: []string{}, Alerts: []string{}}
    for _, problem := range report.Problems {
        model.Alerts = append(model.Alerts, problem.Label())
    }
    for _, warning := range report.Warnings {
        model.Warnings = append(model.Warnings, warning.Label())
    }
    if report.Settings {
        model.Details = append(model.Details, "Réglages lus : date de début au " + report.Calendar.Settings.Start.Numeric() + ", " + strconv.Itoa(report.Counters) + " compteur(s).")
    }
    if len(report.Dates) > 0 {
        model.Details = append(model.Details, strconv.Itoa(len(report.Dates)) + " jour(s) lu(s), de " + report.First().Label() + " à " + report.Last().Label() + ".")
    }
    if report.Adjustments > 0 {
        model.Details = append(model.Details, strconv.Itoa(report.Adjustments) + " ajustement(s) lu(s).")
    }
    if len(report.Validated) > 0 {
        model.Details = append(model.Details, "Mois validés : de " + report.Validated[0].Label() + " à " + report.Validated[len(report.Validated) - 1].Label() + " (" + strconv.Itoa(len(report.Validated)) + ").")
    }
    for _, missing := range report.Missing {
        dates := []string{}
        for _, d := range missing.Dates {
            dates = append(dates, d.Numeric())
        }
        model.Warnings = append(model.Warnings, "Demi-journées non renseignées en " + missing.Month.Label() + " : " + strings.Join(dates, ", ") + ".")
    }
    if report.StopReason != nil {
        reason := "Ce mois n'est pas dans le fichier."
        if !errors.Is(report.StopReason, activity.ErrMonthNotImported) {
            reason = c.months.reason(report.StopReason)
        }
        model.Warnings = append(model.Warnings, "Validation arrêtée en " + report.Stopped.Label() + ". " + reason)
    }
    if len(model.Warnings) > 0 {
        model.Warning = "Points à vérifier :"
    }
    return model
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
