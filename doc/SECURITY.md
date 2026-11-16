# SECURITY

This document describes what Timesheet protects, against whom, with which defences, and which risks it accepts.

It is written for a security auditor, for the operator who exposes the server, and for a developer or an AI agent about to change a guard.

Other documents hold the rest:

- the observable behaviour, the limits and the log fields: [SPECIFICATIONS.md](SPECIFICATIONS.md);
- the rules that the code follows: [Security rules for code](CODING.md#security-rules-for-code) in `CODING.md`;
- the order of the guards in the server: [HTTP chain](ARCHITECTURE.md#http-chain) in `ARCHITECTURE.md`;
- the procedures (network, backup, upgrade): [OPERATIONS.md](OPERATIONS.md); the first start: [README](../README.md#getting-started).

## CONTENTS

- [ASSETS AND DEPLOYMENT](#assets-and-deployment)
- [THREAT MODEL](#threat-model)
- [ACCEPTED RISKS](#accepted-risks)
- [AUTHORIZATION](#authorization)
- [PASSWORDS](#passwords)
- [LOGIN THROTTLING](#login-throttling)
- [CLIENT ADDRESS AND TRUSTED PROXIES](#client-address-and-trusted-proxies)
- [SESSIONS](#sessions)
- [FORMS AND ORIGINS](#forms-and-origins)
- [PAGES](#pages)
- [AVAILABILITY](#availability)
- [DATA INTEGRITY](#data-integrity)
- [FILES](#files)
- [TRANSPORT](#transport)
- [LOGS](#logs)
- [DATA PROTECTION](#data-protection)
- [REPORTING A VULNERABILITY](#reporting-a-vulnerability)

## ASSETS AND DEPLOYMENT

### What the application holds

- Personal data of each user:
  - working days;
  - absences with their reasons;
  - comments, some of which reveal health or family events (the `ABS` code and its comment, for example).
- Password hashes.
- The full inventory, with who reads what and for how long, is in [Personal data](SPECIFICATIONS.md#personal-data).

### Expected deployment

- A handful of accounts.
- Usually a single host, or a server behind a reverse proxy, installed from the Debian package or run in a container.

### Actors

| Actor | Who | Access |
| --- | --- | --- |
| User | A person with an account, profile `USER` or `ADMIN`. | The web pages of their own account. |
| Administrator | A person whose account has the profile `ADMIN`. | In addition, the administration pages. |
| Operator | A person with a shell on the host. | The commands, the database file, the session file, the logs, the backups. |

The profiles and their pages are defined in [Profiles and accounts](SPECIFICATIONS.md#profiles-and-accounts).

## THREAT MODEL

Each defence is one line here; the linked section gives the detail.

| Asset | Threat | Defence | Section |
| --- | --- | --- | --- |
| Activity of a user | Read or changed by another user | Every route that writes activity acts on the account of the session; the administration routes of an `ADMIN` change accounts, never their activity; an `ADMIN` reads other accounts through read-only routes. | [Authorization](#authorization) |
| Accounts | Password guessing | PBKDF2-SHA256 hashes, a password policy, a delay per client address and per login after each failure. | [Passwords](#passwords), [Login throttling](#login-throttling) |
| Accounts | Enumeration | Same answer and similar timing whether the login exists or not. | [Enumeration](#enumeration) |
| Sessions | Theft, fixation | Random identifiers stored hashed, new identifier at each login, strict cookie attributes, expiry. | [Sessions](#sessions) |
| Forms | Cross-site request forgery (CSRF) | A token on every form and an origin check on every unsafe request. | [Forms and origins](#forms-and-origins) |
| Pages | Cross-site scripting (XSS), spoofing | Escaping templates, a strict content security policy, refusal of invisible characters. | [Pages](#pages) |
| Pages | Clickjacking, content sniffing | Framing forbidden, declared content types enforced. | [Pages](#pages) |
| Host | DNS rebinding | Only the accepted origins are served. | [Host check](#host-check) |
| Service | Oversized requests, storage exhaustion | Size limits, timeouts, bounded data per account. | [Availability](#availability) |
| Counters | Wrong balances | Exact rationals, overflow detection, round-trip check of every write. | [Data integrity](#data-integrity) |
| Database | Tampering through the file system | Mode, owner and directory checks; atomic writes. | [Files](#files) |
| Database | Concurrent writers | One exclusive lock; optimistic revisions between browsers. | [Data integrity](#data-integrity) |
| Logs | Leaking secrets | No password, typed login, session identifier, CSRF token, query string or body. | [Logs](#logs) |
| Network traffic | Eavesdropping, cookie theft | TLS on a network, `Secure` cookies and HSTS with https origins. | [Transport](#transport) |

Glossary of the terms of this table:

- CSRF: a foreign page makes the browser of a connected user send a request to Timesheet.
- XSS: a script injected into a page runs with the rights of the user who views it.
- Clickjacking: a foreign page shows Timesheet in a hidden frame to trick clicks.
- Content sniffing: a browser guesses a type other than the declared one, for example runs a text file as a script.
- DNS rebinding: a foreign name is made to resolve to the server, so that a page of that name can call it.

## ACCEPTED RISKS

Each row is a known limit that the application does not remove. The column "Bound or reason" says why it stays acceptable, when a reason exists.

| Risk | Condition | Bound or reason | Mitigation |
| --- | --- | --- | --- |
| An administrator can take over an account. | An `ADMIN` resets the password of another account, then logs in with it. | Administrators are trusted. The reset is logged with its author, and the user can no longer log in with the old password. | Give `ADMIN` to as few people as possible; read the `password reset` events. |
| No account is ever locked. | An attacker guesses the password of one login from many addresses. | Each login also waits after its failures, whatever the address: one guess every `login.max_delay` per login at most once the delay is at its ceiling, about 288 a day by default ([Per login](#per-login)). Guesses spread over many logins are bounded by the delays per address and the [global budget](#global-budget). | Strong passwords (the policy); throttling. |
| A user cannot log in while an attacker fails on the login. | An attacker who knows a login fails on it on purpose, once per `login.max_delay`, even from a single address. | The login waits up to `login.max_delay` (5 min by default) from every address, its owner's included, and the attacker can renew the wait as long as it goes on. The account is never locked: the wait ends `login.max_delay` after the attacker stops, and an open session stays open. | The `login failed` events show the address: block it before the server, or close its access. |
| No second factor. | A password is known to an attacker. | Login takes a login and a password only. | Strong passwords; close the sessions of an exposed account. |
| A legitimate user waits up to `login.max_delay` (5 min by default). | During an attack that exhausts the global budget, the user mistypes once. | The budget alone never refuses a login. | None in the application. |
| New addresses are refused. | The throttle table holds `login.tracked_addresses` blocked addresses at once (4096 by default). | The attacker needs more addresses than that. | None in the application. |
| Processor load from the check of an import. | A signed-in user posts the same import file again and again with « Vérifier » and « Valider les mois importés ». | Each check recomputes the counters month by month, a cost that grows with the square of the number of months: about half a second for eleven years, within the default `server.max_body_bytes`. Nothing is written. Accepted for this deployment. | Keep `server.max_body_bytes` at its default; a reverse proxy that limits the request rate. |
| Processor load from parallel password checks. | An attacker holds many addresses. | One check at a time per address, no global limit; about half a second per check. Accepted for this deployment. | A reverse proxy that limits the request rate. |
| Unbounded write load by a connected user. | A user saves again and again. | Writes are not rate-limited, and each one rewrites the whole database file, so the disk load grows with the database size. Accepted for a handful of accounts. | None in the application. |
| Derived origins can be answered by another host. | `run --listen` without `--origin`. | `<host name>.local` is resolved by mDNS, and the bare host name may be resolved by LLMNR or a DNS search domain. | Give `--origin`. See [Host check](#host-check). |
| A forged client address. | A machine inside a trusted proxy range, or a local user of the proxy host, reaches the server directly. | A trusted proxy is recognised by its source address only. | Trust the narrowest set; keep the listening port out of reach. See [Client address and trusted proxies](#client-address-and-trusted-proxies). |
| One shared throttle entry. | A trusted proxy sends no readable `X-Forwarded-For`. | Every such request has no client address. One failure then delays every user. | Configure the proxy to append the header. |
| Plain HTTP between a reverse proxy and the server. | The proxy runs on another machine. | Passwords and cookies cross that link in clear, and the server does not warn, because its origins are https. | A firewall that admits only the proxy, or TLS on that link too. See [Transport](#transport). |
| Plain HTTP on a shared host. | Another user or web service on the same host. | Cookies are shared by every port of a host name. | TLS even on the loopback. See [Transport](#transport). |
| Group members can delete or replace the files. | The database directory is writable by its group. | A warning is logged. | Remove the group write right. See [Files](#files). |
| Another user can block the program. | The database is in a sticky directory shared with other users. | That user creates the lock or session file first. | Use a directory of the service account. |
| An administrator with a public password. | The example database is served on a network or used as a real database. | Its account `demo`, profile `ADMIN`, has the documented password `demonstration`. | Use it only on the loopback, in a copy; never as a real database. See [Example database](OPERATIONS.md#example-database). |
| An administrator with a default password. | A container starts on a new database without `TIMESHEET_PASSWORD`, or with an empty one. | Its administrator, `admin` unless `TIMESHEET_USERNAME` names another login, has the documented password `Tms@Admin%26`, provisional: the first login must change it, but whoever logs in first takes the account. | Give `TIMESHEET_PASSWORD` at the first start, or log in and change the password before the port is reachable from the network. See [Container](OPERATIONS.md#container). |
| A permanent password known to the operator. | `user add` or `user passwd` set the password. | The operator typed it. | Ask the user to change it from the profile page (« Mon profil »). |
| The session file gives the CSRF tokens. | Someone reads the session file. | It does not give the means to use a session. | Protect it like the database. |
| The database and the backups are not encrypted. | root, the service account or a holder of a backup reads the file. | Every comment and absence reason is readable in clear. | File modes, protected backups. See [Data protection](#data-protection). |
| Login timing differs for some accounts. | A stored hash has a number of iterations other than `passwords.iterations`. | The decoy hash uses the configured number. | Change `passwords.iterations` rarely; a password change rewrites the hash with the configured number. |
| The operator weakens a setting. | Whoever can write the server configuration lowers a duration, a delay or a limit. | Every setting is bounded by the program: at least 600 000 iterations, an initial login delay of at least 100 ms, sessions of at most 30 days, at most 16 384 sessions in all, HSTS of at least 24 h. The file is owned by root on a package; in the container image, it belongs to the service account, so that `config set` works through `docker exec`. | Protect the file like the binary; read `config show --config=<file>` after a change. |
| A security fix of Go needs a rebuild. | A vulnerability is fixed in the Go standard library. | The binary is statically linked; the container image is built with the Go version named in its `Dockerfile`. | Rebuild and reinstall the binary or the package; raise the Go version of the `Dockerfile` when it leaves support, then rebuild the image. |

Glossary of the terms of this table:

- mDNS (multicast DNS): name resolution answered by any host of the local network, used for names ending in `.local`.
- LLMNR (Link-Local Multicast Name Resolution): a similar mechanism for bare names.
- Decoy hash: a hash computed for an unknown login so that its answer takes as long as for a known one.

## AUTHORIZATION

### Own account

- Every route that writes activity works on the account of the session.
- No route that writes activity takes the account from the URL or the form.
- The export, the import and the deletion of the data page « Données » (`/dashboard/data/export`, `/dashboard/data/import`, `/dashboard/data/reset`) work on the account of the session only; an `ADMIN` cannot download, import nor erase the data of another account.
- The deletion of the data asks the login retyped and the CSRF token of the session; it cannot be undone, except by restoring a backup.

### Another account

An `ADMIN` reads the activity of other accounts through read-only routes. On another account, an `ADMIN` can only do the following:

| Action of an `ADMIN` on another account | Allowed |
| --- | --- |
| Change its name | Yes |
| Change its profile | Yes |
| Disable or enable it | Yes |
| Close its sessions | Yes |
| Reset its password (the user must then change it) | Yes |
| Delete it with its data | Yes |
| Change its activity, settings or adjustments | No |

### Own account through the administration

An `ADMIN` cannot change its own profile, reset its own password, disable or delete its own account from the administration pages. These requests are refused with status 422.

The pages and routes of each profile are in [Profiles and accounts](SPECIFICATIONS.md#profiles-and-accounts) and [Routes](SPECIFICATIONS.md#routes).

## PASSWORDS

### Storage

- Hash: PBKDF2-SHA256 (password-based key derivation, with SHA-256).
- Iterations: `passwords.iterations` of the [server configuration](SPECIFICATIONS.md#section-passwords) for every new hash, 600 000 by default and at least, 1 200 000 at most.
- Salt: 16 random bytes per hash.
- A stored hash is accepted with 1 000 to 1 200 000 iterations when the database is read.

### Policy

| Rule | Value |
| --- | --- |
| Length | At least 12 characters |
| Distinct characters | At least 6 |
| Surrounding white space | Refused at the start and at the end |
| Forbidden characters | Control, format (Unicode category Cf) and line or paragraph separators |
| Encoding | Valid UTF-8 |
| Size | At most 256 bytes |
| Login | Different from the login, whatever the case |
| Change from the profile page (« Mon profil ») | Different from the current password |

### Enumeration

- A failed login gives the same message and the same status whether the login exists, the password is wrong or the account is disabled.
- For an unknown login, a decoy hash is checked, so that the timing stays similar.
- Limit: the timing is similar only for stored hashes with the default number of iterations (see [Accepted risks](#accepted-risks)).

### Concurrent change

- A password change from the profile page is written only if the stored hash is still the one that was verified.
- A reset by an `ADMIN` made meanwhile therefore wins.
- The change is then refused, and no session is opened.

### Provisional and permanent passwords

| Set by | Kind | Consequence |
| --- | --- | --- |
| Account creation in the administration pages | Provisional | The user must change it at the next login. |
| Password reset in the administration pages | Provisional | The user must change it at the next login. |
| `user add --provisional`, which the container image runs for its administrator at its first start | Provisional | The user must change it at the next login. |
| `user add` without `--provisional`, or `user passwd` | Permanent | Known to the operator until the user changes it. |

The rule of the forced change is defined in [Profiles and accounts](SPECIFICATIONS.md#profiles-and-accounts).

### On the command line

`user add` and `user passwd` read the password from the first available source:

1. the environment variable `TIMESHEET_PASSWORD`;
2. the terminal, without echo;
3. one line of the standard input.

A password is never read from an argument.

A variable set on the command line stays in the shell history. Prefer the terminal prompt, or a variable read from a protected file.

The usage text (`timesheet help`) lists these sources in another order; the order above is the one the program applies.

## LOGIN THROTTLING

Throttling delays the next login attempt of a client address, and of a login, after a failure.

### Parameters

The values are set in the [section login](SPECIFICATIONS.md#section-login) of the server configuration; the table gives the defaults and the bounds.

| Parameter | Default | Setting and bounds |
| --- | --- | --- |
| First delay after a failure | 1 s | `login.initial_delay`, 100 ms to 10 s, at most the longest delay |
| Growth | Doubled at each further failure | fixed |
| Longest delay | 5 min | `login.max_delay`, 1 s to 1 h |
| Global budget | 100 new addresses a minute | `login.new_addresses_per_window`, 1 to 10 000, in a window of `login.budget_window`, 10 s to 1 h |
| Size of the table | 4096 addresses | `login.tracked_addresses`, 16 to 65 536 |
| Failures forgotten after | 1 h without failure | `login.forget_after`, from the longest delay to 7 days |
| Password checks at once per address | 1 | fixed; a second attempt during a check is refused with status 429 and `Retry-After: 2` |
| Delay per login | the same delays, counted per login whatever the address | the same settings; one password check at a time per login ([Per login](#per-login)) |

### Key

The delay applies to a key derived from the [client address](#client-address-and-trusted-proxies):

| Client address | Key |
| --- | --- |
| IPv4 | The address (`/32`) |
| Public IPv6 | Its `/64` prefix |
| Other IPv6 (private, link-local, loopback, or in a prefix of a local interface) | The address (`/128`) |
| None (a trusted proxy without a readable `X-Forwarded-For`) | One shared key, `::/128` |

The events `login failed` and `login throttled` log this key in their `address` field, not the full client address.

### Per login

- Besides its address, an attempt is delayed by the login it names, whatever the address it comes from.
- Only a login of valid form counts, whether the account exists or not, so that the delay reveals nothing about the accounts.
- The delay of a login follows the rules of [Delay](#delay): the first delay, doubled at each further failure up to the longest delay, the failures forgotten after `login.forget_after`; one password check at a time per login.
- The global budget does not apply to logins.
- A success does not erase the earlier failures of the login, as for an address: an account that exists, whose owner logs in, keeps the same delay as an unknown login.
- The logins are followed in a table of the same size as the addresses, `login.tracked_addresses`. When every entry of it is blocked or being checked, a new login is not followed, and only its address is delayed.
- The password form of the profile page counts the login of the viewer.
- A refusal gives the same status 429 and `Retry-After` as for an address; at the login form, the same `login throttled` event, which gives the key of the address, never the login.

### Delay

- After a failure, the key waits the first delay, 1 s by default.
- The delay doubles at each further failure, up to the longest delay, 5 min by default.
- A login attempted during the delay is refused with status 429 and a `Retry-After` header.
- A success does not erase the earlier failures of the key.
- The failures are forgotten after `login.forget_after` without failure, one hour by default.
- They are forgotten sooner when the table is full, never before the end of a block.

### Global budget

- At most `login.new_addresses_per_window` new keys per `login.budget_window` are admitted with the normal delay, 100 a minute by default.
- A key counts at its first attempt, failed or not; a success at the first attempt gives its place back.
- Beyond the budget, a new key still gets one attempt, but waits the longest delay (5 min by default) after a failure.
- The budget alone never refuses a login.
- Consequence: during an attack, a legitimate user who mistypes once waits up to the longest delay.

### Table

- The table holds `login.tracked_addresses` keys, 4096 by default.
- When it is full, the oldest entry that is neither blocked nor being checked is evicted, over-budget entries first.
- A new key is refused only while every entry is blocked or being checked. The answer is status 429 with a `Retry-After` of the budget window plus one second, `61` by default.
- This needs an attacker holding more addresses than the table; this residual lockout is accepted.

### Password change form

- The current password asked by the password form of the profile page (« Mon profil ») goes through the same throttle.
- A throttled password change is answered with status 429 and logs no event; the access log still records the request.

### Parallel checks

- The throttle allows one password check at a time per key.
- There is no global limit on parallel checks.
- An attacker holding many addresses can keep the processor busy with password hashing, about half a second per check.
- This is accepted for this deployment.

## CLIENT ADDRESS AND TRUSTED PROXIES

The client address is the address that the throttling, the access log and the events attribute to a request.

### Trusted proxies

- The trusted proxies are a list of IP addresses or prefixes.
- They are set by `server.trusted_proxies` in the server configuration file, or by `--trusted-proxy=10.0.0.1` on the command line, repeatable.
- The setting and its priority are described in [Server configuration](SPECIFICATIONS.md#server-configuration).

### How the client is chosen

1. If the source address of the connection is not a trusted proxy, it is the client address.
2. Otherwise the request is believed about its client: the client address is the last element of the last `X-Forwarded-For` header, the one that this proxy appended.
3. The earlier elements are ignored.
4. An element written `address:port` is accepted; the port is dropped.
5. If no element can be read, the request has no client address and shares one throttle key with all such requests.

### Default

- The default server configuration trusts the private ranges of RFC 1918: `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`.
- This has no effect while the server listens on the loopback only.
- The container image listens on every interface of the container and empties the list, since Docker forwards the connections to a published port from a private address ([Container](OPERATIONS.md#container)).

### Advice

- Once the server listens on a private network, trust the narrowest set that holds the proxies.
- Reason: every machine of a trusted range that can reach the server can choose the client address it announces.
- A trusted proxy is recognised by its source address only.
- With `--trusted-proxy=127.0.0.1`, any local user of the proxy host can reach the server directly and choose the `X-Forwarded-For` address.
- That gives a fresh throttle key per attempt, the blocking of a chosen address, and forged addresses in the logs.
- Keep the listening address out of reach of the other users of that host.

## SESSIONS

### Identifier and cookie

| Property | Value |
| --- | --- |
| Identifier | 32 random bytes (256 bits) |
| Storage | Only the SHA-256 of the identifier is kept |
| Renewal | A new identifier at each login; the previous session of the browser is closed |
| Cookie name | `ts_session`; `__Host-ts_session` when the origins are https |
| Cookie attributes | `Path=/`, `HttpOnly`, `SameSite=Strict` |
| `Secure` attribute | Set when the origins are https |
| Cookie lifetime | No `Max-Age`: the cookie ends with the browser session |
| CSRF token | One per session |

The prefix `__Host-` makes the browser refuse the cookie unless it is `Secure`, has `Path=/` and no `Domain`: another site of the same domain, or a plain HTTP page of the same host, cannot set or shadow it. The login cookie `ts_login` takes the same prefix. `HttpOnly` hides the cookie from scripts. `SameSite=Strict` keeps the browser from sending it with a request started by another site. `Secure` keeps it off plain HTTP.

### Expiry

| Kind | Default | Setting and bounds | Meaning |
| --- | --- | --- | --- |
| Sliding | 8 h | `sessions.idle_timeout`, 5 min to 7 days | A session unused for this long ends. |
| Absolute | 24 h | `sessions.max_lifetime`, from the sliding expiry to 30 days | A session ends this long after the login, used or not. |

Both expiries hold across restarts: the server saves the sessions when it stops, so that only a crash loses up to `sessions.save_interval` of last activity, which can end a session that much earlier. The configured values apply to the restored sessions too ([Section sessions](SPECIFICATIONS.md#section-sessions)).

### Session file

The sessions are kept in memory and saved in a file next to the database, so that a restart keeps the users connected.

| Property | Value |
| --- | --- |
| Name | Named after the database: `timesheet-database.json.sessions` next to `timesheet-database.json` |
| Mode | 0600 |
| Writing | Atomic, like the database |
| Size | At most 8 MiB |
| Content of each session | SHA-256 of the identifier, login, CSRF token, creation time, last activity |

- The file never holds the identifier itself, so it does not give the cookies.
- Reading it gives the CSRF tokens and who is connected, not the means to use a session.
- Protect it like the database.

The commands that act on the file are in [Command line](SPECIFICATIONS.md#command-line); its size limit is also in [Limits](SPECIFICATIONS.md#limits).

### When the file is written

- At each login and logout.
- At each closing of sessions.
- When an expired session is presented and dropped.
- By `user profile`, `user passwd` and `user del`, which remove the sessions of their account before changing it. A change refused afterwards (for example the last administrator) has already closed them.
- For the sliding expiry, at most once per `sessions.save_interval`, one minute by default.
- When the server stops.
- The file is written outside the lock of the sessions, so that a write does not hold up the other requests; a write never replaces the file with an older state of the sessions.

### At start

- `run` reloads the file.
- It drops the expired sessions.
- It drops the sessions dated in the future, which would otherwise never expire.
- A file that cannot be read is replaced by an empty one, with a warning. A file that gives any right to its group or to others counts as unreadable.

### Closing sessions

| Event | Sessions closed |
| --- | --- |
| Profile change of an account | All sessions of the account |
| Password reset or change | All sessions of the account (a change keeps the user connected with a new identifier) |
| Account disabled | All sessions of the account |
| Account deleted | All sessions of the account |
| « Déconnecter toutes mes sessions » on the profile page | All sessions of one's own account |
| Closing of the sessions of another account by an `ADMIN` in the administration | All sessions of that account |
| `session clear` (server stopped) | Every session |

### Checks on every request

- The account of the session is reloaded on every request.
- A deleted or disabled account loses its sessions.
- A session created before its account is refused. This happens when a session stays in the file after a deletion, a restored backup or a new database, and the login is then created again.
- A changed profile takes effect at once, and the sessions of the account are closed anyway.

### Limit per account

- At most `sessions.per_account` sessions per account, 64 by default.
- `limits.accounts` × `sessions.per_account` is at most 16 384, so that every session fits in the session file and can be restored.
- Beyond, the oldest session of the account is dropped.

## FORMS AND ORIGINS

### CSRF token

- Every form carries the CSRF token of the session.
- The login form, shown before any session, uses a double-submitted token: the same random value is sent in the cookie `ts_login` and in the form, and both must match.
- The cookie `ts_login` has a `Max-Age` of `login.form_lifetime` (3600 seconds by default, 1 min to 24 h) and is renewed each time the login form is shown. Its token is reused, so several tabs share it. Once the browser drops it, the login posted from the old form is refused (403).
- A missing or wrong token is refused with status 403.
- The import of the data page posts `multipart/form-data`: the guard parses it in memory (`maxMultipartMemory`, 8 MiB, at least the largest `server.max_body_bytes`) before reading the token, so that no file part is written to a temporary file; a body beyond `server.max_body_bytes` is refused with 400 before any processing.

### Origin check

An unsafe request is a request with a method other than `GET`, `HEAD`, `OPTIONS` and `TRACE`. The origin guard (`OriginGuard`) refuses every unsafe request with status 403 unless:

- its `Origin` header is one of the accepted origins (not empty, not `null`, same scheme);
- its `Sec-Fetch-Site` header is absent, `same-origin` or `none`.

### Host check

The host guard (`HostGuard`) refuses with status 421 a request whose `Host` is not one of the accepted origins.

| Case | Accepted origins |
| --- | --- |
| `--origin` given | The origins given with `--origin` |
| Neither `--origin` nor `--listen` | The origins of `server.origins` in the server configuration file |
| `--listen` given without `--origin` | Derived from the listener, `server.origins` being ignored |

Derived origins are, on the listening port:

- `localhost`, `127.0.0.1` and `::1`;
- the IPv4 addresses of the interfaces;
- the host name and `<host name>.local`.

They are derived even when the listener is on the loopback. `<host name>.local` and the bare host name can be answered by another host of the local network (see [Accepted risks](#accepted-risks)). Give `--origin` to accept only the names you choose, for example the loopback names.

## PAGES

| Defence | Rule |
| --- | --- |
| Templates | Pages are rendered with `html/template` only, which escapes by context. |
| Trusted content | No trusted-content type (`template.HTML` and the like) is used. |
| Colour sheet | `/dashboard/colors.css` is built from code identifiers and `#rrggbb` colours validated at load. |
| Inline code | No inline script or style. |
| Scripts | JavaScript uses the DOM API and `textContent` only. |
| User text | Comments are escaped by the templates. |
| Invisible characters | Labels, comments and reasons refuse control, format and bidirectional characters, the zero-width non-joiner and joiner (U+200C, U+200D) excepted. |

The rules for writing such code are in [Security rules for code](CODING.md#security-rules-for-code).

### Security headers

Every response that reaches the handlers carries these headers:

| Header | Value |
| --- | --- |
| `Content-Security-Policy` | See below |
| `X-Content-Type-Options` | `nosniff` |
| `X-Frame-Options` | `DENY` |
| `Referrer-Policy` | `same-origin` |
| `Cross-Origin-Resource-Policy` | `same-origin` |
| `Cross-Origin-Opener-Policy` | `same-origin` |
| `Permissions-Policy` | `camera=(), microphone=(), geolocation=(), payment=(), usb=(), interest-cohort=()` |
| `Cache-Control` | `no-store` |
| `Pragma` | `no-cache` |
| `Expires` | `0` |
| `Strict-Transport-Security` | `max-age` of `server.hsts_max_age` (`31536000`, one year, by default; 24 h to 2 years), only with https origins, without `includeSubDomains` |

The content security policy (CSP) tells the browser which sources a page may use:

```text
default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self'; connect-src 'self'; font-src 'self'; form-action 'self'; base-uri 'none'; frame-ancestors 'none'; require-trusted-types-for 'script'; trusted-types 'none'
```

- `frame-ancestors 'none'` and `X-Frame-Options: DENY` forbid framing.
- `require-trusted-types-for 'script'` with `trusted-types 'none'` (Trusted Types) makes the browser refuse a string in a script sink such as `innerHTML`.
- HSTS (HTTP Strict Transport Security) tells the browser to use only https for this host for `server.hsts_max_age`, one year by default.
- The other headers and the CSP are fixed.

Limit: requests refused by the HTTP parser of Go itself (headers over the limit, status 431; malformed request, status 400) are answered in plain text, without these headers.

## AVAILABILITY

- The size limits (URL, body, headers, data of one account, adjustments, planning horizon, import file) are listed in [Limits](SPECIFICATIONS.md#limits).
- Sessions, accounts, codes and counters are bounded.
- The data of one account is limited to `limits.account_bytes` and `limits.adjustments`, 1 MiB and 1000 by default, and the accounts to `limits.accounts`, 200 by default.
- The request sizes and the timeouts are set in the [section server](SPECIFICATIONS.md#section-server) of the server configuration, within bounds: the request body never goes below 256 KiB, the size of the largest legitimate form.
- Months can be entered up to 24 months ahead.
- The import file is bounded to 8 MiB and decoded strictly.

### Timeouts

| Timeout | Default | Setting and bounds |
| --- | --- | --- |
| Reading the request headers | 5 s | `server.read_header_timeout`, 1 s to 10 min |
| Reading the whole request | 15 s | `server.read_timeout`, 1 s to 10 min |
| Writing the answer | 30 s | `server.write_timeout`, 1 s to 10 min |
| Idle connection | 1 min | `server.keep_alive_timeout`, 1 s to 10 min |
| Graceful shutdown | 10 s | `server.shutdown_timeout`, 1 s to 60 s |

### Write load

Writes are not rate-limited, and each rewrites the whole file (see [Accepted risks](#accepted-risks)).

## DATA INTEGRITY

- Quantities are exact rationals, with bounded denominators and magnitudes.
- Overflows are detected.
- Every write is checked by a round trip: the encoded data is decoded back strictly before it is written, and a failure refuses the write.
- One exclusive lock is shared by the server and the commands that write.
- Concurrent edits from several browsers are detected by optimistic revisions: a save made on a stale revision is refused.
- The database refuses any write that rewrites a validated month, even when a controller or a command forgets to check it; only the deletion of the data erases the history ([Invariants](SPECIFICATIONS.md#invariants)).

## FILES

| Check | Applies to | Result |
| --- | --- | --- |
| Regular file, not a symbolic link | Database, session file | Refused |
| Owner is the user running the program | Database, session file, their directory | Refused |
| No right for the group or others (0600 expected) | Database, session file | Refused (the session file is replaced by an empty one at start) |
| Directory not writable by others, unless sticky | Directory of the database | Refused |
| Directory writable by its group | Directory of the database | Warning |

A database that fails a check is refused. A session file that fails any check is replaced by an empty one at start, with a warning.

- `run` and every command that writes check the file and its directory.
- `user list` takes no lock and checks only the file itself.
- Writes are atomic: a temporary file is written, then renamed.
- When the directory is group-writable, the members of the group can still delete or replace the files.
- In a sticky directory shared with other users, another user can block the program by creating its lock or session file first.

## TRANSPORT

### Plain HTTP

- Plain HTTP is meant for `127.0.0.1` only, on a host without other users or other local web services.
- Reason: cookies are shared by every port of a host name.
- Another service listening on the loopback therefore receives the session cookie of a browser that visits it, and can replace it.
- On a shared host, use TLS even on the loopback, so that the cookie is marked `Secure`.

### Warning at start

| Origins | Certificate pair | Behaviour |
| --- | --- | --- |
| http | None | Serves plain HTTP and logs a warning |
| https | None | Assumes a TLS reverse proxy; no warning |
| https | Given | Serves TLS |
| http | Given | Refused at start |

### Built-in TLS

- TLS (Transport Layer Security) 1.2 at least, or 1.3 at least with `server.tls_min_version` set to `1.3`.
- Six cipher suites for TLS 1.2, all with ECDHE key exchange and authenticated encryption (AES-GCM or ChaCha20-Poly1305); they are fixed. TLS 1.3 uses its own suites.
- HTTP/1.1 only.

### On a network

| Option | Settings | Effect |
| --- | --- | --- |
| Certificate pair | `server.tls_cert` and `server.tls_key` (or `--tls-cert`, `--tls-key`), https origins | The server serves TLS itself. |
| TLS reverse proxy | `--origin=https://<public name>` (or `server.origins`), `--trusted-proxy=<proxy address>` (or `server.trusted_proxies`) | The proxy serves TLS and forwards to the server. |

Both options give:

- a session cookie marked `Secure`;
- the HSTS header;
- the real client address for the throttling.

### Reverse proxy on another machine

- The link from the proxy to the server is plain HTTP, unless the server also has a certificate pair.
- On that link, passwords and cookies travel in clear, and the server does not warn.
- Any machine that reaches the listening port can talk to the server without the proxy, by sending the public `Host`.
- Restrict the listening port to the proxy with a firewall, or give the server a certificate pair too.
- The proxy must pass the public host unchanged and append the client address to `X-Forwarded-For`.

### Certificate renewal

- The certificate pair is read once, when `run` starts.
- After renewing it, restart the server.
- Sessions survive the restart.

## LOGS

### Never logged

- A password.
- A login typed at a failed login.
- A session identifier.
- A CSRF token.
- A query string: only `r.URL.Path` may be logged, and the access log writes the path without its query string.
- A request body.
- The `User-Agent` beyond its first 200 bytes.
- The content of the days: codes, comments, reasons.

Limits of this guarantee:

- `secrets_test.go` checks the log output after a failed and a successful login; the other paths are checked by review.
- Some events write the text of an error (for example `account creation refused`); that text is checked by review only.

### What the logs tell

- They tell who opened which page and when, the reads of other accounts by an `ADMIN` included.
- They hold client addresses and logins.
- Protect them like the database.
- Their retention is the one of the service manager or of the file that collects them.
- Requests refused by the HTTP parser of Go (status 431 or 400) leave no log line.
- Commands of the command line log no request and no event; they print their warnings on the standard error.

The fields, the events and their levels are in [Logs](SPECIFICATIONS.md#logs).

## DATA PROTECTION

### Sensitive data

- Health or family events can be read from day codes and comments.
- Who reads what, retention and deletion: see [Personal data](SPECIFICATIONS.md#personal-data).

### Backups

- A backup is a copy of the single database file.
- Protect it like the database: it contains the password hashes and every user's absences and comments.
- The database and its backups are not encrypted.
- Restoring a backup brings back the passwords, profiles, states and accounts of its date.
- A password changed, or an account disabled or deleted, since that date must be changed, disabled or deleted again.

### Import files

- The spreadsheet that the yearly summary is modelled on, and any import file made from real data, are never committed to the repository.
- Tests build synthetic import files.
- An uploaded import file is decoded strictly (unknown fields refused, 8 MiB at most) and never stored as a file: only its days, settings and adjustments, once checked, enter the database.
- The file downloaded from the data page « Données » holds the comments and adjustment reasons of the account: it is sent with `Cache-Control: no-store` and is the responsibility of the person once downloaded.

## REPORTING A VULNERABILITY

- Report a vulnerability privately to the maintainer named in the `Maintainer` field of `debian/control`.
- Do not open a public issue for it.
- No list of supported versions is published.
