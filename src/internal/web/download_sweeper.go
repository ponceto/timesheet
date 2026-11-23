/*
 * download_sweeper.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "log/slog"
    "sync"
    "time"

    "timesheet/internal/activity"
)

// ---------------------------------------------------------------------------
// DownloadSweeper declaration
// ---------------------------------------------------------------------------

type DownloadSweeper struct {
    downloads Downloads
    clock     activity.Clock
    settings  activity.AssistantSettings
    logger    *slog.Logger
    ctx       context.Context
    cancel    context.CancelFunc
    wait      sync.WaitGroup
}

// ---------------------------------------------------------------------------
// DownloadSweeper implementation
// ---------------------------------------------------------------------------

func NewDownloadSweeper(downloads Downloads, clock activity.Clock, settings activity.AssistantSettings, logger *slog.Logger) *DownloadSweeper {
    ctx, cancel := context.WithCancel(context.Background())
    return &DownloadSweeper{downloads: downloads, clock: clock, settings: settings, logger: logger, ctx: ctx, cancel: cancel}
}

func (s *DownloadSweeper) Start() {
    s.wait.Add(1)
    go s.run()
}

func (s *DownloadSweeper) Stop() {
    s.cancel()
    s.wait.Wait()
}

func (s *DownloadSweeper) Sweep() int {
    removed := s.downloads.Sweep(s.clock.Now(), s.settings.DownloadTTL.Std())
    if removed > 0 {
        s.logger.Info("downloads removed", "count", removed)
    }
    return removed
}

func (s *DownloadSweeper) run() {
    defer s.wait.Done()
    ticker := time.NewTicker(s.settings.DownloadSweep.Std())
    defer ticker.Stop()
    s.Sweep()
    for {
        select {
            case <-s.ctx.Done():
                return
            case <-ticker.C:
                s.Sweep()
        }
    }
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
