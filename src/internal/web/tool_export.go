/*
 * tool_export.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "log/slog"
    "time"

    "timesheet/internal/activity"
    "timesheet/internal/assistant"
)

// ---------------------------------------------------------------------------
// ToolExport declaration
// ---------------------------------------------------------------------------

type ToolExport struct {
    downloads Downloads
    ttl       time.Duration
    logger    *slog.Logger
}

const (
    ToolExportData = "export_data"
    exportNote     = "Donne ce lien tel quel à l'utilisateur, en lien Markdown [nom du fichier](url), et précise l'heure d'expiration ; n'invente jamais de lien."
)

// ---------------------------------------------------------------------------
// ToolExport implementation
// ---------------------------------------------------------------------------

func NewToolExport(downloads Downloads, ttl time.Duration, logger *slog.Logger) *ToolExport {
    return &ToolExport{downloads: downloads, ttl: ttl, logger: logger}
}

func (e *ToolExport) Run(ws *Workspace, now time.Time) map[string]any {
    data, err := activity.NewImportFileOf(ws.Calendar, ws.Config).Encode()
    if err != nil {
        e.logger.Error("cannot export the data", "error", err.Error())
        return map[string]any{"error": "export impossible"}
    }
    id, err := e.downloads.Save(ws.Owner.Login, data, now)
    if err != nil {
        e.logger.Error("cannot save the download", "error", err.Error())
        return map[string]any{"error": "export impossible"}
    }
    e.logger.Info("data exported", "login", string(ws.Owner.Login), "by", "assistant")
    expires := now.Add(e.ttl).In(ws.Location)
    return map[string]any{"url": assistant.DownloadsPath + id, "filename": ws.ExportName(ws.Today), "expires": expires.Format("15:04"), "content": "réglages, jours avec leurs commentaires et ajustements, au format d'import", "note": exportNote}
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
