/*
 * login.go - Copyright (c) 2023-2026 - Olivier Poncet
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

package account

import (
    "errors"
    "regexp"
)

// ---------------------------------------------------------------------------
// Login declaration
// ---------------------------------------------------------------------------

type Login string

var ErrInvalidLogin = errors.New("account: invalid login ([a-z0-9][a-z0-9._-]{1,31})")

// ---------------------------------------------------------------------------
// Login implementation
// ---------------------------------------------------------------------------

func NewLogin(text string) (Login, error) {
    login := Login(text)
    if !login.Valid() {
        return "", ErrInvalidLogin
    }
    return login, nil
}

func (l Login) Valid() bool {
    return regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{1,31}$`).MatchString(string(l))
}

func (l Login) Equal(other Login) bool {
    return l == other
}

func (l Login) String() string {
    return string(l)
}

func (l *Login) UnmarshalText(data []byte) error {
    login, err := NewLogin(string(data))
    if err != nil {
        return err
    }
    *l = login
    return nil
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
