/*
 * tool_box.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "strconv"
    "strings"
    "time"

    "timesheet/internal/activity"
    "timesheet/internal/assistant"
)

// ---------------------------------------------------------------------------
// ToolBox declaration
// ---------------------------------------------------------------------------

type ToolBox struct {
    ws         *Workspace
    planner    *ActionPlanner
    positions  *PositionView
    statements *StatementView
    now        time.Time
    pending    *assistant.PendingAction
    export     *ToolExport
}

type toolArguments struct {
    Month   string `json:"month"`
    Year    int    `json:"year"`
    Counter string `json:"counter"`
}

type toolDay struct {
    Date    string `json:"date"`
    Weekday string `json:"weekday"`
    Weekend bool   `json:"weekend"`
    AM      string `json:"am"`
    PM      string `json:"pm"`
    Comment string `json:"comment,omitempty"`
}

type toolAdjustment struct {
    ID      string `json:"id"`
    Date    string `json:"date"`
    Counter string `json:"counter"`
    Amount  string `json:"amount"`
    Reason  string `json:"reason"`
}

type toolCounterMonth struct {
    Debit      string `json:"debit,omitempty"`
    Credit     string `json:"credit"`
    Adjustment string `json:"adjustment"`
    Balance    string `json:"balance"`
}

type toolSummaryRow struct {
    Month       string                      `json:"month"`
    State       string                      `json:"state"`
    Counters    map[string]toolCounterMonth `json:"counters,omitempty"`
    WorkingDays string                      `json:"working_days,omitempty"`
    Holidays    string                      `json:"holidays,omitempty"`
    Missing     string                      `json:"missing,omitempty"`
}

const (
    noParameters    = `{"type":"object","properties":{}}`
    monthParameters = `{"type":"object","properties":{"month":{"type":"string","description":"Mois au format AAAA-MM, par exemple 2026-11."}},"required":["month"]}`
    maxToolResult   = 64000
)

// ---------------------------------------------------------------------------
// ToolBox implementation
// ---------------------------------------------------------------------------

func NewToolBox(ws *Workspace, planner *ActionPlanner, now time.Time) *ToolBox {
    return &ToolBox{ws: ws, planner: planner, positions: NewPositionView(), statements: NewStatementView(), now: now}
}

func (b *ToolBox) WithExport(export *ToolExport) *ToolBox {
    b.export = export
    return b
}

func (b *ToolBox) Pending() *assistant.PendingAction {
    return b.pending
}

func (b *ToolBox) Tools() []assistant.Tool {
    tools := b.tools()
    if b.export != nil {
        tools = append(tools, NewAssistantTool(ToolExportData, "Prépare le fichier d'export des données de l'utilisateur (réglages, jours avec leurs commentaires, ajustements, au format d'import de « Mes données ») et donne un lien de téléchargement temporaire, réservé à l'utilisateur.", noParameters, b.exportData))
    }
    return tools
}

func (b *ToolBox) exportData(raw json.RawMessage) string {
    return b.encode(b.export.Run(b.ws, b.now))
}

func (b *ToolBox) tools() []assistant.Tool {
    return []assistant.Tool{
        NewAssistantTool("read_balances", "Soldes de tous les compteurs de l'utilisateur : aujourd'hui, au dernier mois validé et en prévision.", noParameters, b.readBalances),
        NewAssistantTool("read_month", "Demi-journées, commentaires, ajustements, anomalies, indicateurs et état (ouvert ou validé) d'un mois.", monthParameters, b.readMonth),
        NewAssistantTool("read_statement", "Relevé d'un compteur pour un mois : solde d'ouverture, écritures ligne à ligne et solde de clôture.", `{"type":"object","properties":{"counter":{"type":"string","description":"Identifiant du compteur, par exemple CP ou RTT."},"month":{"type":"string","description":"Mois au format AAAA-MM."}},"required":["counter","month"]}`, b.readStatement),
        NewAssistantTool("read_year", "Synthèse annuelle mois par mois : débits, crédits, ajustements et soldes de chaque compteur, jours ouvrés, fériés et non renseignés.", `{"type":"object","properties":{"year":{"type":"integer","description":"Année, par exemple 2026."}},"required":["year"]}`, b.readYear),
        NewAssistantTool("read_configuration", "Codes d'activité, compteurs et zone des jours fériés de l'installation.", noParameters, b.readConfiguration),
        NewAssistantTool("read_settings", "Réglages de l'utilisateur (début du suivi, compteurs, zone des jours fériés) et liste de tous ses ajustements avec leur identifiant.", noParameters, b.readSettings),
        NewAssistantTool(ToolDays, "Propose de saisir, changer ou effacer des demi-journées d'un mois ouvert. N'écrit rien : l'utilisateur confirme ou annule dans la page. Un code vide efface la demi-journée.", `{"type":"object","properties":{"month":{"type":"string","description":"Mois au format AAAA-MM."},"days":{"type":"array","description":"Les jours à écrire.","items":{"type":"object","properties":{"date":{"type":"string","description":"Date au format AAAA-MM-JJ."},"am":{"type":"string","description":"Code du matin."},"pm":{"type":"string","description":"Code de l'après-midi."},"comment":{"type":"string","description":"Commentaire du jour, facultatif."}},"required":["date","am","pm"]}}},"required":["month","days"]}`, b.propose(ToolDays)),
        NewAssistantTool(ToolValidation, "Propose de valider un mois terminé et complet. N'écrit rien : l'utilisateur confirme ou annule dans la page.", monthParameters, b.propose(ToolValidation)),
        NewAssistantTool(ToolReopening, "Propose de rouvrir le dernier mois validé, avec un motif. N'écrit rien : l'utilisateur confirme ou annule dans la page.", `{"type":"object","properties":{"month":{"type":"string","description":"Mois au format AAAA-MM."},"reason":{"type":"string","description":"Motif de la réouverture, une ligne."}},"required":["month","reason"]}`, b.propose(ToolReopening)),
        NewAssistantTool(ToolAdjustment, "Propose un ajustement signé d'un compteur dans un mois ouvert. N'écrit rien : l'utilisateur confirme ou annule dans la page.", `{"type":"object","properties":{"date":{"type":"string","description":"Date au format AAAA-MM-JJ."},"counter":{"type":"string","description":"Identifiant du compteur."},"amount":{"type":"string","description":"Quantité signée en jours, par exemple -1,5 ou 7/12."},"reason":{"type":"string","description":"Motif, une ligne."}},"required":["date","counter","amount","reason"]}`, b.propose(ToolAdjustment)),
        NewAssistantTool(ToolAdjustmentRemoval, "Propose de supprimer un ajustement d'un mois ouvert, désigné par son identifiant donné par read_settings. N'écrit rien : l'utilisateur confirme ou annule dans la page.", `{"type":"object","properties":{"id":{"type":"string","description":"Identifiant de l'ajustement."}},"required":["id"]}`, b.propose(ToolAdjustmentRemoval)),
    }
}

func (b *ToolBox) arguments(raw json.RawMessage) toolArguments {
    arguments := toolArguments{}
    json.Unmarshal(raw, &arguments)
    return arguments
}

func (b *ToolBox) encode(value any) string {
    var data bytes.Buffer
    encoder := json.NewEncoder(&data)
    encoder.SetEscapeHTML(false)
    if encoder.Encode(value) != nil {
        return `{"error":"résultat illisible"}`
    }
    if data.Len() > maxToolResult {
        return `{"error":"résultat trop volumineux, demandez une période plus courte"}`
    }
    return strings.TrimSpace(data.String())
}

func (b *ToolBox) refuse(message string) string {
    return b.encode(map[string]string{"error": message})
}

func (b *ToolBox) month(text string) (activity.Month, bool) {
    m, err := activity.NewMonthFromText(strings.TrimSpace(text))
    return m, (err == nil) && m.Valid()
}

func (b *ToolBox) readBalances(raw json.RawMessage) string {
    book := b.ws.Book
    return b.encode(map[string]any{"today": b.ws.Today.String(), "positions": b.positions.Build(book, book.Journal(book.Horizon(b.ws.Today)), b.ws.Today)})
}

func (b *ToolBox) readMonth(raw json.RawMessage) string {
    m, valid := b.month(b.arguments(raw).Month)
    if !valid {
        return b.refuse(invalidMonth)
    }
    state := "ouvert"
    if b.ws.Calendar.IsValidated(m) {
        state = "validé"
    }
    days := []toolDay{}
    for d := m.First(); !d.After(m.Last()); d = d.AddDays(1) {
        if b.ws.Calendar.Tracks(d) {
            entry := b.ws.Calendar.Days[d]
            days = append(days, toolDay{Date: d.String(), Weekday: d.WeekdayName(), Weekend: d.IsWeekend(), AM: entry.AM, PM: entry.PM, Comment: entry.Comment})
        }
    }
    anomalies := []string{}
    for _, anomaly := range b.ws.Book.Anomalies(m) {
        anomalies = append(anomalies, anomaly.Date.Label() + " : " + anomaly.Message)
    }
    adjustments := []toolAdjustment{}
    for _, adjustment := range b.adjustments() {
        if strings.HasPrefix(adjustment.Date, m.String()) {
            adjustments = append(adjustments, adjustment)
        }
    }
    return b.encode(map[string]any{"month": m.String(), "label": m.Label(), "state": state, "editable": b.planner.editor.Editable(b.ws, m), "tracking_start": b.ws.Calendar.Settings.Start.String(), "days": days, "adjustments": adjustments, "anomalies": anomalies, "indicators": b.ws.Book.Indicators(m, b.ws.Today)})
}

func (b *ToolBox) readStatement(raw json.RawMessage) string {
    arguments := b.arguments(raw)
    m, valid := b.month(arguments.Month)
    rule, known := b.ws.Config.Counter(strings.TrimSpace(arguments.Counter))
    if !valid || !known {
        return b.refuse("Compteur inconnu ou mois invalide (format AAAA-MM).")
    }
    book := b.ws.Book
    return b.encode(b.statements.Build(book.Journal(book.Horizon(b.ws.Today, m)), rule, m))
}

func (b *ToolBox) readYear(raw json.RawMessage) string {
    year := b.arguments(raw).Year
    if (year < activity.MinYear) || (year > activity.MaxYear) {
        return b.refuse("Année invalide.")
    }
    first, last := activity.NewMonth(year, time.January), activity.NewMonth(year, time.December)
    book := b.ws.Book
    summary := activity.NewSummary(book, book.Journal(book.Horizon(b.ws.Today, last)), first, last, b.ws.Today)
    rows := []toolSummaryRow{}
    for _, row := range append(summary.Rows, summary.Totals) {
        item := toolSummaryRow{Month: row.Month.String(), State: row.State.Label()}
        if row.Month.IsZero() {
            item.Month, item.State = "total " + strconv.Itoa(year), ""
        }
        if row.State != activity.RowUntracked {
            item.Counters = map[string]toolCounterMonth{}
            for _, rule := range summary.Counters {
                counter := toolCounterMonth{Credit: row.Credits[rule.ID].Format(), Adjustment: row.Adjustments[rule.ID].FormatSigned(), Balance: row.Balances[rule.ID].Format()}
                if b.ws.Config.Consumed(rule.ID) {
                    counter.Debit = row.Debits[rule.ID].Format()
                }
                item.Counters[rule.ID] = counter
            }
            item.WorkingDays, item.Holidays, item.Missing = row.Indicators.WorkingDays.FormatDays(), row.Indicators.Holidays.FormatDays(), row.Indicators.Missing.FormatDays()
        }
        rows = append(rows, item)
    }
    opening := map[string]string{}
    for id, quantity := range summary.Opening {
        opening[id] = quantity.Format()
    }
    return b.encode(map[string]any{"year": year, "opening": opening, "rows": rows})
}

func (b *ToolBox) readConfiguration(raw json.RawMessage) string {
    return b.encode(map[string]any{"public_holidays": b.ws.Config.PublicHolidayZone(), "activities": b.ws.Config.Activities, "counters": b.ws.Config.Counters})
}

func (b *ToolBox) readSettings(raw json.RawMessage) string {
    return b.encode(map[string]any{"settings": b.ws.Calendar.Settings, "adjustments": b.adjustments()})
}

func (b *ToolBox) adjustments() []toolAdjustment {
    adjustments := []toolAdjustment{}
    for _, adjustment := range b.ws.Calendar.Adjustments {
        adjustments = append(adjustments, toolAdjustment{ID: adjustment.ID, Date: adjustment.Date.String(), Counter: adjustment.Counter, Amount: adjustment.Amount.FormatSigned(), Reason: adjustment.Reason})
    }
    return adjustments
}

func (b *ToolBox) propose(tool string) func(arguments json.RawMessage) string {
    return NewToolProposal(b, tool).Run
}

func (b *ToolBox) record(tool string, raw json.RawMessage) string {
    var compact bytes.Buffer
    if (json.Compact(&compact, raw) != nil) || (compact.Len() > assistant.MaxArgumentBytes) {
        return b.refuse(invalidArguments)
    }
    plan, refusal := b.planner.Plan(b.ws, tool, compact.String())
    if refusal != "" {
        return b.refuse(refusal)
    }
    summary := []string{}
    for _, line := range plan.Summary {
        if len(summary) < assistant.MaxSummaryLines {
            summary = append(summary, b.line(line))
        }
    }
    b.pending = &assistant.PendingAction{Tool: tool, Arguments: compact.String(), Summary: summary, Revision: b.ws.Calendar.Revision, Proposed: b.now}
    return b.encode(map[string]any{"status": "action en attente de confirmation par l'utilisateur dans la page", "summary": summary})
}

func (b *ToolBox) line(text string) string {
    text = strings.Join(strings.Fields(text), " ")
    if runes := []rune(text); len(runes) > assistant.MaxSummaryLine {
        return string(runes[:assistant.MaxSummaryLine - 1]) + "…"
    }
    return text
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
