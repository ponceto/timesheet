/*
 * month_editor.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "io"
    "strings"

    "timesheet/internal/activity"
)

// ---------------------------------------------------------------------------
// MonthEditor declaration
// ---------------------------------------------------------------------------

type MonthEditor struct {
    random io.Reader
}

const (
    lockedMonth       = "Ce mois n'est pas modifiable : il est validé, précède le début du suivi ou dépasse l'horizon de planification de 24 mois."
    invalidEntries    = "Certaines saisies sont invalides, rien n'a été enregistré."
    invalidDate       = "Date d'ajustement invalide."
    closedAdjustment  = "Un ajustement se saisit dans un mois ouvert, à partir du début du suivi."
    unknownCounter    = "Compteur inconnu."
    invalidAmount     = "Quantité invalide : un nombre signé non nul entre -366 et 366, par exemple -1,5 ou 7/12."
    reasonRequired    = "Le motif est obligatoire (une ligne, 200 caractères au plus)."
    internalError     = "Erreur interne."
    frozenAdjustment  = "Cet ajustement appartient à un mois validé."
    unknownAdjustment = "Ajustement inconnu."
)

// ---------------------------------------------------------------------------
// MonthEditor implementation
// ---------------------------------------------------------------------------

func NewMonthEditor(random io.Reader) *MonthEditor {
    return &MonthEditor{random: random}
}

func (e *MonthEditor) Editable(ws *Workspace, m activity.Month) bool {
    return !ws.Calendar.IsValidated(m) && !m.Before(ws.Calendar.Settings.Start.Month()) && !m.After(ws.Horizon())
}

func (e *MonthEditor) Apply(ws *Workspace, calendar *activity.Calendar, entries map[activity.Date]activity.DayEntry) map[activity.Date][]string {
    problems := map[activity.Date][]string{}
    for d, entry := range entries {
        if found := ws.Book.CheckEntry(d, calendar.Days[d], entry); len(found) > 0 {
            problems[d] = found
            continue
        }
        if entry.Empty() {
            delete(calendar.Days, d)
        } else {
            calendar.Days[d] = entry
        }
    }
    return problems
}

func (e *MonthEditor) AddAdjustment(ws *Workspace, calendar *activity.Calendar, date string, counter string, amount string, reason string) (activity.Month, string) {
    d, err := activity.NewDateFromText(date)
    if err != nil {
        return activity.Month{}, invalidDate
    }
    m := d.Month()
    if !calendar.Tracks(d) || calendar.IsValidated(m) || m.After(ws.Horizon()) {
        return m, closedAdjustment
    }
    rule, known := ws.Config.Counter(counter)
    if !known || rule.Archived {
        return m, unknownCounter
    }
    quantity, err := activity.NewQuantityFromText(strings.Replace(strings.TrimSpace(amount), ",", ".", 1))
    if (err != nil) || quantity.IsZero() || (quantity.Abs().Cmp(activity.NewQuantity(MaxDays, 1)) > 0) {
        return m, invalidAmount
    }
    reason = strings.TrimSpace(reason)
    if (reason == "") || !activity.NewTextPolicy().Accepts(reason, activity.MaxReasonLength) {
        return m, reasonRequired
    }
    id, err := activity.NewIdentifierGenerator(e.random).Next()
    if err != nil {
        return m, internalError
    }
    calendar.Adjustments = append(calendar.Adjustments, activity.Adjustment{ID: id, Date: d, Counter: rule.ID, Amount: quantity, Reason: reason})
    return m, ""
}

func (e *MonthEditor) DeleteAdjustment(calendar *activity.Calendar, id string) (activity.Month, string) {
    for i, adjustment := range calendar.Adjustments {
        if adjustment.ID != id {
            continue
        }
        m := adjustment.Date.Month()
        if calendar.IsValidated(m) {
            return m, frozenAdjustment
        }
        calendar.Adjustments = append(calendar.Adjustments[:i:i], calendar.Adjustments[i + 1:]...)
        return m, ""
    }
    return activity.Month{}, unknownAdjustment
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
