/*
 * key_ring.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "encoding/base64"
    "errors"
    "fmt"
    "log/slog"
    "strings"
    "sync"
)

// ---------------------------------------------------------------------------
// KeyRing declaration
// ---------------------------------------------------------------------------

type KeyRing struct {
    mu     sync.Mutex
    file   *SecureFile
    logger *slog.Logger
    aead   cipher.AEAD
}

const (
    keyBytes    = 32
    maxKeyBytes = 1 << 10
)

var (
    ErrMalformedKeyFile = errors.New("store: malformed key file")
    ErrUnreadableSecret = errors.New("store: the secret cannot be opened with the key file")
)

// ---------------------------------------------------------------------------
// KeyRing implementation
// ---------------------------------------------------------------------------

func NewKeyRing(path string, logger *slog.Logger) *KeyRing {
    return &KeyRing{file: NewSecureFile(path, logger, maxKeyBytes), logger: logger}
}

func (k *KeyRing) Path() string {
    return k.file.Path()
}

func (k *KeyRing) Seal(plain string, context string) (string, error) {
    aead, err := k.cipher(true)
    if err != nil {
        return "", err
    }
    nonce := make([]byte, aead.NonceSize(), aead.NonceSize() + len(plain) + aead.Overhead())
    if _, err := rand.Read(nonce); err != nil {
        return "", err
    }
    return base64.StdEncoding.EncodeToString(aead.Seal(nonce, nonce, []byte(plain), []byte(context))), nil
}

func (k *KeyRing) Open(sealed string, context string) (string, error) {
    aead, err := k.cipher(false)
    if err != nil {
        return "", err
    }
    data, err := base64.StdEncoding.DecodeString(sealed)
    if (err != nil) || (len(data) < aead.NonceSize()) {
        return "", ErrUnreadableSecret
    }
    plain, err := aead.Open(nil, data[:aead.NonceSize()], data[aead.NonceSize():], []byte(context))
    if err != nil {
        return "", ErrUnreadableSecret
    }
    return string(plain), nil
}

func (k *KeyRing) cipher(create bool) (cipher.AEAD, error) {
    k.mu.Lock()
    defer k.mu.Unlock()
    if k.aead != nil {
        return k.aead, nil
    }
    data, found, err := k.file.Read()
    if err != nil {
        return nil, err
    }
    var key []byte
    switch {
        case found:
            key, err = base64.StdEncoding.DecodeString(strings.TrimSpace(string(data)))
            if (err != nil) || (len(key) != keyBytes) {
                return nil, fmt.Errorf("%w: %s", ErrMalformedKeyFile, k.Path())
            }
        case !create:
            return nil, ErrUnreadableSecret
        default:
            key = make([]byte, keyBytes)
            if _, err := rand.Read(key); err != nil {
                return nil, err
            }
            if err := k.file.Write([]byte(base64.StdEncoding.EncodeToString(key) + "\n")); err != nil {
                return nil, err
            }
            k.logger.Info("key file created", "file", k.Path())
    }
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }
    if k.aead, err = cipher.NewGCM(block); err != nil {
        return nil, err
    }
    return k.aead, nil
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
