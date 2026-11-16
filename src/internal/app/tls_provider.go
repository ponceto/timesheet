/*
 * tls_provider.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "crypto/tls"
    "errors"
    "fmt"
    "os"

    "timesheet/internal/activity"
)

// ---------------------------------------------------------------------------
// TLSProvider declaration
// ---------------------------------------------------------------------------

type TLSProvider struct {
    certPath   string
    keyPath    string
    minVersion uint16
}

var ErrInvalidCertificate = errors.New("app: invalid TLS certificate pair")

// ---------------------------------------------------------------------------
// TLSProvider implementation
// ---------------------------------------------------------------------------

func NewTLSProvider(certPath string, keyPath string, minVersion string) *TLSProvider {
    provider := &TLSProvider{certPath: certPath, keyPath: keyPath, minVersion: tls.VersionTLS12}
    if minVersion == activity.TLSVersion13 {
        provider.minVersion = tls.VersionTLS13
    }
    return provider
}

func (p *TLSProvider) Config() (*tls.Config, error) {
    certificate, err := tls.LoadX509KeyPair(p.certPath, p.keyPath)
    if err != nil {
        return nil, fmt.Errorf("%w: %v", ErrInvalidCertificate, err)
    }
    return &tls.Config{Certificates: []tls.Certificate{certificate}, MinVersion: p.minVersion, CipherSuites: p.cipherSuites(), NextProtos: []string{"http/1.1"}}, nil
}

func (p *TLSProvider) KeyExposed() bool {
    info, err := os.Stat(p.keyPath)
    return (err == nil) && ((info.Mode().Perm() & 0o037) != 0)
}

func (p *TLSProvider) cipherSuites() []uint16 {
    return []uint16{
        tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
        tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
        tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
        tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
        tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
        tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
    }
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
