/*
 * download_store_test.go - Copyright (c) 2023-2026 - Olivier Poncet
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

package store

import (
    "bytes"
    "errors"
    "log/slog"
    "os"
    "path/filepath"
    "testing"
    "time"

    "timesheet/internal/account"
    "timesheet/internal/assistant"
)

// ---------------------------------------------------------------------------
// DownloadStore tests
// ---------------------------------------------------------------------------

func newTestDownloads(t *testing.T) (*DownloadStore, string) {
    t.Helper()
    directory := filepath.Join(t.TempDir(), "timesheet.json.downloads")
    return NewDownloadStore(directory, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))), directory
}

func TestDownloadStoreKeepsAFileForItsAccountOnly(t *testing.T) {
    downloads, directory := newTestDownloads(t)
    id, err := downloads.Save("jean-pierre", []byte(`{"days":{}}`), time.Now())
    if err != nil {
        t.Fatal(err)
    }
    for path, mode := range map[string]os.FileMode{directory: 0o700, filepath.Join(directory, id + "-jean-pierre.json"): 0o600} {
        if info, err := os.Stat(path); (err != nil) || (info.Mode().Perm() != mode) {
            t.Fatalf("%s: %v %v", path, info, err)
        }
    }
    download, err := downloads.Open("jean-pierre", id, time.Now(), 30 * time.Minute)
    if (err != nil) || (string(download.Data) != `{"days":{}}`) || download.Created.IsZero() {
        t.Fatalf("got %+v %v", download, err)
    }
    for _, attempt := range [][2]string{{"other", id}, {"jean-pierre", "../" + id}, {"jean-pierre", "0123"}} {
        if _, err := downloads.Open(account.Login(attempt[0]), attempt[1], time.Now(), 30 * time.Minute); !errors.Is(err, assistant.ErrDownloadMissing) {
            t.Fatalf("%v: got %v", attempt, err)
        }
    }
    if _, err := downloads.Open("jean-pierre", id, time.Now().Add(31 * time.Minute), 30 * time.Minute); !errors.Is(err, assistant.ErrDownloadExpired) {
        t.Fatalf("an old file is expired: %v", err)
    }
    if _, err := downloads.Open("jean-pierre", id, time.Now(), 30 * time.Minute); !errors.Is(err, assistant.ErrDownloadMissing) {
        t.Fatalf("an expired file is removed: %v", err)
    }
}

func TestDownloadStoreSweepsAndBoundsTheFiles(t *testing.T) {
    downloads, directory := newTestDownloads(t)
    for range MaxDownloadsPerAccount + 3 {
        if _, err := downloads.Save("demo", []byte("{}"), time.Now()); err != nil {
            t.Fatal(err)
        }
    }
    kept, _ := os.ReadDir(directory)
    if len(kept) != MaxDownloadsPerAccount {
        t.Fatalf("got %d files", len(kept))
    }
    other, _ := downloads.Save("root", []byte("{}"), time.Now())
    if removed := downloads.Sweep(time.Now(), 30 * time.Minute); removed != 0 {
        t.Fatalf("nothing is expired yet: %d", removed)
    }
    downloads.RemoveAccount("demo")
    if left, _ := os.ReadDir(directory); len(left) != 1 {
        t.Fatalf("got %d files", len(left))
    }
    if removed := downloads.Sweep(time.Now().Add(time.Hour), 30 * time.Minute); removed != 1 {
        t.Fatalf("got %d", removed)
    }
    if _, err := downloads.Open("root", other, time.Now(), 30 * time.Minute); !errors.Is(err, assistant.ErrDownloadMissing) {
        t.Fatalf("got %v", err)
    }
}

func TestDownloadStoreRefusesAnOpenDirectory(t *testing.T) {
    downloads, directory := newTestDownloads(t)
    if err := os.Mkdir(directory, 0o755); err != nil {
        t.Fatal(err)
    }
    if _, err := downloads.Save("demo", []byte("{}"), time.Now()); !errors.Is(err, ErrInsecureDirectory) {
        t.Fatalf("got %v", err)
    }
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
