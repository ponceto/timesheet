/*
 * provider_transport.go - Copyright (c) 2023-2026 - Olivier Poncet
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

package assistant

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
)

// ---------------------------------------------------------------------------
// ProviderTransport declaration
// ---------------------------------------------------------------------------

type ProviderTransport struct {
    client *http.Client
}

const maxResponseBytes = 4 << 20

// ---------------------------------------------------------------------------
// ProviderTransport implementation
// ---------------------------------------------------------------------------

func NewProviderTransport(client *http.Client) *ProviderTransport {
    return &ProviderTransport{client: client}
}

func (t *ProviderTransport) Post(ctx context.Context, url string, headers map[string]string, payload any, answer any) error {
    body, err := json.Marshal(payload)
    if err != nil {
        return err
    }
    request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
    if err != nil {
        return err
    }
    request.Header.Set("Content-Type", "application/json")
    request.Header.Set("Accept", "application/json")
    for name, value := range headers {
        request.Header.Set(name, value)
    }
    response, err := t.client.Do(request)
    if err != nil {
        return err
    }
    defer response.Body.Close()
    data, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes + 1))
    if err != nil {
        return err
    }
    if (response.StatusCode < 200) || (response.StatusCode > 299) {
        return &ProviderError{Status: response.StatusCode}
    }
    if len(data) > maxResponseBytes {
        return fmt.Errorf("%w: more than %d bytes", ErrMalformedAnswer, maxResponseBytes)
    }
    if err := json.Unmarshal(data, answer); err != nil {
        return fmt.Errorf("%w: %v", ErrMalformedAnswer, err)
    }
    return nil
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
