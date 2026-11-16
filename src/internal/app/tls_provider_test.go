/*
 * tls_provider_test.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "os"
    "path/filepath"
    "testing"
)

// ---------------------------------------------------------------------------
// TLSProvider tests
// ---------------------------------------------------------------------------

func TestTLSKeyMayBeReadByItsGroup(t *testing.T) {
    key := filepath.Join(t.TempDir(), "key.pem")
    writeTestFile(t, key, "key")
    for mode, exposed := range map[os.FileMode]bool{0o600: false, 0o640: false, 0o660: true, 0o644: true} {
        os.Chmod(key, mode)
        if NewTLSProvider("cert.pem", key, "").KeyExposed() != exposed {
            t.Errorf("mode %o: exposed must be %v", mode, exposed)
        }
    }
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
