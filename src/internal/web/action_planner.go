/*
 * action_planner.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "encoding/json"
    "errors"
    "io"
    "slices"
    "strings"

    "timesheet/internal/activity"
)

// ---------------------------------------------------------------------------
// ActionPlanner declaration
// ---------------------------------------------------------------------------

type ActionPlanner struct {
    clock  activity.Clock
    random io.Reader
    editor *MonthEditor
    months *MonthView
}

type plannedDay struct {
    Date    string `json:"date"`
    AM      string `json:"am"`
    PM      string `json:"pm"`
    Comment string `json:"comment"`
}

type plannedAction struct {
    Month   string       `json:"month"`
    Days    []plannedDay `json:"days"`
    Reason  string       `json:"reason"`
    Date    string       `json:"date"`
    Counter string       `json:"counter"`
    Amount  string       `json:"amount"`
    ID      string       `json:"id"`
}

const (
    ToolDays              = "propose_days"
    ToolValidation        = "propose_validation"
    ToolReopening         = "propose_reopening"
    ToolAdjustment        = "propose_adjustment"
    ToolAdjustmentRemoval = "propose_adjustment_removal"

    invalidArguments = "Les arguments de l'action sont invalides."
    invalidMonth     = "Mois invalide : le format attendu est AAAA-MM."
    noDays           = "Aucune demi-journée à saisir."
    unknownAction    = "Action inconnue."
)

// ---------------------------------------------------------------------------
// ActionPlanner implementation
// ---------------------------------------------------------------------------

func NewActionPlanner(clock activity.Clock, random io.Reader) *ActionPlanner {
    return &ActionPlanner{clock: clock, random: random, editor: NewMonthEditor(random), months: NewMonthView()}
}

func (p *ActionPlanner) Writes(tool string) bool {
    return (tool == ToolDays) || (tool == ToolValidation) || (tool == ToolReopening) || (tool == ToolAdjustment) || (tool == ToolAdjustmentRemoval)
}

func (p *ActionPlanner) Plan(ws *Workspace, tool string, arguments string) (ActionPlan, string) {
    action := plannedAction{}
    decoder := json.NewDecoder(bytes.NewReader([]byte(arguments)))
    decoder.DisallowUnknownFields()
    if err := decoder.Decode(&action); err != nil {
        return ActionPlan{}, invalidArguments
    }
    switch tool {
        case ToolDays:
            return p.days(ws, action)
        case ToolValidation:
            return p.validation(ws, action)
        case ToolReopening:
            return p.reopening(ws, action)
        case ToolAdjustment:
            return p.adjustment(ws, action)
        case ToolAdjustmentRemoval:
            return p.removal(ws, action)
    }
    return ActionPlan{}, unknownAction
}

func (p *ActionPlanner) month(text string) (activity.Month, bool) {
    m, err := activity.NewMonthFromText(strings.TrimSpace(text))
    return m, (err == nil) && m.Valid()
}

func (p *ActionPlanner) days(ws *Workspace, action plannedAction) (ActionPlan, string) {
    m, valid := p.month(action.Month)
    if !valid {
        return ActionPlan{}, invalidMonth
    }
    if !p.editor.Editable(ws, m) {
        return ActionPlan{}, lockedMonth
    }
    if (len(action.Days) == 0) || (len(action.Days) > m.Days()) {
        return ActionPlan{}, noDays
    }
    calendar := ws.Calendar.Clone()
    entries := map[activity.Date]activity.DayEntry{}
    plan := ActionPlan{Month: m, Notice: "saved", Event: "month saved", Done: "Les demi-journées de " + m.Label() + " ont été enregistrées."}
    for _, day := range action.Days {
        d, err := activity.NewDateFromText(strings.TrimSpace(day.Date))
        if (err != nil) || !m.Contains(d) || !calendar.Tracks(d) {
            return ActionPlan{}, "La date " + day.Date + " n'appartient pas au mois " + m.Label() + " ou précède le début du suivi."
        }
        if _, twice := entries[d]; twice {
            return ActionPlan{}, "La date " + day.Date + " apparaît deux fois."
        }
        entry := activity.DayEntry{AM: strings.TrimSpace(day.AM), PM: strings.TrimSpace(day.PM), Comment: strings.TrimSpace(day.Comment)}
        entries[d] = entry
        plan.Summary = append(plan.Summary, p.describe(ws, d, entry))
    }
    if problems := p.editor.Apply(ws, &calendar, entries); len(problems) > 0 {
        dates := []activity.Date{}
        for d := range problems {
            dates = append(dates, d)
        }
        slices.SortFunc(dates, p.compare)
        lines := []string{}
        for _, d := range dates {
            lines = append(lines, d.Label() + " : " + strings.Join(problems[d], " "))
        }
        return ActionPlan{}, invalidEntries + " " + strings.Join(lines, " ; ")
    }
    plan.Calendar = calendar
    return plan, ""
}

func (p *ActionPlanner) compare(a activity.Date, b activity.Date) int {
    return a.Compare(b)
}

func (p *ActionPlanner) describe(ws *Workspace, d activity.Date, entry activity.DayEntry) string {
    if entry.Empty() {
        return d.Label() + " : saisie effacée"
    }
    text := d.Label() + " : matin " + p.code(ws, entry.AM) + ", après-midi " + p.code(ws, entry.PM)
    if entry.Comment != "" {
        text += ", commentaire « " + entry.Comment + " »"
    }
    return text
}

func (p *ActionPlanner) code(ws *Workspace, id string) string {
    if code, known := ws.Config.Activity(id); known {
        return code.ID + " (" + code.Label + ")"
    }
    if id == "" {
        return "non renseigné"
    }
    return id
}

func (p *ActionPlanner) validation(ws *Workspace, action plannedAction) (ActionPlan, string) {
    m, valid := p.month(action.Month)
    if !valid {
        return ActionPlan{}, invalidMonth
    }
    closing := activity.NewMonthClosing(ws.Book, activity.NewIdentifierGenerator(p.random))
    calendar, err := closing.Validate(m, p.clock.Now().UTC(), ws.Today)
    if err != nil {
        return ActionPlan{}, "Validation impossible : " + p.months.reason(err)
    }
    summary := []string{"Valider le mois de " + m.Label() + " : ses écritures seront figées dans les compteurs et ne pourront plus être modifiées, sauf en rouvrant le mois."}
    return ActionPlan{Calendar: calendar, Month: m, Summary: summary, Notice: "validated", Event: "month validated", Done: "Le mois de " + m.Label() + " a été validé."}, ""
}

func (p *ActionPlanner) reopening(ws *Workspace, action plannedAction) (ActionPlan, string) {
    m, valid := p.month(action.Month)
    if !valid {
        return ActionPlan{}, invalidMonth
    }
    reason := strings.TrimSpace(action.Reason)
    closing := activity.NewMonthClosing(ws.Book, activity.NewIdentifierGenerator(p.random))
    calendar, err := closing.Reopen(m, reason, p.clock.Now().UTC())
    if errors.Is(err, activity.ErrReasonRequired) {
        return ActionPlan{}, "Indiquez le motif de la réouverture (une ligne, 200 caractères au plus)."
    }
    if err != nil {
        return ActionPlan{}, "Seul le dernier mois validé peut être rouvert."
    }
    summary := []string{"Rouvrir le mois de " + m.Label() + " : ses écritures seront contre-passées.", "Motif : « " + reason + " »"}
    return ActionPlan{Calendar: calendar, Month: m, Summary: summary, Notice: "reopened", Event: "month reopened", Done: "Le mois de " + m.Label() + " a été rouvert."}, ""
}

func (p *ActionPlanner) adjustment(ws *Workspace, action plannedAction) (ActionPlan, string) {
    calendar := ws.Calendar.Clone()
    m, refusal := p.editor.AddAdjustment(ws, &calendar, strings.TrimSpace(action.Date), strings.TrimSpace(action.Counter), action.Amount, action.Reason)
    if refusal != "" {
        return ActionPlan{}, refusal
    }
    added := calendar.Adjustments[len(calendar.Adjustments) - 1]
    rule, _ := ws.Config.Counter(added.Counter)
    summary := []string{"Ajustement de " + added.Amount.FormatSigned() + " jour sur le compteur " + rule.ShortLabel + " (" + rule.Label + ") au " + added.Date.Numeric(), "Motif : « " + added.Reason + " »"}
    return ActionPlan{Calendar: calendar, Month: m, Summary: summary, Notice: "adjusted", Event: "adjustments changed", Done: "L'ajustement a été enregistré."}, ""
}

func (p *ActionPlanner) removal(ws *Workspace, action plannedAction) (ActionPlan, string) {
    calendar := ws.Calendar.Clone()
    id := strings.TrimSpace(action.ID)
    var removed activity.Adjustment
    for _, adjustment := range calendar.Adjustments {
        if adjustment.ID == id {
            removed = adjustment
        }
    }
    m, refusal := p.editor.DeleteAdjustment(&calendar, id)
    if refusal != "" {
        return ActionPlan{}, refusal
    }
    summary := []string{"Supprimer l'ajustement de " + removed.Amount.FormatSigned() + " jour sur le compteur " + removed.Counter + " au " + removed.Date.Numeric() + ", motif « " + removed.Reason + " »"}
    return ActionPlan{Calendar: calendar, Month: m, Summary: summary, Notice: "unadjusted", Event: "adjustments changed", Done: "L'ajustement a été supprimé."}, ""
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
