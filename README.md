# TIMESHEET

## ABSTRACT

Timesheet is a web application with which employees working in France keep their own record of their activity and of their leave counters, to check what their employer reports and their payslips against it.

![Timesheet](doc/timesheet.png)

## CONTENTS

- [ABSTRACT](#abstract)
- [CONTENTS](#contents)
- [DESCRIPTION](#description)
- [REQUIREMENTS](#requirements)
- [GETTING STARTED](#getting-started)
- [BUILD](#build)
- [RUN](#run)
- [LICENCE](#licence)
- [CREDITS](#credits)

## DESCRIPTION

### Overview

Timesheet is a personal activity and leave tracker for employees working in France. Each day of the year is split into a morning and an afternoon, and each half-day receives an activity code: working time, part time, paid leave (congés payés), RTT, public holiday, weekend, unpaid leave or absence, with a free comment for the day. From these entries, the application keeps the leave counters of French labour practice up to date.

It answers a common problem. Many employers track the activity and the leave of their employees with tools that work poorly: balances shown late or wrong, entries that are not up to date, rules applied unevenly, counters on the payslip that are hard to reconcile with the days actually taken. The employee usually has no other reference than these figures, so an error can go unnoticed for months, until the facts are hard to establish and the correction hard to obtain.

Timesheet gives the employee that reference. They keep, on their own and independently of the tools of their employer, their activity day by day. The application keeps each counter like a bank account: every monthly accrual of paid leave or RTT, every day of leave taken, every rounding or reset at the end of a period and every correction is a dated entry with its label, and the balance is the sum of the entries. The French rules are applied automatically: the public holidays of the chosen zone, paid leave accrued at 2.08 days a month over a June-to-May period and rounded up at its end, RTT days, part-time days and the complementary days of a part-time contract.

Each month that is over is validated like a bank statement: its entries are frozen, and a later correction never erases them but reopens the month or adds a dated adjustment with its reason. The history stays complete and traceable. The dashboard (« Tableau de bord ») gives the balance of each counter today and a forecast at the end of its period, the month page (« Mois ») the statement of each counter, and the year page (« Année ») a yearly summary month by month, modelled on a spreadsheet.

The employee then compares their own records with what their employer reports and with each payslip. A gap is spotted early, while the facts are still easy to establish, and can be raised with the employer, dates and figures in hand. A correction the employee accepts, an adjustment of the payslip for instance, is entered with its reason, so that the two records agree again.

An existing history, kept in a spreadsheet for instance, can be converted to a JSON file and imported, and the data of an account can be exported in the same format at any time. Timesheet does not replace the tool of the employer nor its payroll: it is the employee's own reference to check them against. Although it supports several accounts, it is meant above all for personal use, on a workstation or a small server, from the sources, as a Debian or RPM package or in a container. The reasons and the scope are detailed in [Purpose](doc/SPECIFICATIONS.md#purpose).

### Features

- Designed for the French labour market:
  - the public holidays of metropolitan France, of Alsace-Moselle and of the overseas territories, with a default zone for the database and an optional zone per user;
  - paid leave (congés payés), accrued at 2.08 days a month over a June-to-May period and rounded up at its end;
  - RTT days, the weekly day off and the complementary days of a part-time contract;
  - a web interface in French.
- The activity codes and the counters are configurable. The calendar of public holidays and the rules are French: the application is not meant, as it stands, for another country.
- The rules are choices of the application, simpler than common French practice: see [Simplifications](doc/SPECIFICATIONS.md#simplifications).
- One static binary, no external service: it runs on a workstation or a small server.
- An optional AI assistant answers questions on your own data, in French, with Markdown tables, through your own API key of Mistral, Anthropic, OpenAI or Google, which you can share with other accounts; the writes it proposes are applied only once you confirm them, and it can give your export as a download link valid for 30 minutes (see [Use the assistant](doc/USER-GUIDE.md#use-the-assistant)).
- Each user imports a file, exports their data in the same JSON format and erases them, from « Mes données ».
- Although it supports several accounts, it is essentially meant for personal use.
- The database is, for now, a single JSON file; SQLite and other databases are planned for the future.
- The web interface is in French. The French labels quoted in this document (« Tableau de bord », « Activité »…) are the ones shown on screen. The command line, code and documentation are in English.
- Free software, released under the GNU General Public License version 2 or later.

### Where to start

| You want to | Read |
| --- | --- |
| Understand what the application does | [Concepts](doc/USER-GUIDE.md#concepts), [Pages](doc/USER-GUIDE.md#pages) and [Accounts](doc/USER-GUIDE.md#accounts) |
| Try it on your workstation | [Requirements](#requirements), then [Getting started](#getting-started) |
| Try it with example data | [Example database](doc/OPERATIONS.md#example-database) |
| Use it day after day | [User guide](doc/USER-GUIDE.md) |
| See the pages | [Screenshots](doc/USER-GUIDE.md#screenshots) |
| Bring an existing history in | [Importing](doc/OPERATIONS.md#importing) |
| Build it, run the tests, build the package | [doc/BUILD.md](doc/BUILD.md) |
| Install it as a service | [Debian package](doc/OPERATIONS.md#debian-package), [RPM package](doc/OPERATIONS.md#rpm-package) |
| Run it in a container | [Container](doc/OPERATIONS.md#container) |
| Change the address, the host names, the time zone, the sessions or the limits | [Server configuration](doc/OPERATIONS.md#server-configuration) |
| Put it behind a reverse proxy | [Network and security](doc/OPERATIONS.md#network-and-security) |
| Upgrade an installation | [Upgrade](doc/OPERATIONS.md#upgrade) |
| Fix a refusal or an error message | [Troubleshooting](doc/OPERATIONS.md#troubleshooting) |
| Change the code | [doc/BUILD.md](doc/BUILD.md) and [Documentation](#documentation) |

### Repository layout

- One static Go binary, standard library only.
- Hand-written HTML, CSS and JavaScript. The pages work without JavaScript.
- The activity codes, the counters, the accounts and the data live in one JSON file, the database, rewritten atomically at each change.
- The server configuration (address, origins, database path, timeouts, sessions, login throttle, limits) is a second JSON file.
- The open sessions are saved in a third file, next to the database, and the key that seals the API keys of the AI subscriptions in a fourth one.

```text
.ci/    scripts run by the Makefile targets and by the continuous integration
bin/    timesheet (built), timesheet.sh (runs the server on the configuration and the database of the repository)
debian/ Debian packaging (package, systemd unit, post-installation script)
rpm/    RPM packaging (the spec of the package, built for Fedora)
doc/    ARCHITECTURE.md  BUILD.md  CODING.md  OPERATIONS.md  SECURITY.md  SPECIFICATIONS.md  USER-GUIDE.md  timesheet.png  timesheet-*.png (screenshots)
etc/    timesheet/timesheet.json (the server configuration, installed as is by the packages)
share/  timesheet/ (configuration template read by init, example database, import template)
src/    Go sources
var/    lib/timesheet/ (database of the repository, given with --database, its lock, its session file and its key file)
```

### Documentation

| Document | Content |
| --- | --- |
| [doc/USER-GUIDE.md](doc/USER-GUIDE.md) | Concepts, activity codes, pages and profiles; the tasks of the employee and of the administrator; the labels shown on screen. |
| [doc/BUILD.md](doc/BUILD.md) | Requirements, `make` targets, installation of the files, Debian and RPM packages, container image. |
| [doc/OPERATIONS.md](doc/OPERATIONS.md) | Commands, server configuration, network, logs, import, example database, Debian and RPM packages, container, backup, recovery, upgrade, troubleshooting. |
| [doc/SPECIFICATIONS.md](doc/SPECIFICATIONS.md) | The functional reference: glossary, profiles, entries, counters and bookkeeping, pages, AI assistant, routes, database schema, server configuration, command line, import format, logs, limits, personal data, simplifications, migration notes. |
| [doc/ARCHITECTURE.md](doc/ARCHITECTURE.md) | Packages, entities, HTTP chain, concurrency, persistence, lifecycle. Start with [Packages](doc/ARCHITECTURE.md#packages). |
| [doc/SECURITY.md](doc/SECURITY.md) | Threat model, sessions, passwords, login throttling, transport, AI assistant, data protection, accepted risks. |
| [doc/CODING.md](doc/CODING.md) | The rules of the code: dependencies, file layout, templates, styles, tests, Makefile, packaging, commits. Before a commit, see [Checks before a commit](doc/CODING.md#checks-before-a-commit). |

## REQUIREMENTS

| To | You need |
| --- | --- |
| Run | A Unix system (Linux, BSD, macOS) for the database file lock, and its time zone database (`tzdata`). |
| Build | Go 1.24 or later, and `make`; `gcc` to run the tests. The tools of the packages are listed in [doc/BUILD.md](doc/BUILD.md#requirements). |

The password prompt without echo needs Linux. Elsewhere, `user add` and `user passwd` read one visible line of the standard input, with a warning when it is a terminal or another character device: prefer the variable `TIMESHEET_PASSWORD` or a pipe.

## GETTING STARTED

This section goes from the sources to a running server on your workstation. To install a package instead, see [Debian package](doc/OPERATIONS.md#debian-package) or [RPM package](doc/OPERATIONS.md#rpm-package); to run a container, see [Container](doc/OPERATIONS.md#container).

- Run every command from the root of the repository.
- `<login>` is the login you choose: 2 to 32 characters among lower-case letters, digits, dots, dashes and underscores, starting with a letter or a digit.

### First start

1. Get the sources, by a clone of the repository or an archive, and enter their directory.

2. Build the binary:

   ```sh
   make
   ```

   The first build takes a few seconds more, the time to fill the Go cache `.cache/`. It ends with `=== build ok ===` and produces `bin/timesheet`.

3. Protect the directory of the database, shipped as `var/lib/timesheet/`:

   ```sh
   chmod 700 var/lib/timesheet
   ```

   - The directory must belong to the user who runs the commands and the server.
   - Without the `chmod`, a directory writable by its group gives a warning (`directory is writable by its group`).
   - A directory writable by others, without the sticky bit, is refused.

4. Create the database:

   ```sh
   bin/timesheet init --template=share/timesheet/database-template.json --database=var/lib/timesheet/timesheet-database.json --config=etc/timesheet/timesheet.json
   ```

   It answers:

   ```text
   Database created: …/var/lib/timesheet/timesheet-database.json
   Now create an administrator: timesheet user add <login> --profile=ADMIN --database=…/var/lib/timesheet/timesheet-database.json --config=…/etc/timesheet/timesheet.json
   ```

   - The command it prints is the next step; from the repository root, prefix it with `bin/`. The database starts with the activity codes and counters of the configuration template and no account.
   - No path has a default: `--config` names the server configuration, and `--template` the configuration template of `init`.
   - `--database` is needed from the sources: the server configuration of the repository names the database of the packages, `/var/lib/timesheet/timesheet-database.json` ([Server configuration](doc/OPERATIONS.md#server-configuration)).

5. Create your account, as an administrator:

   ```sh
   bin/timesheet user add <login> --profile=ADMIN --database=var/lib/timesheet/timesheet-database.json --config=etc/timesheet/timesheet.json
   ```

   - On a Linux terminal, the password is asked twice, without echo.
   - It has at least 12 characters, 6 of them distinct, no space at its start or end, no control character, 256 bytes at most, and differs from the login.
   - To script it, give the password in the variable `TIMESHEET_PASSWORD`.
   - It answers `Account <login> created (ADMIN).`.

6. Start the server:

   ```sh
   bin/timesheet.sh
   ```

   The script runs `bin/timesheet run` with the server configuration and the database of the repository, from any directory.

   - It stays in the foreground and logs on the standard error: one line per request and one per event, such as a login or a saved month. See [Logs](doc/OPERATIONS.md#logs).
   - The warning `serving plain HTTP` is expected on a workstation. Read [Network and security](doc/OPERATIONS.md#network-and-security) before exposing the server on a network.
   - Ctrl-C or SIGTERM stops it after the requests in progress (`server.shutdown_timeout`, 10 s by default).

7. Open `http://127.0.0.1:8080` or `http://localhost:8080`, click the arrow at the right of the header (« Se connecter ») and log in. You arrive on « Tableau de bord ».

Then follow the [User guide](doc/USER-GUIDE.md).

### Next starts

- Run `bin/timesheet.sh` from any directory, or `bin/timesheet run --database=var/lib/timesheet/timesheet-database.json --config=etc/timesheet/timesheet.json` from the repository root.
- The sessions survive a restart.

## BUILD

`make` builds `bin/timesheet` and `make check` runs the tests. The other targets, the requirements, the Debian and RPM packages and the container image are described in [doc/BUILD.md](doc/BUILD.md).

## RUN

- From the repository: `bin/timesheet.sh` from any directory, or `bin/timesheet run --database=var/lib/timesheet/timesheet-database.json --config=etc/timesheet/timesheet.json` from the root (see [Getting started](#getting-started)).
- As a service: build the Debian or RPM package, install it, then create the first administrator (see [Debian package](doc/OPERATIONS.md#debian-package) or [RPM package](doc/OPERATIONS.md#rpm-package)).
- In a container: build the image with `make docker`, then run it with a volume for the database; its first start creates the database and an administrator, `admin` or `TIMESHEET_USERNAME`, whose password, `Tms@Admin%26` or `TIMESHEET_PASSWORD`, must be changed at the first login (see [Container](doc/OPERATIONS.md#container)).

Everything else about running the server is in [doc/OPERATIONS.md](doc/OPERATIONS.md): server configuration, network and reverse proxy, logs, import, backup and restore, recovery, upgrade and troubleshooting.

## LICENCE

Timesheet is free software, released under the GNU General Public License version 2 or later. See [COPYING](COPYING).

```text
timesheet - Copyright (c) 2023-2026 - Olivier Poncet

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 2 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
```

## CREDITS

Olivier Poncet, see [AUTHORS](AUTHORS).
