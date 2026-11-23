/*
 * application.go - Copyright (c) 2023-2026 - Olivier Poncet
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
    "context"
    "crypto/rand"
    "crypto/tls"
    "errors"
    "flag"
    "fmt"
    "io"
    "log"
    "log/slog"
    "net"
    "net/http"
    "net/netip"
    "os"
    "strings"
    "sync"
    "time"
    "unicode"

    "timesheet/internal/activity"
    "timesheet/internal/store"
    "timesheet/internal/web"
)

// ---------------------------------------------------------------------------
// Application declaration
// ---------------------------------------------------------------------------

type Application struct {
    args       []string
    lookupEnv  func(string) (string, bool)
    stdin      io.Reader
    stdout     io.Writer
    stderr     io.Writer
    location   *time.Location
    clock      activity.Clock
    random     io.Reader
    iterations int
    ready      chan struct{}
    mu         sync.Mutex
    listen     string
    signals    func()
}

const (
    ExitOK    = 0
    ExitError = 1
    ExitUsage = 2

    ProgramName = "timesheet"

    providerHandshake       = 10 * time.Second
    providerIdleConnections = 16
    providerIdleTimeout     = 90 * time.Second
)

// ---------------------------------------------------------------------------
// Application implementation
// ---------------------------------------------------------------------------

func NewApplication(args []string, lookupEnv func(string) (string, bool), stdin io.Reader, stdout io.Writer, stderr io.Writer, location *time.Location) *Application {
    return &Application{args: args, lookupEnv: lookupEnv, stdin: stdin, stdout: stdout, stderr: stderr, location: location, clock: activity.NewSystemClock(), random: rand.Reader, ready: make(chan struct{}), signals: func() {}}
}

func (a *Application) WithSignals(release func()) *Application {
    a.signals = release
    return a
}

func (a *Application) WithClock(clock activity.Clock) *Application {
    a.clock = clock
    return a
}

func (a *Application) WithIterations(iterations int) *Application {
    a.iterations = iterations
    return a
}

func (a *Application) Ready() <-chan struct{} {
    return a.ready
}

func (a *Application) Listen() string {
    a.mu.Lock()
    defer a.mu.Unlock()
    return a.listen
}

func (a *Application) Run(ctx context.Context) int {
    settings, err := NewSettings(a.args)
    if errors.Is(err, flag.ErrHelp) {
        NewUsage(ProgramName).Write(a.stdout)
        return ExitOK
    }
    if err != nil {
        fmt.Fprintf(a.stderr, "%s: %v\nTry '%s help' for more information.\n", ProgramName, err, ProgramName)
        return ExitUsage
    }
    logger := slog.New(slog.NewTextHandler(a.stderr, &slog.HandlerOptions{Level: settings.LogLevel()}))
    if settings.Command() != "run" {
        a.signals()
    }
    switch settings.Command() {
        case "help":
            NewUsage(ProgramName).Write(a.stdout)
            return ExitOK
        case "version":
            fmt.Fprintln(a.stdout, NewBuildInfo().String())
            return ExitOK
        case "config show", "config set":
            return a.report(NewConfigCommand(settings, logger).Run(a.stdout))
    }
    configuration := NewServerFile(settings.Config())
    server := activity.NewDefaultServerConfiguration()
    if settings.HasConfig() {
        server, err = configuration.Load()
    }
    if (err != nil) && (settings.Command() == "run") {
        return a.fail(logger, "cannot load " + configuration.Path(), err)
    }
    if err == nil {
        err = settings.ResolveDatabase(string(server.Database.Path))
    }
    if err != nil {
        return a.report(err)
    }
    server.Database.Path = activity.NullableText(settings.Database())
    switch settings.Command() {
        case "init":
            return a.report(NewInitCommand(settings, server, logger).Run(a.stdout))
        case "session clear":
            return a.report(NewSessionCommand(settings, server, logger).Run(a.stdout))
        case "import":
            return a.report(NewImportCommand(settings, server, a.clock, a.random, logger).Run(a.stdout))
        case "run":
            return a.serve(ctx, settings, server, a.journal(settings, server, logger))
    }
    return a.report(NewUserCommand(settings, server, NewPasswordReader(a.stdin, a.stderr, a.lookupEnv), a.clock, a.random, a.passwordIterations(server), logger).Run(a.stdout))
}

func (a *Application) journal(settings *Settings, server activity.ServerConfiguration, logger *slog.Logger) *slog.Logger {
    if settings.ExplicitLogLevel() {
        return logger
    }
    level, err := server.Server.Level()
    if err != nil {
        return logger
    }
    return slog.New(slog.NewTextHandler(a.stderr, &slog.HandlerOptions{Level: level}))
}

func (a *Application) passwordIterations(server activity.ServerConfiguration) int {
    if a.iterations > 0 {
        return a.iterations
    }
    return int(server.Passwords.Iterations)
}

func (a *Application) report(err error) int {
    if err == nil {
        return ExitOK
    }
    fmt.Fprintf(a.stderr, "%s: %s\n", ProgramName, a.printable(err.Error()))
    if errors.Is(err, ErrUsage) {
        return ExitUsage
    }
    return ExitError
}

func (a *Application) providerClient(settings activity.AssistantSettings) *http.Client {
    transport := &http.Transport{Proxy: http.ProxyFromEnvironment, ForceAttemptHTTP2: true, TLSHandshakeTimeout: providerHandshake, MaxIdleConns: providerIdleConnections, IdleConnTimeout: providerIdleTimeout}
    return &http.Client{Timeout: settings.Timeout.Std(), Transport: transport, CheckRedirect: a.refuseRedirect}
}

func (a *Application) refuseRedirect(request *http.Request, via []*http.Request) error {
    return http.ErrUseLastResponse
}

func (a *Application) serve(ctx context.Context, settings *Settings, server activity.ServerConfiguration, logger *slog.Logger) int {
    logger.Info("starting", "version", NewBuildInfo().String())
    server, err := NewEnvironmentOverrides(a.lookupEnv).Apply(server)
    if err != nil {
        return a.fail(logger, "invalid environment variable", err)
    }
    config := server.Server
    database := store.NewDatabase(settings.Database(), logger, server.Database.MaxBytes, server.Limits)
    if err := database.Lock(); err != nil {
        return a.fail(logger, "cannot lock " + database.Path(), err)
    }
    defer database.Unlock()
    if err := database.Load(); err != nil {
        return a.fail(logger, "cannot load " + database.Path(), err)
    }
    if database.AdminCount() == 0 {
        logger.Warn("no administrator account", "hint", fmt.Sprintf("stop the server, run '%s user add <login> --profile=ADMIN %s', then start it again", ProgramName, settings.Paths()))
    }
    listen := config.Listen
    if settings.Listen() != "" {
        listen = settings.Listen()
    }
    proxies := config.TrustedProxies
    if len(settings.TrustedProxies()) > 0 {
        proxies = settings.TrustedProxies()
    }
    trusted, err := activity.NewTrustedProxies(proxies)
    if err != nil {
        return a.fail(logger, "invalid trusted proxies", err)
    }
    certificate, key := string(config.TLSCert), string(config.TLSKey)
    if settings.TLS() {
        certificate, key = settings.TLSPair()
    }
    secure := certificate != ""
    listener, err := net.Listen("tcp", listen)
    if err != nil {
        return a.fail(logger, "cannot listen on " + listen, err)
    }
    addresses, prefixes := a.interfaces()
    origins, err := a.origins(settings, config, secure, listener, addresses)
    if err != nil {
        listener.Close()
        return a.fail(logger, "invalid origins", err)
    }
    if secure && !origins.Secure() {
        listener.Close()
        return a.fail(logger, "TLS needs https origins", errors.New("give --origin=https://... or run config set server.origins with https origins, or remove the pair with config set server.tls_cert null server.tls_key null"))
    }
    if secure {
        provider := NewTLSProvider(certificate, key, config.TLSMinVersion)
        tlsConfig, err := provider.Config()
        if err != nil {
            listener.Close()
            return a.fail(logger, "TLS setup failed", err)
        }
        if provider.KeyExposed() {
            logger.Warn("the TLS key is readable by other users: give it mode 0600 or 0640", "path", key)
        }
        listener = tls.NewListener(listener, tlsConfig)
    } else if !origins.Secure() {
        logger.Warn("serving plain HTTP: passwords and session cookies travel in clear; use a TLS pair or a TLS reverse proxy on an untrusted network")
    }
    sessions := store.NewSessionFile(settings.Database(), logger)
    if err := sessions.Prepare(); err != nil {
        listener.Close()
        return a.fail(logger, "cannot prepare " + sessions.Path(), err)
    }
    site, err := web.NewSite(web.SiteDependencies{Repository: database, Sessions: sessions, Clock: a.clock, Random: a.random, Logger: logger, Origins: origins, Server: server, TrustedProxies: trusted, LocalPrefixes: prefixes, Iterations: a.passwordIterations(server), Version: NewBuildInfo().Version(), Client: a.providerClient(server.Assistant), Downloads: store.NewDownloadStore(settings.Database() + activity.DownloadsSuffix, logger)})
    if err != nil {
        listener.Close()
        return a.fail(logger, "cannot build the site", err)
    }
    httpServer := &http.Server{
        Handler:           site.Handler(),
        ReadHeaderTimeout: config.ReadHeaderTimeout.Std(),
        ReadTimeout:       config.ReadTimeout.Std(),
        WriteTimeout:      config.WriteTimeout.Std(),
        IdleTimeout:       config.KeepAliveTimeout.Std(),
        MaxHeaderBytes:    int(config.MaxHeaderBytes),
        ErrorLog:          log.New(NewServerErrorLog(logger), "", 0),
    }
    a.mu.Lock()
    a.listen = listener.Addr().String()
    a.mu.Unlock()
    close(a.ready)
    logger.Info("listening", "address", listener.Addr().String(), "origins", strings.Join(origins.List(), " "), "database", database.Path())
    served := make(chan error, 1)
    go a.run(httpServer, listener, served)
    code := ExitOK
    select {
        case <-ctx.Done():
        case err := <-served:
            if (err != nil) && !errors.Is(err, http.ErrServerClosed) {
                logger.Error("server stopped", "error", err.Error())
                code = ExitError
            }
    }
    shutdown, cancel := context.WithTimeout(context.WithoutCancel(ctx), config.ShutdownTimeout.Std())
    defer cancel()
    if err := httpServer.Shutdown(shutdown); err != nil {
        httpServer.Close()
    }
    site.Stop()
    site.Flush()
    logger.Info("stopped")
    return code
}

func (a *Application) run(server *http.Server, listener net.Listener, served chan<- error) {
    served <- server.Serve(listener)
}

func (a *Application) origins(settings *Settings, config activity.ServerSettings, secure bool, listener net.Listener, addresses []netip.Addr) (*web.Origins, error) {
    if public := settings.Origins(); len(public) > 0 {
        return web.NewPublicOrigins(public)
    }
    if settings.Listen() == "" {
        return web.NewPublicOrigins(config.Origins)
    }
    port := 0
    if tcp, ok := listener.Addr().(*net.TCPAddr); ok {
        port = tcp.Port
    }
    hostname, _ := os.Hostname()
    return web.NewLocalOrigins(addresses, hostname, port, secure), nil
}

func (a *Application) interfaces() ([]netip.Addr, []netip.Prefix) {
    addresses := []netip.Addr{}
    prefixes := []netip.Prefix{}
    list, err := net.InterfaceAddrs()
    if err != nil {
        return addresses, prefixes
    }
    for _, item := range list {
        prefix, err := netip.ParsePrefix(item.String())
        if err != nil {
            continue
        }
        addresses = append(addresses, prefix.Addr().Unmap())
        prefixes = append(prefixes, prefix.Masked())
    }
    return addresses, prefixes
}

func (a *Application) printable(text string) string {
    return strings.Map(a.printableRune, text)
}

func (a *Application) printableRune(r rune) rune {
    if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
        return unicode.ReplacementChar
    }
    return r
}

func (a *Application) fail(logger *slog.Logger, message string, err error) int {
    logger.Error(message, "error", err.Error())
    return ExitError
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
