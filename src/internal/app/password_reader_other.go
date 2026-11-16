//go:build !linux

/*
 * password_reader_other.go - Copyright (c) 2023-2026 - Olivier Poncet
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

package app

import (
    "errors"
    "fmt"
    "os"
)

// ---------------------------------------------------------------------------
// PasswordReader implementation (other platforms)
// ---------------------------------------------------------------------------

func (p *PasswordReader) hideEcho(file *os.File) (func(), error) {
    if info, err := file.Stat(); (err == nil) && ((info.Mode() & os.ModeCharDevice) != 0) {
        fmt.Fprintln(p.stderr, "warning: the password is read with echo on this system; prefer " + PasswordVariable + " or a pipe")
    }
    return nil, errors.New("app: hiding the terminal echo is only supported on Linux")
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
