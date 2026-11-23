/*
 * admin_server_controller.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "net/http"
    "strconv"

    "timesheet/internal/activity"
)

// ---------------------------------------------------------------------------
// AdminServerController declaration
// ---------------------------------------------------------------------------

type AdminServerController struct {
    server   activity.ServerConfiguration
    version  string
    renderer *Renderer
}

type serverRecord struct {
    Groups []serverGroupRecord
}

type serverGroupRecord struct {
    Title string
    Facts []serverFactRecord
}

type serverFactRecord struct {
    Label  string
    Values []string
}

// ---------------------------------------------------------------------------
// AdminServerController implementation
// ---------------------------------------------------------------------------

func NewAdminServerController(server activity.ServerConfiguration, version string, renderer *Renderer) *AdminServerController {
    return &AdminServerController{server: server, version: version, renderer: renderer}
}

func (c *AdminServerController) Show(w http.ResponseWriter, r *http.Request) {
    c.renderer.Render(w, r, http.StatusOK, "admin_server", PageModel{Section: "server", Data: serverRecord{Groups: c.groups()}})
}

func (c *AdminServerController) groups() []serverGroupRecord {
    server, database, sessions, login, limits, assistant := c.server.Server, c.server.Database, c.server.Sessions, c.server.Login, c.server.Limits, c.server.Assistant
    return []serverGroupRecord{
        {Title: "Logiciel", Facts: []serverFactRecord{
            {"Nom", c.one("Timesheet")},
            {"Version", c.one(c.version)},
            {"Licence", c.one("GNU GPL version 2 ou ultérieure")},
        }},
        {Title: "Réseau", Facts: []serverFactRecord{
            {"Adresse d'écoute", c.one(server.Listen)},
            {"Origines publiques", c.many(server.Origins)},
            {"Proxys de confiance", c.many(server.TrustedProxies)},
            {"Fuseau horaire", c.one(server.Timezone)},
            {"Niveau du journal", c.one(server.LogLevel)},
            {"Certificat TLS", c.one(c.optional(string(server.TLSCert)))},
            {"Clé TLS", c.one(c.optional(string(server.TLSKey)))},
            {"Version TLS minimale", c.one(server.TLSMinVersion)},
            {"Durée HSTS", c.one(server.HSTSMaxAge.Label())},
        }},
        {Title: "Requêtes", Facts: []serverFactRecord{
            {"Lecture des en-têtes", c.one(server.ReadHeaderTimeout.Label())},
            {"Lecture de la requête", c.one(server.ReadTimeout.Label())},
            {"Écriture de la réponse", c.one(server.WriteTimeout.Label())},
            {"Connexion inactive", c.one(server.KeepAliveTimeout.Label())},
            {"Arrêt du serveur", c.one(server.ShutdownTimeout.Label())},
            {"Taille des en-têtes", c.one(c.size(server.MaxHeaderBytes))},
            {"Taille du corps", c.one(c.size(server.MaxBodyBytes))},
            {"Longueur de l'adresse", c.one(strconv.FormatInt(server.MaxURLBytes, 10) + " octets")},
        }},
        {Title: "Base de données", Facts: []serverFactRecord{
            {"Fichier", c.one(string(database.Path))},
            {"Taille maximale", c.one(c.size(database.MaxBytes))},
        }},
        {Title: "Sessions", Facts: []serverFactRecord{
            {"Expiration sans activité", c.one(sessions.IdleTimeout.Label())},
            {"Durée maximale", c.one(sessions.MaxLifetime.Label())},
            {"Sessions par compte", c.one(strconv.FormatInt(sessions.PerAccount, 10))},
            {"Enregistrement des sessions", c.one(sessions.SaveInterval.Label())},
        }},
        {Title: "Connexion", Facts: []serverFactRecord{
            {"Validité du formulaire", c.one(login.FormLifetime.Label())},
            {"Premier délai", c.one(login.InitialDelay.Label())},
            {"Délai maximal", c.one(login.MaxDelay.Label())},
            {"Fenêtre du budget", c.one(login.BudgetWindow.Label())},
            {"Nouvelles adresses par fenêtre", c.one(strconv.FormatInt(login.NewAddressesPerWindow, 10))},
            {"Adresses suivies", c.one(strconv.FormatInt(login.TrackedAddresses, 10))},
            {"Oubli d'une adresse", c.one(login.ForgetAfter.Label())},
        }},
        {Title: "Mots de passe", Facts: []serverFactRecord{
            {"Itérations PBKDF2", c.one(strconv.FormatInt(c.server.Passwords.Iterations, 10))},
        }},
        {Title: "Limites", Facts: []serverFactRecord{
            {"Comptes", c.one(strconv.FormatInt(limits.Accounts, 10))},
            {"Taille d'un compte", c.one(c.size(limits.AccountBytes))},
            {"Ajustements par compte", c.one(strconv.FormatInt(limits.Adjustments, 10))},
        }},
        {Title: "Assistant", Facts: []serverFactRecord{
            {"Délai d'un appel au fournisseur", c.one(assistant.Timeout.Label())},
            {"Tours d'outils par question", c.one(strconv.FormatInt(assistant.MaxRounds, 10))},
            {"Jetons par réponse", c.one(strconv.FormatInt(assistant.MaxTokens, 10))},
            {"Conversations par compte", c.one(strconv.FormatInt(assistant.Conversations, 10))},
            {"Échanges par conversation", c.one(strconv.FormatInt(assistant.Exchanges, 10))},
            {"Taille d'une question", c.one(c.size(assistant.QuestionBytes))},
            {"Durée d'un lien de téléchargement", c.one(assistant.DownloadTTL.Label())},
            {"Nettoyage des téléchargements", c.one(assistant.DownloadSweep.Label())},
        }},
    }
}

func (c *AdminServerController) one(value string) []string {
    return []string{value}
}

func (c *AdminServerController) many(values []string) []string {
    if len(values) == 0 {
        return c.one(c.optional(""))
    }
    return values
}

func (c *AdminServerController) optional(value string) string {
    if value == "" {
        return "aucun"
    }
    return value
}

func (c *AdminServerController) size(bytes int64) string {
    switch {
        case (bytes % (1 << 20)) == 0:
            return strconv.FormatInt(bytes >> 20, 10) + " Mio"
        case (bytes % (1 << 10)) == 0:
            return strconv.FormatInt(bytes >> 10, 10) + " Kio"
    }
    return strconv.FormatInt(bytes, 10) + " octets"
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
