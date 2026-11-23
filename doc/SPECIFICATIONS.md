# SPECIFICATIONS

This document is the single functional reference of Timesheet: what the application does, what each page shows and accepts, what each route and command answers, and what the database holds.

It is written for the maintainer, for an AI agent changing the code and for a new developer. The sections on routes, the command line, the server configuration, the database, the logs and the limits also serve the operator who runs the service and the administrator who manages the accounts.

Every change of behaviour, screen, label, route, form field, computation rule, database schema or command updates this document in the same change. The rules for writing the documentation are in [CODING.md](CODING.md#documentation).

| Document | Content |
| --- | --- |
| [README.md](../README.md) | Entry point, first start from the sources |
| [BUILD.md](BUILD.md) | Requirements, `make` targets, Debian and RPM packages, container image |
| [OPERATIONS.md](OPERATIONS.md) | Commands, server configuration, network, logs, import, package, container, backup, recovery and upgrade procedures, troubleshooting |
| [USER-GUIDE.md](USER-GUIDE.md) | Concepts, daily tasks of the employee and of the administrator, labels shown on screen |
| [ARCHITECTURE.md](ARCHITECTURE.md) | Packages and import direction, entities, HTTP chain, flows, concurrency, persistence, lifecycle |
| [CODING.md](CODING.md) | How the code, the templates, the styles and the tests are written |
| [SECURITY.md](SECURITY.md) | Threat model, accepted risks, sessions, passwords, login throttling, transport, data protection |

## CONTENTS

- [CONVENTIONS](#conventions)
- [PURPOSE](#purpose)
- [GLOSSARY](#glossary)
- [PROFILES AND ACCOUNTS](#profiles-and-accounts)
- [CALENDAR AND DAYS](#calendar-and-days)
- [ACTIVITY CODES](#activity-codes)
- [COUNTERS](#counters)
- [USER SETTINGS](#user-settings)
- [ADJUSTMENTS](#adjustments)
- [COUNTER BOOKKEEPING](#counter-bookkeeping)
- [MONTHLY INDICATORS](#monthly-indicators)
- [PAGE STRUCTURE](#page-structure)
- [PUBLIC PAGES](#public-pages)
- [DASHBOARD](#dashboard)
- [ACTIVITY PAGE](#activity-page)
- [MONTH PAGE](#month-page)
- [YEAR PAGE](#year-page)
- [PROFILE PAGE](#profile-page)
- [SETTINGS PAGE](#settings-page)
- [DATA PAGE](#data-page)
- [AI ASSISTANT](#ai-assistant)
- [ADMINISTRATION](#administration)
- [COLOURS](#colours)
- [ROUTES](#routes)
- [NOTICES](#notices)
- [FORM FIELDS](#form-fields)
- [COMMAND LINE](#command-line)
- [IMPORT](#import)
- [DATABASE](#database)
- [SESSION FILE](#session-file)
- [SERVER CONFIGURATION](#server-configuration)
- [CONFIGURATION TEMPLATE](#configuration-template)
- [EXAMPLE DATABASE](#example-database)
- [MIGRATION NOTES](#migration-notes)
- [LOGS](#logs)
- [SECURITY REQUIREMENTS](#security-requirements)
- [LIMITS](#limits)
- [NON-FUNCTIONAL REQUIREMENTS](#non-functional-requirements)
- [OUT OF SCOPE](#out-of-scope)
- [SIMPLIFICATIONS](#simplifications)
- [PERSONAL DATA](#personal-data)

## CONVENTIONS

- The web pages are in French. The command line (help, prompts, messages, import report), identifiers, code, logs and documentation are in English; the command line keeps in French only the texts it shares with the pages (labels of codes and counters, refusals of the grid).
- Interface texts are quoted between « » and are the exact French texts shown on screen.
- Texts printed by the command line are quoted in backticks and are the exact English texts printed.
- A part of an interface text that varies is written `<name>` inside the quotes, for instance « Soldes au \<date du jour> ».
- Identifiers, values, paths, fields, routes and commands are written in backticks.
- An HTTP status is written as its number after the case it answers, for instance "refused (409)".
- In a command synopsis, a placeholder is written in upper case (`LOGIN`, `FILE.json`) or as `...` after an option.
- Dates shown by the interface follow [Dates](#dates). "Today" is defined in [Today and time zone](#today-and-time-zone).
- The words of this document are defined in the [Glossary](#glossary); each section uses only these words.

## PURPOSE

### Why this application

- Some employers track the activity and the leave counters of their employees with tools that work poorly: balances out of date, wrong or hard to read, late or missing entries.
- Timesheet lets an employee keep, on their own and independently of these tools, their activity day by day and their counters.
- The employee then compares their own records with what the employer reports and with the payslips, month after month.
- A gap found this way is spotted early, while the facts are still easy to establish, and can be raised with the employer.
- A gap the employee accepts, a correction of the payslip for instance, is entered as an adjustment with its reason ([Adjustments](#adjustments)); validated months freeze the records as they stood ([Validation](#validation)).
- Timesheet does not replace the tool of the employer nor its payroll: it is the employee's own reference to check them against.

### Scope

- Timesheet is designed for employees working in France.
- It follows the public holidays of metropolitan France, of Alsace-Moselle and of the overseas territories, one zone per account ([Public holidays](#public-holidays)).
- It keeps paid leave (congés payés) with a June-to-May acquisition period and a rounding at the end of the period.
- It keeps RTT days, part-time days and the extra days some employers grant to part-time employees.
- The web interface is in French; the command line is in English.
- Activity codes and counters are configurable ([Activity codes](#activity-codes), [Counters](#counters)).
- The calendar of public holidays and the rules are French: another country is out of scope ([Out of scope](#out-of-scope)).
- The rules are the choices of the application; their differences with common French practice are listed in [Simplifications](#simplifications).

### What the application does

- It tracks a professional activity half-day by half-day, for every day of the year.
- It keeps leave counters up to date like bank accounts.
- A counter receives monthly or yearly accruals.
- A counter is debited by the leave taken, day after day.
- At the end of each period, a counter carries its balance over or is zeroed.
- An employee corrects a balance with an adjustment.
- The monthly validation freezes the entries of a month.
- A yearly summary gives, month by month, the movements and balance of each counter.

### Worked example

This is a fictitious example of an employee, used throughout the document.

| Parameter | Value |
| --- | --- |
| Working time | part time at 80 % |
| Weekly day off | every Wednesday, entered by hand with the code `TP` |
| Extra days off granted by the employer to this part-time employee | 7 days a year, set by the employee as the own amount `7/12` of the counter `TPC` |
| `TPC` accrual | `7/12` day credited on the last day of each month, over the calendar year |
| `TPC` consumption | debited as the days are taken |
| `TPC` at the end of the year | not carried over: the balance is zeroed on 1 January |
| Paid leave (`CP`) accrual | 2.08 days credited on the last day of each month |
| `CP` balance | one single balance |
| `CP` period | 1 June to 31 May; the balance is carried over to the next period |
| `CP` rounding | on 31 May, a positive fractional balance is rounded up to the next whole day |

The number of extra days comes from the employment contract: the application does not compute it ([Out of scope](#out-of-scope)).

## GLOSSARY

| Term | Meaning |
| --- | --- |
| `TT` | Code and counter « Temps de travail »: a worked half-day. |
| `WE` | Code « Week-end » and counter « Week-ends »: a weekend half-day. |
| `JF` | Code « Jour férié » and counter « Jours fériés »: a half-day off entered like a public holiday, such as a public holiday or a bridge day. A "day entered `JF`" is not always a [public holiday](#public-holidays). |
| `CP` | Code « Congé payé » and counter « Congés payés »: paid leave. |
| `CSS` | Code « Congé sans solde » and counter « Congés sans solde »: unpaid leave. |
| `RTT` | Code « Récupération temps de travail » and counter « Récupérations temps de travail »: days off granted for working beyond 35 hours a week. |
| `TP` | Code « Temps partiel » and counter « Temps partiels »: the weekly day off of a part-time employee, entered by hand. |
| `TPC` | Code « Temps partiel complémentaire » and counter « Temps partiels complémentaires »: extra days off some employers grant to part-time employees, managed like `RTT`. The name is the template's, not a legal notion. |
| `ABS` | Code « Absence » and counter « Absences »: an absence, its reason going into the comment. |
| AA | The WCAG 2 contrast level of 4.5:1 for normal text. |
| AI provider | A company whose language models the [AI assistant](#ai-assistant) can call: one row of `configuration.ai_providers`, with its API format and its address. |
| account | The record of one person: login, name, profile, state, password hash and data. |
| accrual | A credit given by the rule of a counter, every month or every year. |
| adjustment | A signed quantity entered by an employee on a counter at a date, with a reason ([Adjustments](#adjustments)). Also the kind `adjustment` of ledger entries made by the rules. |
| administrator | A person whose account has the profile `ADMIN`. |
| alert | A paragraph `<p role="alert">`, in red at the top of the page, telling why a request was refused ([Main content](#main-content)). |
| anomaly | A problem of a stored day entry, listed above the grid ([Validation](#validation)). |
| balance | The value of a counter: the sum of its entries. |
| cancelled entry | A ledger entry that a reversal cancels. |
| carry over | Keeping the balance of a counter into the next period. |
| confirmation | The choice of the user, « Confirmer » or « Annuler », that applies or drops a pending action of the assistant. |
| configuration | The section of the database holding the default zone of public holidays, the activity codes and the counters. |
| configuration page | The page `/admin/settings` that edits the configuration. |
| configuration template | The JSON file read by `init` to create the configuration ([Configuration template](#configuration-template)). |
| connected | The access level of the routes reserved to a valid session of a `USER` or an `ADMIN`. |
| consuming code | An activity code that debits its counter. |
| conversation | The questions of a user to the assistant and its answers, kept on the account ([Conversations](#conversations)). |
| counter | An account of days, kept like a bank account ([Counters](#counters)). |
| day entry | What is entered for one day: a code for the morning, a code for the afternoon, a comment. |
| default amount | The accrual amount of a counter in the configuration. |
| effective amount | The own amount of an employee when set, otherwise the default amount. |
| example database | The anonymized database `share/timesheet/database-example.json`, to try the application ([Example database](#example-database)). |
| feeding code | An activity code that credits its counter. |
| forecast | The future entries computed up to the end of the period of each counter: accruals, planned leave, roundings and zeroings. |
| frozen entry | A ledger entry written at the validation of a month. |
| guest | A request without a valid session (profile `GUEST`). |
| HTML template | A page template under `src/internal/web/templates/`. |
| import template | The fictitious import file `share/timesheet/import-template.json`. |
| indicators | The day counts of a month ([Monthly indicators](#monthly-indicators)). |
| journal | The sequence of the entries of a counter: the frozen ones plus the provisional ones computed for the open months. |
| ledger | The stored list of frozen entries and reversals, `ledger` in the database. |
| login | The identifier of an account; also the act of logging in. |
| missing half-day | An empty half-day of a tracked day before today, shown « Non renseigné ». |
| notice | A paragraph `<p role="status">`, in green at the top of the page, confirming a successful action ([Main content](#main-content)). |
| notice code | The code of the notice kept in the session by a successful action, for the next page ([Notices](#notices)). |
| open month | A month that is not validated. |
| operator | A person with shell access to the host that runs the service. |
| options | The `--name=value` arguments of the command line. |
| over | A month is over when today is after its last day. |
| pending action | A write proposed by the assistant, shown to the user and applied only after their confirmation ([Actions to confirm](#actions-to-confirm)). |
| own amount | The accrual amount an employee sets for a counter in the settings page. |
| period | The year of a counter, from its period start to the day before the next one. |
| period start | The first day of the period of a counter, `MM-DD`. |
| planning horizon | The last month that can be entered: 24 months after the current one. |
| position | The three balances of a counter on the dashboard: today, validated, forecast. |
| problem | A reason why a submitted value is refused. |
| public holiday | A day of the calendar of the zone of public holidays of an account, such as « Noël » ([Public holidays](#public-holidays)). |
| provisional entry | An entry computed for an open month on each display, never stored. |
| recap | The monthly table of one counter on the year page. |
| reversal | An entry of opposite amount that cancels an earlier entry, labelled « Contre-passation : \<motif> ». |
| revision | A number raised by each write, used to refuse a form shown before another write. |
| rounding | The adjustment that rounds a carried balance up to the next whole day. |
| server page | The page `/admin/` that shows the name, version and licence of the software, then the server configuration read-only ([Server page](#server-page)). |
| server configuration | The file given by `--config`, `etc/timesheet/timesheet.json` in the repository (`/etc/timesheet/timesheet.json` in the packages), holding the settings of the server and of its operation: address, origins, database path, timeouts, sessions, login throttle, password hashing and capacity ([Server configuration](#server-configuration)). |
| skill | The Markdown text embedded in the program that tells the assistant what the application is and how to answer ([Agent and skill](#agent-and-skill)). |
| statement | The table of the entries of one counter for one month, with opening and closing balances. |
| subscription | An API key of an AI provider with a model, owned by one account and possibly shared with others ([Subscriptions](#subscriptions)). |
| summary | The yearly table of the movements of every counter ([Yearly summary](#yearly-summary)). |
| tool | A function of the application that the assistant may call to read the data of the user or to propose a write ([Tools](#tools)). |
| tile | One counter on the dashboard, an `<article>`. |
| totals | The days of each activity code over a year or a month. |
| tracking start | The first tracked day of an account, « Date de début » in the settings page. |
| unsafe request | A request whose method is not `GET`, `HEAD`, `OPTIONS` or `TRACE`. |
| untracked half-day | A half-day before the tracking start, shown « Hors suivi ». |
| upcoming half-day | An empty half-day from today on, shown « À venir ». |
| user | The person using the application; `USER` is the profile of a person who is not an administrator. |
| user settings | The settings of an account: tracking start, zone of public holidays, counters, colours ([User settings](#user-settings)). |
| validated month | A month whose entries are frozen ([Validation](#validation)). |
| warning | A paragraph `<p role="status" class="is-warning">`, in yellow at the top of the page, pointing out what to check or to do ([Main content](#main-content)). |
| worked day | A day entered `TT`. |
| working day | A day from Monday to Friday that is not a public holiday, whatever is entered. |
| zeroing | The adjustment that brings the balance of a counter to zero at its period start. |
| zone of public holidays | One of the thirteen calendars of [Public holidays](#public-holidays): the zone chosen in the settings of an account, otherwise the default zone of the configuration. |

Technical words:

- CSRF: cross-site request forgery.
- CSP: Content Security Policy.
- HSTS: HTTP Strict Transport Security.
- TLS: Transport Layer Security.
- PBKDF2: a password-based key derivation function.
- IANA time zone: a name of the IANA time zone database, such as `Europe/Paris`.
- RFC 1918: the private IPv4 ranges `10.0.0.0/8`, `172.16.0.0/12` and `192.168.0.0/16`.
- WCAG: Web Content Accessibility Guidelines.

## PROFILES AND ACCOUNTS

### Profiles

| Profile | Stored | Rights |
| --- | --- | --- |
| `GUEST` | never: implicit for any request without a valid session | `/`, `/login`, `/static/...`; every `/dashboard/...` and `/admin/...` route and `/logout` redirect (303) to `/login` |
| `USER` | yes | own activity, counters, user settings, profile and password; `/logout` |
| `ADMIN` | yes | everything a `USER` does with their own data, plus the rights listed after this table |

Rights of an `ADMIN` over the other accounts and the configuration:

- list the accounts;
- create an account;
- disable and enable an account;
- delete an account;
- close the sessions of an account;
- change the name and the profile of an account;
- reset the password of an account;
- read, never write, the balances, months and years of any account ([Read-only views of another account](#read-only-views-of-another-account));
- read the server configuration ([Server page](#server-page));
- edit the configuration ([Configuration page](#configuration-page)).

Rules:

- Profiles are ordered `GUEST < USER < ADMIN`.
- A route requires a minimal profile ([Routes](#routes)). The access level `connected` means `USER` or `ADMIN`.
- A `USER` on a route `/admin/...` gets the page « Accès refusé » (403) with the alert « Cette page est réservée aux administrateurs. ».
- The profile is read from the database on every request, not from the session: a change applies at the next request.

### Logins

- A login matches `[a-z0-9][a-z0-9._-]{1,31}`: 2 to 32 characters among lower-case letters, digits, dot, hyphen and underscore, starting with a letter or a digit.
- A login never changes.

### Account states

| State | Meaning |
| --- | --- |
| `enabled` | The account can log in. Every account is created `enabled`. |
| `disabled` | The account keeps its data but cannot log in. |

- A login to a disabled account gets the same answer as a wrong password: « Identifiant ou mot de passe incorrect. » (401). The attempt is counted by the login throttle.
- Disabling an account closes its sessions.
- A session of a disabled account is refused like one of a deleted account.
- The state is changed from the web only ([Administration](#administration)); `user list` shows it.

### Protection of the last administrator and of one's own account

| Action | On the last enabled `ADMIN` | On one's own account, from the web |
| --- | --- | --- |
| Delete | refused, from the web (409) and the command line (exit 1) | refused (422) |
| Change the profile | refused when it demotes, from the web (409) and the command line (exit 1) | refused (422) |
| Disable | refused (409) | refused (422) |
| Enable | not applicable | refused (422) |
| Reset the password | allowed on another account | refused (422): one's own password changes in the [profile page](#profile-page) |
| Close the sessions | allowed | allowed, from the profile page or the table of `/admin/accounts`; the current session closes too |
| Change the name | allowed | allowed, from the profile page or the account page |

The messages are in [Account page](#account-page).

### Provisional password

1. An account created by an `ADMIN`, or whose password an `ADMIN` resets, gets a provisional password and the flag `must_change_password`.
2. Until the password is changed, every connected route allowed to the profile redirects (303) to `/dashboard/profile`, for `GET` too. The settings page is redirected too.
3. On that page, the password form is marked « (obligatoire) », and the warning `change-password` is shown at the top while the password stays provisional.

Routes that are not redirected: `/dashboard/profile`, `/dashboard/colors.css` and `/logout`.

Accounts created on the command line have the flag only with `--provisional` ([`timesheet user add`](#timesheet-user-add)), and `user passwd` clears it.

### Closing of sessions

| Event | From | Sessions closed | New session |
| --- | --- | --- | --- |
| Change of profile | account page, `user profile` | every session of the account | none |
| Password reset | account page | every session of the account | none |
| Password change | `user passwd` | every session of the account | none |
| Account deleted | accounts page, account page, `user del` | every session of the account | none |
| Account disabled | accounts page, account page | every session of the account | none |
| Own password changed | profile page | every session of the account | one, for the current browser |
| « Déconnecter toutes mes sessions » | profile page | every session of the account, the current one included; the page « Vous êtes déconnecté » follows | none |
| « Fermer les sessions » | accounts page, account page | every session of the account; on one's own account the current one too, and the page « Vous êtes déconnecté » follows | none; a new login stays possible |
| « Se déconnecter » | user menu, `/logout` | the current session | none |
| `session clear` | command line | every session of every account | none |

- `user profile`, `user passwd` and `user del` first remove the sessions of the account from the [session file](#session-file).
- An unreadable session file is then replaced by an empty one, with a warning.
- `user profile` removes the sessions before changing the profile: a refused change (last administrator) has already closed them.
- A session created before its account is refused: a session of a deleted account never comes back with an account recreated under the same login.

### Concurrent password changes

- Login: when the password was changed, or the account deleted or disabled, while the password was being checked, the login gets « Identifiant ou mot de passe incorrect. » (401) and no session.
- Profile page: when the current password was replaced while it was being checked (a reset by an `ADMIN`, another change), the change is refused (409) with « Le mot de passe a changé entre-temps : il n'a pas été modifié. ». Nothing changes and no session opens.

### Start condition of the server

`run` starts even when the database has no account, or no enabled `ADMIN` account. It then logs the warning `no administrator account` with the command that creates one ([Bootstrap](#bootstrap)). Nobody can use the administration until an `ADMIN` is created with `user add` or `user profile`, the server stopped.

## CALENDAR AND DAYS

### Half-days

- Each day has a morning (`am`) and an afternoon (`pm`).
- Each half-day holds an activity code or nothing.
- Every day is entered, weekends and public holidays included: the template has the codes `WE` « Week-end » and `JF` « Jour férié » for them.
- Nothing is prefilled.
- The calendar only names the rows: the name of the public holiday for a public holiday (« Noël »), « Week-end » for a Saturday or a Sunday, « Semaine » for a working day.
- The calendar refuses a consuming code on a weekend or a public holiday ([Day kinds](#day-kinds)).

### Comments

- A day has an optional free comment, for instance a conference, the reason of an absence or « Vacances ».
- A comment is plain text on one line, 500 characters at most, and follows the [free-text rule](#free-text-rule).
- Surrounding spaces are removed.
- A comment can exist alone, without any code, on any tracked day, weekends and public holidays included.
- A comment never changes an indicator or an amount.
- A comment is copied into the label of the debits and credits of its day ([Ledger entries](#ledger-entries)), where it stays once the month is validated.

### Free-text rule

Comments, reasons, labels and names are valid UTF-8 text on one line. They refuse:

- control characters;
- format characters (Unicode category Cf, bidirectional controls included), except the zero-width non-joiner U+200C and the zero-width joiner U+200D, which are accepted;
- line and paragraph separators (Unicode categories Zl and Zp).

| Text | Length | Surrounding spaces | Empty |
| --- | --- | --- | --- |
| Comment of a day | 500 characters at most | removed | allowed |
| Reason of an adjustment | 200 characters at most | removed | refused |
| Reason of a reopening | 200 characters at most | removed | refused |
| Label of an activity code | 1 to 60 characters | removed by the form, refused in a file | refused |
| Label of a counter | 1 to 60 characters | removed by the form, refused in a file | refused |
| Short label of a counter | 1 to 6 characters | removed by the form, refused in a file | refused |
| Name of an account | 60 characters at most | every run of white space, tabs and line breaks included, becomes one space; surrounding spaces removed | allowed |
| Label of a ledger entry (database) | 1000 characters at most | kept | allowed |

Lengths count characters (Unicode code points), not bytes.

### Day kinds

| Kind | Definition | Row label | Consuming code |
| --- | --- | --- | --- |
| Weekend day | Saturday or Sunday | « Week-end » | refused |
| Public holiday | one of the [public holidays](#public-holidays), on any weekday | the name of the public holiday, « \<nom> » | refused |
| Working day | Monday to Friday, not a public holiday | « Semaine » | allowed |

- A code that is not consuming can be entered on any day, and credits its counter wherever it is entered.
- The indicators Jours fériés and Ouvrés follow the calendar of public holidays of the account, whatever is entered.
- A bridge day (a working day the employer gives off between a public holiday and a weekend) entered `JF` stays a working day; it shows in the « Crédit » of `JF`.
- A worked Whit Monday still counts as a public holiday.

### Public holidays

The public holidays of an account are those of its zone of public holidays: the zone chosen in its [settings](#user-settings), otherwise the default zone of the [configuration](#configuration-page), « Métropole » in the template. Every zone has the eleven national public holidays:

| Public holiday | Date or rule |
| --- | --- |
| « Jour de l'an » | 1 January |
| « Lundi de Pâques » | Easter + 1 |
| « Fête du travail » | 1 May |
| « Victoire 1945 » | 8 May |
| « Ascension » | Easter + 39 |
| « Lundi de Pentecôte » | Easter + 50 |
| « Fête nationale » | 14 July |
| « Assomption » | 15 August |
| « Toussaint » | 1 November |
| « Armistice 1918 » | 11 November |
| « Noël » | 25 December |

Some zones add their own public holidays:

| Zone | Identifier | Extra public holidays |
| --- | --- | --- |
| « Métropole » | `fr-metropole` | none |
| « Alsace-Moselle » | `fr-alsace-moselle` | « Vendredi saint », Easter - 2; « Saint-Étienne », 26 December |
| « Guadeloupe » | `fr-guadeloupe` | « Abolition de l'esclavage », 27 May |
| « Guyane » | `fr-guyane` | « Abolition de l'esclavage », 10 June |
| « Martinique » | `fr-martinique` | « Abolition de l'esclavage », 22 May |
| « Mayotte » | `fr-mayotte` | « Abolition de l'esclavage », 27 April |
| « Nouvelle-Calédonie » | `fr-nouvelle-caledonie` | none |
| « La Réunion » | `fr-la-reunion` | « Abolition de l'esclavage », 20 December |
| « Polynésie française » | `fr-polynesie-francaise` | none |
| « Saint-Barthélemy » | `fr-saint-barthelemy` | « Abolition de l'esclavage », 9 October |
| « Saint-Martin » | `fr-saint-martin` | « Abolition de l'esclavage », 27 May until 2017, 28 May from 2018 |
| « Wallis-et-Futuna » | `fr-wallis-et-futuna` | none |
| « Saint-Pierre-et-Miquelon » | `fr-saint-pierre-et-miquelon` | none |

- The identifier of a zone starts with the code of its country, ISO 3166-1 alpha-2 in lower case, and a hyphen (`fr-`), so that zones of other countries can be added later; the rest is the name of the zone in the public dataset of the French administration (calendrier.api.gouv.fr).
- The zones and their dates are those of that dataset; the local public holidays of the territories of the Pacific are not part of it.
- Changing the zone, in the settings of an account or the default zone of the configuration, applies from the first open month: validated months keep their entries and their indicators; their grid shows the current zone.
- Easter Sunday follows the Meeus/Butcher algorithm.
- A public holiday falling on a weekend is labelled with its name on the row, not « Week-end ».

### Today and time zone

- Today is the date in the time zone `server.timezone` of the [server configuration](#server-configuration).
- Today decides which half-days are past (missing) or future (upcoming).
- Today decides which months are over.
- Today decides the planning horizon.
- Today decides the default tracking start of a new account.
- Times of the database are stored in UTC.
- The pages show the dates taken from these times in `server.timezone`: validation, reopening, creation of an account, last login.
- `user list` prints these times as UTC timestamps (RFC 3339).
- The time zone database of the system must hold the zone.

### Tracking start

The tracking start is a date, « Date de début » in the [settings page](#settings-page). By default it is the first day of the month in which the account was created; it can be any day, for instance the day of hiring.

| Period | Display | Save | Counters, indicators, anomalies |
| --- | --- | --- | --- |
| Month before the month of the tracking start | read-only, status « avant le début du suivi » | refused (409) | not counted |
| Days of that month before the tracking start | shown without field, half-days « Hors suivi », comment visible | kept untouched by a save of the month | stored entries ignored; never « Non renseigné » |
| From the tracking start | editable while the month is open and within the planning horizon | accepted | counted |

- Nothing is credited before the tracking start: the initial balances stand for it ([Initial balance](#initial-balance)).
- A tracking start after the first day of its month (a hiring in the course of the month) gives that month no monthly credit: the share acquired is entered as an adjustment.
- The tracking start accepts any date from 2000-01-01 to 2099-12-31, a future one included. When the tracking start falls in a later month, the current month is « avant le début du suivi ».
- The tracking start cannot move after the date of an existing adjustment; it can move to that very date.

### Supported years and planning horizon

- Years 2000 to 2099 only.
- Any other year, or an invalid month, in a URL gives 404.
- In a URL, `{year}` is written on four digits and `{month}` on two digits, with a leading zero: `/dashboard/month/2026/09`. Any other form (`9`, `009`) gives 404.
- Forecasts and the planning horizon stop at December 2099.
- A month can be entered up to 24 months after the current one: this is the planning horizon.
- Beyond it, the grid is read-only with the status « au-delà de l'horizon de planification », and saving is refused (409).

### States of a half-day

| State | Label | Condition | Blocks the validation |
| --- | --- | --- | --- |
| entered | the label of its code | a code is entered | no, unless it is an anomaly |
| missing | « Non renseigné » | no code, tracked day, date before today, whatever the day | yes |
| upcoming | « À venir » | no code, tracked day, date from today on | no, but a month is validated only once over |
| untracked | « Hors suivi » | date before the tracking start | no |

The colour of each state is in [Colours](#colours).

### Size of an account

| Limit | Value | Refusal |
| --- | --- | --- |
| Account record, encoded as compact JSON | `limits.account_bytes`, 1 MiB by default | 413 « Les données de ce compte atteignent la taille maximale autorisée. » |
| Adjustments of one account | `limits.adjustments`, 1000 by default | 413, same message |

The limits are set in the [section limits](#section-limits) of the server configuration.

- The account record is the whole account: login, name, password hash, dates, settings, colours, days, adjustments, months and ledger. An empty account weighs about 400 bytes.
- On disk, where the file is indented, an account takes more room: about 1.8 times for ordinary data, close to 1 time for long comments.
- A save that would grow an account beyond a limit is refused, from the grid, the adjustments, the settings, a validation or a reopening.
- A save that shrinks an account, or keeps its size, is accepted.

How long the limit lasts:

- Validated entries are never removed, except by the [deletion of the data](#deletion-of-the-data), which empties the account.
- With every day entered and every month validated, the default limit is reached after about ten years of tracking.
- It is reached sooner with long comments or many reopenings: each reopening adds a reversal per entry of the month.
- Nothing else in the application then frees space.

## ACTIVITY CODES

### Fields of a code

| Field | Format | Meaning |
| --- | --- | --- |
| `id` | `[A-Z][A-Z0-9_]{0,15}`: one capital letter, then up to 15 capital letters, digits or `_` | identifier, never modified |
| `label` | 1 to 60 characters, [free-text rule](#free-text-rule) | name shown in the lists and the legends |
| `consumes` | identifier of a counter, or `null` | the counter the code debits |
| `feeds` | identifier of a counter, or `null` | the counter the code credits |
| `color` | `#rrggbb`, mandatory | default colour of the code |
| `archived` | `true` or `false` | an archived code is no longer offered |

- Every code names exactly one counter, in exactly one of `consumes` and `feeds`.
- A code has no group: what it represents is its counter.

### Feeding and consuming

| Effect | Amount | Days | Shown in |
| --- | --- | --- | --- |
| Consuming code | debits its counter by 0.5 day per half-day | working days only: refused on a weekend or a public holiday | the « Débit » of its counter |
| Feeding code | credits its counter by 0.5 day per half-day | any day, weekends and public holidays included | the « Crédit » of its counter |

### Codes of the template

| Code | Label | Effect | Counter | Colour |
| --- | --- | --- | --- | --- |
| `TT` | « Temps de travail » | feeds | `TT` | `#33d17a` |
| `WE` | « Week-end » | feeds | `WE` | `#9a9996` |
| `JF` | « Jour férié » | feeds | `JF` | `#ff7800` |
| `CP` | « Congé payé » | consumes | `CP` | `#e01b24` |
| `CSS` | « Congé sans solde » | feeds | `CSS` | `#e01b24` |
| `RTT` | « Récupération temps de travail » | consumes | `RTT` | `#e01b24` |
| `TP` | « Temps partiel » | feeds | `TP` | `#3584e4` |
| `TPC` | « Temps partiel complémentaire » | consumes | `TPC` | `#9141ac` |
| `ABS` | « Absence » | feeds | `ABS` | `#e01b24` |

- The reason of an absence goes into the comment of the day.
- An administrator adds other codes in the [configuration page](#configuration-page).

### Codes offered in a grid

- The lists of a grid offer the codes that are not archived.
- They leave out a consuming code whose counter is disabled for the user or archived.
- A feeding code of a disabled or archived counter stays offered: its half-days count in the indicators and the totals but credit nothing.
- A value already stored on a half-day stays offered, even if its code is archived or its counter disabled.
- A half-day that keeps its stored code is not checked again, by the grid or by an import: neither whether its code is offered, nor the rule of a consuming code on a weekend or a public holiday. A consuming code stored on a day that became a public holiday after a change of zone is therefore kept by a save or an import with `--replace`; it shows as a blocking anomaly of the month ([Anomalies](#anomalies)).

## COUNTERS

### Fields of a counter

| Field | Format | Meaning |
| --- | --- | --- |
| `id` | same format as the `id` of a [code](#fields-of-a-code) | identifier, never modified |
| `label` | 1 to 60 characters, [free-text rule](#free-text-rule) | name shown in the pages |
| `short_label` | 1 to 6 characters | name in the headers of the summary |
| `enabled_by_default` | `true` or `false` | whether the counter is enabled for an account that did not choose |
| `accrual.every` | `month` or `year` | rhythm of the accrual, « par mois » or « par an » |
| `accrual.amount` | a [quantity](#quantities) between 0 and 366 | default amount of each accrual |
| `period_start` | `MM-DD`, 29 February excluded | first day of the period |
| `carry_over` | `true` or `false` | whether the balance is kept into the next period |
| `archived` | `true` or `false` | an archived counter is disabled for every user |

Every counter has a single balance.

### Quantities

| Rule | Value |
| --- | --- |
| Arithmetic | exact rationals, never floats |
| Input | a decimal with at most six decimals, or a fraction (`7/12`); the forms of the pages accept a comma or a point (`2.08`, `-1,5`), files (database, configuration template, import) a point only |
| Denominator of a fraction | a divisor of 63 000 000 (2^6 × 3^2 × 5^6 × 7), for instance `7/12`, `1/3`, `1/7` |
| Magnitude | at most 1 000 000 |
| Written form | a quantity read from a form, a file or the configuration template keeps the form it was written in: `8/12` stays `8/12`, `0.50` stays `0.50`; only the sign, leading zeros and a leading or trailing point are normalised (`+2` is `2`, `.5` is `0.5`, `0001` and `1.` are `1`), and a fraction whose denominator is a power of ten is written as a decimal with as many decimals as that power (`5/10` is `0.5`, `50/100` is `0.50`, `3/1` is `3`) |
| Computed form | the result of a computation (balance, total, sum of a day, rounding) is reduced, and written as a decimal when it is exact: `1/2` is `0.5`, `8/12 + 4/12` is `1`, a balance of `8/12` alone is `2/3` |
| Comparison | equal values compare equal whatever their form: `8/12` equals `2/3` |
| Storage | a string, in its written or computed form: `"2.08"`, `"8/12"`, `"0.5"` |
| Display | rounded to two decimals with a comma: `7/12` is shown `0,58`; the fields of the settings and configuration pages show the written form |
| Display of day counts | `1`, `0,5` |
| Overflow | an arithmetic overflow stops the operation instead of returning a wrong balance |

The bounds on denominators and magnitude keep every sum exact and readable back. The same rules apply to the forms, the database and the configuration template.

### Counters of the template

The labels of the counters are in the plural, those of the codes in the singular: the counter « Congés payés » is fed or consumed by the code « Congé payé ».

| Counter | Label | Short label | Accrual | Fed by | Period start | Carry over | Enabled by default |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `TT` | « Temps de travail » | `TT` | none (0 a year) | `TT`: worked days of the year | `01-01` | no | yes |
| `WE` | « Week-ends » | `WE` | none (0 a year) | `WE`: weekend days entered in the year | `01-01` | no | yes |
| `JF` | « Jours fériés » | `JF` | none (0 a year) | `JF`: days entered `JF` in the year | `01-01` | no | yes |
| `CP` | « Congés payés » | `CP` | 2.08 a month | none (consumed by `CP`) | `06-01` | yes | yes |
| `CSS` | « Congés sans solde » | `CSS` | none (0 a year) | `CSS`: unpaid leave days of the year | `01-01` | no | yes |
| `RTT` | « Récupérations temps de travail » | `RTT` | 8/12 a month (8 days a year) | none (consumed by `RTT`) | `01-01` | no | yes |
| `TP` | « Temps partiels » | `TP` | none (0 a year) | `TP`: part-time days of the year | `01-01` | no | yes |
| `TPC` | « Temps partiels complémentaires » | `TPC` | 0 a month: each employee concerned sets an own amount, `7/12` in the [worked example](#worked-example) | none (consumed by `TPC`) | `01-01` | no | no |
| `ABS` | « Absences » | `ABS` | none (0 a year) | `ABS`: absence days of the year | `01-01` | no | yes |

### Accrual

- A monthly accrual is credited on the last day of the month, labelled « Acquisition \<mois> \<année> ».
- A yearly accrual is credited on the period start day, labelled « Acquisition annuelle \<année> ».
- An accrual of zero writes no entry.
- Only an enabled counter receives an accrual.
- Nothing is credited before the tracking start: the initial balances stand for it.
- A tracking start after the first day of its month gives that month no monthly credit: the share acquired is entered as an adjustment.

### Feeding

- Each half-day of a feeding code credits 0.5 day to its counter, on its date.
- Weekends and public holidays are included.
- The credit is labelled with the label of the code followed by the comment of the day ([Ledger entries](#ledger-entries)).
- Nothing is credited before the tracking start.
- Nothing is credited to a disabled or archived counter.

### Consumption

- Each half-day of a consuming code on a working day debits 0.5 day, on its date.
- Debits follow the chronological order, morning before afternoon.
- The debit is labelled with the label of the code followed by the comment of the day.
- Leave entries keep debiting a counter that the user later disabled, or that an administrator archived.

### End of a period

#### With carry over

On the last day of each period of a counter with `carry_over`:

1. The debits and credits of that day are written.
2. If the counter is enabled and its balance is positive and fractional, an `adjustment` entry « Ajustement : arrondi du solde reporté à l'entier supérieur » rounds it up to the next whole day.
3. The adjustments entered by the user on that day are written after the rounding.
4. The balance is kept into the next period.

- The rounding counts with the adjustments of its month.
- It applies to the balance, not to the days acquired over the period: its amount depends on the leave taken, on fractional initial balances and on adjustments ([Simplifications](#simplifications)).
- A disabled or archived counter is not rounded.

#### Without carry over

On each period start of a counter without `carry_over`, except the first day of the tracking:

1. If the balance is not zero, an `adjustment` entry brings it to zero, whether it is positive or negative: a negative balance is forgiven.
2. This zeroing comes before every other entry of the day.

| Counter | Label of the zeroing |
| --- | --- |
| consumed by some code, archived or not | « Ajustement : perte du solde au \<jj/mm/aaaa> » |
| consumed by no code | « Ajustement : remise à zéro du solde au \<jj/mm/aaaa> » |

The zeroing also applies to a disabled or archived counter.

### Order of the entries of a day

Within one day, the entries of a counter come in this order:

1. the zeroing of a balance at a period start;
2. the debits of consuming codes;
3. the yearly accrual;
4. the credits of feeding codes;
5. on the last day of the month, the monthly accrual;
6. on the last day of a period, the rounding of a carried balance;
7. last, the adjustments entered by the user.

The adjustments come last so that an adjustment corrects the final balance of its day.

### Negative balances

- A balance that becomes negative is accepted.
- It is shown in red (class `is-negative`).
- At the period start of a counter without carry over, a negative balance is zeroed, hence forgiven.

### Disabled and archived counters

| Aspect | Disabled by the user | Archived by an administrator |
| --- | --- | --- |
| Scope | one account | every account |
| Accruals and feeding credits | none | none |
| Debits of existing day entries | kept | kept |
| Consuming codes in the grid | no longer offered | no longer offered |
| Feeding codes in the grid | still offered, credit nothing | still offered, credit nothing |
| New adjustment | accepted | refused (« Compteur inconnu. ») |
| Zeroing at the period start | applies | applies |
| Rounding of a carried balance | none | none |
| History | shown and counted | shown and counted |
| Row in the settings page | shown | hidden |

## USER SETTINGS

The settings are edited in the [settings page](#settings-page).

### Table of the settings

| Setting | Label | Default | Bounds | After the first validation |
| --- | --- | --- | --- | --- |
| Tracking start | « Date de début » | first day of the month of the account creation | 2000-01-01 to 2099-12-31, not after the date of an existing adjustment | locked |
| Zone of public holidays | « Calendrier des jours fériés » | « Par défaut (\<zone>) »: the default zone of the configuration, followed when it changes | one of the [zones](#public-holidays) | applies from the first open month |
| Counter enabled | « Activé » | `enabled_by_default` of the counter | | applies from the first open month |
| Own amount | « Montant propre » | empty: the default amount | 0 to 366 | applies from the first open month |
| Initial balance | « Solde initial » | 0 | -366 to 366 | locked |
| Colour of a code | « Couleur » | the colour of the configuration | `#rrggbb` | applies at once |

- A counter added to the configuration later follows its `enabled_by_default`, even for an account that saved its settings before.
- The theme is not a setting of the account: it is chosen with the theme button and kept in the browser only ([Theme button](#theme-button)).
- The colours are described in [Colours](#colours).

### Initial balance

1. The initial balance of a counter is its balance at the start of the first day of the tracking.
2. It is written as an `adjustment` entry « Ajustement : solde initial » on that day, before the other entries of the day.
3. No zeroing happens on that day.
4. A yearly accrual due that day is added on top.

### Lock after the first validation

- While some month is validated, the tracking start and the initial balances are locked: the fields are disabled, and values posted anyway are ignored without error.
- Corrections then go through adjustments.
- The lock follows the history of the account: it is lifted when the first validated month is reopened.
- The other settings (zone of public holidays, enabled, own amount) apply from the first open month: validated months keep their entries.

### Stale settings form

The form carries the account revision and the configuration revision.

| Case | Status | Message |
| --- | --- | --- |
| The configuration changed since the form was shown (a counter added meanwhile would otherwise be saved as disabled) | 409 | « La configuration a changé entre-temps. Vérifiez puis recommencez. » |
| The data of the account changed since the form was shown | 409 | « Les données ont changé entre-temps. Vérifiez puis recommencez. » |

## ADJUSTMENTS

An adjustment corrects the balance of a counter at a date. There is no forced balance: a gap with a payslip is entered as an adjustment.

| Field | Label | Rule |
| --- | --- | --- |
| Date | « Date » | in an open month of the tracking, not before the tracking start, within the planning horizon |
| Counter | « Compteur » | an existing counter that is not archived; a counter disabled by the user is accepted |
| Quantity | « Quantité signée » | a signed, non-zero [quantity](#quantities) between -366 and 366, with a comma or a point, or a fraction |
| Reason | « Motif » | mandatory, one line, 200 characters at most, [free-text rule](#free-text-rule), surrounding spaces removed |

- An adjustment is added and deleted in the [activity page](#activity-page) of its month.
- An adjustment receives an identifier.
- An account holds at most `limits.adjustments` adjustments, 1000 by default ([Size of an account](#size-of-an-account)).
- An adjustment can be deleted while its month is open.
- When its month is validated, an adjustment becomes a frozen `adjustment` entry « Ajustement : \<motif> ».
- The refusals are listed in [Refusals of the activity page](#refusals-of-the-activity-page).

## COUNTER BOOKKEEPING

### Principle

- Each counter of an account is kept like a bank account.
- Its journal is a list of dated entries.
- A statement gives its entries month by month.
- Its balance is the sum of its entries.

### Ledger entries

#### Fields

An entry has:

1. an identifier;
2. its month;
3. its date;
4. its counter;
5. its kind;
6. a signed amount;
7. a label;
8. an optional reference to another entry;
9. its creation time.

#### Kinds

| Kind | Label | Written by | Reference |
| --- | --- | --- | --- |
| `debit` | the label of the consuming code, then « , \<commentaire> » when the day has a comment | each day with a consuming code | none |
| `credit` | the label of the feeding code, then « , \<commentaire> » when the day has a comment (« Absence, Grippe ») | each day with a feeding code | none |
| `credit` | « Acquisition \<mois> \<année> » | the monthly accrual | none |
| `credit` | « Acquisition annuelle \<année> » | the yearly accrual | none |
| `adjustment` | « Ajustement : \<motif> » | an adjustment entered by the user | the adjustment |
| `adjustment` | « Ajustement : solde initial » | the initial balance, on the first day of the tracking | none |
| `adjustment` | « Ajustement : arrondi du solde reporté à l'entier supérieur » | the rounding of a carried balance | none |
| `adjustment` | « Ajustement : perte du solde au \<jj/mm/aaaa> » | the zeroing of a consumed counter | none |
| `adjustment` | « Ajustement : remise à zéro du solde au \<jj/mm/aaaa> » | the zeroing of a counter that no code consumes | none |
| `reversal` | « Contre-passation : \<motif> » | a reopening | the cancelled entry |

- There is one debit or credit entry per day and per counter: a day with `CP` in the morning and `RTT` in the afternoon gives two debits.
- When two codes of the same counter share a day, the label is the one of the morning code.
- Once the month is validated, the label of a debit or a credit keeps the comment of its day, even if the day is edited after a reopening, until the data of the account or the account itself is deleted. This includes the reason of an absence entered `ABS` ([Personal data](#personal-data)).

### Open month

- The entries of an open month are provisional.
- They are recomputed on each display, from the day entries, the user settings and the rules in force.
- Only the day entries and the adjustments of an open month are stored.

### Validation

#### Conditions of a validation

A month can be validated when all these conditions hold, checked in this order:

1. it is not before the month of the tracking start;
2. it is not validated already;
3. it is over (today is after its last day);
4. its previous month is validated, or it is the month of the tracking start;
5. it has no missing half-day;
6. it has no blocking anomaly.

#### Anomalies

| Anomaly | Blocking | Text listed above the grid |
| --- | --- | --- |
| A consuming code on a weekend or a public holiday | yes | « \<jour>, \<Matin ou Après-midi> : « \<libellé> » est décompté et ne peut pas être posé un week-end ou un jour férié » |
| A consuming code whose counter is disabled or archived | no | « \<jour>, \<Matin ou Après-midi> : le compteur « \<libellé> » n'est pas activé » |
| An unknown code | yes | cannot occur: the load of the database refuses it |

#### Effects of a validation

1. The provisional entries of the month are frozen in the ledger (`ledger`), in order, each with an identifier.
2. A snapshot of the indicators of the month is stored.
3. A snapshot of the effective amount of every counter enabled for the account (not archived, not disabled) is stored (`rules`).
4. The month is marked `validated` with its validation time.
5. The history of the month records « Validé le \<jj/mm/aaaa> ».
6. The month becomes read-only: grid, comments and adjustments.

#### Disabled button

The button « Valider le mois » is shown disabled, with the reason, when the validation is not allowed.

| Condition not met | Reason shown |
| --- | --- |
| The month is before the tracking start | « Le mois précède le début du suivi. » |
| The month is not over | « Le mois n'est pas terminé. » |
| The previous month is open | « Le mois précédent doit être validé d'abord. » |
| A half-day is missing | « Des demi-journées ne sont pas renseignées. » |
| A blocking anomaly exists | « Des saisies sont invalides. » |
| Any other case (the month is already validated) | « Le mois ne peut pas être validé. » |

The validation form is not shown for a month before the tracking start, and the reopening form replaces it for the last validated month.

#### Validation request

- `POST /dashboard/activity/{year}/{month}/validate`, with the account revision.
- Success: 303 back to the activity page of the month, notice `validated` « Le mois a été validé : ses écritures sont figées. ».
- Refusal: 409 « Validation impossible : \<raison> », the reason being one of [Disabled button](#disabled-button) ([Refusals of the activity page](#refusals-of-the-activity-page)).

### Reopening

#### Conditions of a reopening

- Only the last validated month can be reopened.
- A reason is mandatory: one line, 200 characters at most, [free-text rule](#free-text-rule).

#### Effects of a reopening

1. The entries of the month are never erased.
2. Each active entry of the month receives a `reversal` entry of opposite amount, « Contre-passation : \<motif> », referencing it.
3. The reversal has the same date as the cancelled entry, so that every intermediate balance stays right.
4. The reversal is created at the reopening time and attached to the month.
5. The month becomes open again; its validation time, rules and indicators are cleared.
6. The history of the month records « Rouvert le \<jj/mm/aaaa> : \<motif> ».
7. The month is then validated again normally, recomputed with the rules in force.
8. The statement lists the cancelled entries in a collapsed history (`<details>` « Écritures annulées par contre-passation (\<nombre>) »).

#### Reaching an older month

1. Reopen the last validated month.
2. The month before it becomes the last validated month; reopen it in turn.
3. Repeat until the month to correct is open.
4. Correct it, then validate again every reopened month, in order.

#### Reopening request

- `POST /dashboard/activity/{year}/{month}/reopen`, with the account revision and the field `reason`.
- Success: 303 back to the activity page of the month, notice `reopened` « Le mois a été rouvert : ses écritures ont été contre-passées. ».
- Refusal: 409 « Seul le dernier mois validé peut être rouvert. » or 409 « Indiquez le motif de la réouverture (une ligne, 200 caractères au plus). ».

### Correcting without reopening

- An error in an older month can also be corrected by an adjustment in an open month.
- Such an adjustment corrects the balance only.
- The indicators and the summary of the erroneous month stay as validated.
- The adjustment counts in the balance of its own month.

### Rule changes

- A change of a counter rule or of the configuration never touches a validated month.
- It applies from the first open month of each account.
- The period start, the rhythm (`accrual.every`) and the carry over of a counter can change: the new rule applies from the first open month, so two period ends, or none, can then occur within twelve months.
- A change of the user settings (enabled, own amount) also applies from the first open month.

### Invariants

These invariants are checked by the tests. The database also enforces invariant 2 at every write of an account's data, whatever the page or the command: the entries of the ledger are kept, in their order, and only appended to, with the entries of a month validated by that write or the reversals of the last validated month that it reopens; the days and the adjustments of a validated month stay the same, and so do its validation date, its frozen rules, its indicators and its history while it stays validated; a validated month stays validated, except the last one, which a reopening opens again; the tracking start does not move while a month is validated. A write that breaks it is refused as an internal error (500, or exit 1 on the command line), and nothing is written. The deletion of the data is the only exception.

1. The balance equals the sum of the entries.
2. A validated month is never rewritten; only the [deletion of the data](#deletion-of-the-data) erases the whole tracking of an account at once.
3. The frozen entries equal the provisional ones computed just before the validation.
4. Reopening then validating again, without any change of day entries, settings or rules in between, gives the same balance.
5. A reopened month is recomputed with the rules in force at the reopening.
6. The credits of a reopened then revalidated month are counted once.

## MONTHLY INDICATORS

Indicators are computed in days (a half-day is 0.5) for one month, over every day of the month.

| Indicator | Definition |
| --- | --- |
| Jours fériés (« Fériés » in the summary) | public holidays from Monday to Friday |
| Total (« Total ») | days from Monday to Friday |
| Ouvrés (« Ouvrés ») | Total minus Jours fériés |
| X, per counter | half-days of the codes that feed or consume counter X, every day of the month: `TT` the worked days, `WE` the weekend days, `TP` the part-time days, `CP` the paid leave taken |
| « Non renseignés » | missing half-days of a tracked month, every day of the month |

- Jours fériés, Total and Ouvrés always cover the whole month, the days before the tracking start included.
- The days before the tracking start count neither their codes nor « Non renseignés ».

### Invariant of the indicators

Over the tracked days of a month:

```text
sum of the half-days of every counter + « Non renseignés » + upcoming half-days = number of tracked days
```

When the tracking started before the month, the number of tracked days is the number of days of the month.

## PAGE STRUCTURE

### List of the pages

| Page | Title (`h1`) | Route | Access | Section |
| --- | --- | --- | --- | --- |
| Home | « Timesheet » | `/` | public | [Public pages](#public-pages) |
| Login | « Connexion » | `/login` | public | [Public pages](#public-pages) |
| Logout confirmation | « Déconnexion » | `/logout` | connected | [Public pages](#public-pages) |
| Logged out | « Vous êtes déconnecté » | after `POST /logout` | connected | [Public pages](#public-pages) |
| Dashboard | « Tableau de bord » | `/dashboard/` | connected | [Dashboard](#dashboard) |
| Activity page | « \<Mois> \<année> » | `/dashboard/activity/{year}/{month}` | connected | [Activity page](#activity-page) |
| Month page | « \<Mois> \<année> » | `/dashboard/month/{year}/{month}` | connected | [Month page](#month-page) |
| Year page | « Année \<année> » | `/dashboard/year/{year}` | connected | [Year page](#year-page) |
| Profile page | « Mon profil » | `/dashboard/profile` | connected | [Profile page](#profile-page) |
| Subscriptions page | « Mes souscriptions » | `/dashboard/subscriptions` | connected | [Subscriptions](#subscriptions) |
| Settings page | « Réglages » | `/dashboard/settings` | connected | [Settings page](#settings-page) |
| Data page | « Données » | `/dashboard/data` | connected | [Data page](#data-page) |
| Assistant page | « Assistant » | `/dashboard/assistant` | connected | [AI assistant](#ai-assistant) |
| Conversation page | « \<titre de la conversation> » | `/dashboard/assistant/{id}` | connected | [Conversations](#conversations) |
| Server page | « Serveur » | `/admin/` | `ADMIN` | [Server page](#server-page) |
| Accounts page | « Comptes » | `/admin/accounts` | `ADMIN` | [Accounts page](#accounts-page) |
| Account page | « Compte \<login> » | `/admin/accounts/{login}` | `ADMIN` | [Account page](#account-page) |
| Read-only views | « Soldes de \<login> », « \<Mois> \<année> de \<login> », « Année \<année> de \<login> » | `/admin/accounts/{login}/...` | `ADMIN` | [Read-only views of another account](#read-only-views-of-another-account) |
| Configuration page | « Configuration » | `/admin/settings` | `ADMIN` | [Configuration page](#configuration-page) |
| Error pages | « Page introuvable », « Accès refusé », and the titles of [Refusals](#refusals) | any | any | [Public pages](#public-pages) |

### Document skeleton

- `<html lang="fr">` with `<meta name="color-scheme" content="light dark">`.
- `body` is a grid of three rows: `<header>`, `<main>`, `<footer>`. The footer stays at the bottom of short pages.
- One `h1` per page, headings without skipped level, sections with `h2`.
- Data are in tables with `<caption>`, `<thead>` and `<th scope>`.
- Forms use `<fieldset>` and `<legend>`, and `<label class="field">` wrapping a `<span class="field-name">` and its control.
- No `div`, no `id`.
- The authoring rules of the HTML templates are in [CODING.md](CODING.md#templates).

### Header

- The header is a dark bar, sticky at the top, full width, with its content on 1200 px.
- It holds the brand: the logo and « Timesheet », a link to `/dashboard/` when connected and to `/` otherwise.
- It holds the main `<nav>` of the family of the page.
- On the pages of the dashboard and admin families, it holds the [user menu](#user-menu).
- Every family shows a [theme button](#theme-button), connected or not, after the user menu when there is one.

#### Navigation of each family

| Family | Pages | Entries | `aria-label` of the `<nav>` |
| --- | --- | --- | --- |
| public | home, login, logout, error pages for a guest | for a guest, a link to `/login` shown as an icon only, an arrow entering a door drawn in the text colour, with `title` and `aria-label` « Se connecter »; for a connected browser, a « Tableau de bord » button, preceded for an `ADMIN` by an « Administration » button; nothing on the login page | « Navigation principale » |
| dashboard | dashboard, activity, month, year, profile, subscriptions, settings, data and assistant pages | for an `ADMIN`, « Administration », a link to `/admin/`; then « Tableau de bord », « Activité », « Mois », « Année »; then « Assistant », a link to `/dashboard/assistant`, only while the account can use a [subscription](#subscriptions) | « Navigation principale » |
| admin | server, accounts, account, configuration pages and read-only views | « Tableau de bord », a link to `/dashboard/`; then « Serveur », « Comptes », « Configuration » | « Navigation de l'administration » |

- The current entry carries `aria-current="page"` and is highlighted.
- Every entry of a menu is shown on every page of its family, except « Assistant », shown only to an account that owns a subscription or has one shared with it and not hidden.

#### Theme button

- The button shows a sun (`sun.svg`) while the light theme is shown, and a moon (`moon.svg`) while the dark theme is shown.
- It has `aria-label="Thème sombre"` and `aria-pressed`.
- It is shown by `chrome_page.js` and hidden without JavaScript.
- It switches between the light and the dark theme.
- It keeps the choice in the `localStorage` of the browser under the key `theme`, never on the server. When the browser refuses that storage, the choice still switches and lasts until the next page.
- With no stored choice, the theme follows the system.
- It is the last element of the header of the public family.

#### Narrow windows

| Window width | Change |
| --- | --- |
| any | the name of the user beside its badge is cut with an ellipsis beyond 12 rem; a long value of a list of facts (an address, a path, a list of origins) wraps at any character instead of overflowing its card |
| up to 1024 px | the dashboard and admin families replace their `<nav>` by a burger menu |
| up to 720 px | the name of the user beside its badge is hidden |
| up to 600 px | the lists of facts (server page, profile page, account page) put each value under its label |
| up to 480 px | the text of the brand is visually hidden; it stays the name of the link and the logo alone remains |

The burger menu:

- is a `<details class="nav-menu">` placed last, at the right end of the header, after the theme button;
- has a `<summary>` showing the icon `menu.svg` (`alt="Menu"`);
- is framed like the theme button: same size, a thin border in the soft colour of the bar, brighter on hover and while open;
- opens, below it, the same entries in a `<nav>` with the same label;
- works without JavaScript;
- is closed by `chrome_page.js` on a click outside or on `Échap`, like the user menu.

### User menu

#### Summary

- The user menu is a `<details class="user-menu">`.
- Its `<summary>` shows the name of the user, or the login when no name is set.
- Before it, a decorative badge (`<span class="avatar" aria-hidden="true">`) shows two initials of that label, in capitals:
  - with several words, the first letter of the first word, then the first letter of the last word: « OP » for « Olivier PONCET »;
  - with a single word, its first two letters: « DE » for the login `demo`.
- Words are separated by spaces; only letters and digits count, so a word without any is ignored.
- It opens without JavaScript, by click or keyboard.
- `chrome_page.js` closes it on a click outside or on `Échap`.

#### Entries

The panel is a `<nav aria-label="Compte">`, the same on every page of the dashboard and admin families, for every profile. It holds only what concerns the user:

1. the profile as its heading (`<p class="menu-heading">`): « Administrateur » or « Utilisateur »;
2. « Mon profil »;
3. « Mes souscriptions », which opens the page of the AI subscriptions;
4. « Mes réglages », which opens the settings page « Réglages »;
5. « Mes données », which opens the data page « Données »;
6. the form « Se déconnecter ».

- The entry of the current page carries `aria-current="page"`.
- « Administration » is not in the user menu: it is an entry of the header ([Navigation of each family](#navigation-of-each-family)).

#### Logout form

- « Se déconnecter » posts to `POST /logout` with the CSRF token, without confirmation.
- The confirmation page `GET /logout` stays available.

### Main content

- The result of an action is always shown at the top of `<main>`, before the page header, on every page, in three levels with the usual colours:

| Level | Meaning | Markup | Colour |
| --- | --- | --- | --- |
| alert | the action is refused or failed | `<p role="alert">`, details in `<ul class="alert-details">` | red (`--danger`) |
| warning | the action is done or possible, with points to check | `<p role="status" class="is-warning">`, details in `<ul class="warning-details">` | yellow (`--warning`) |
| notice | the action succeeded | `<p role="status">`, details in `<ul class="notice-details">` | green (`--success`) |

- They come in this order: alert, warning, notice.
- The notices and warnings that follow a redirection are listed in [Notices](#notices).
- Field-level refusals of a grid or a form (an invalid half-day, a counter row) stay next to their field, in addition to the alert.
- Then comes the content, which opens with a page header `<header class="page-header">` holding the `h1` and a line of explanation.
- Sections are cards.
- The login, logout and logged-out pages show a single centred card: the login page holds the logo, its `h1` and its form; the logout page its `h1`, an explanation and its form; the logged-out page its `h1` and an explanation.

### Footer

- The footer is centred and has no link.
- It shows the brand: the logo and « Timesheet ».
- It never shows the version, connected or not: the version appears only on the [server page](#server-page).
- It ends with two lines: « Logiciel libre sous licence GNU GPL version 2 ou ultérieure », then « Copyright © 2023-\<année> - Olivier Poncet. », the year being the current one.

### Columns of tiles

The tiles of the dashboard and the features of the home page are laid out as follows.

| Window width | Columns |
| --- | --- |
| above 1120 px | 4 |
| up to 1120 px | 2 |
| up to 600 px | 1 |

### Dates

| Context | Format | Example |
| --- | --- | --- |
| Dates in the pages | `dd/mm/yyyy` | « Soldes au 11/11/2026 », « validé le 31/10/2026 » |
| Row of the grid | weekday and `dd/mm/yyyy` | « mardi 03/11/2026 » |
| Period start | `dd/mm` | « 01/06 » |
| Labels of the ledger | `dd/mm/yyyy` | « Ajustement : perte du solde au 01/01/2026 » |
| `aria-label` of the fields of the grid | day and month in words, easier to hear | « Matin du 3 novembre » |

### Themes

- `chrome_page.js` sets `data-theme="light"` or `data-theme="dark"` on `<html>`, with the matching `color-scheme`, when the browser keeps a choice.
- `theme.css` defines the light tokens on `:root`.
- It defines the dark tokens under `:root[data-theme="dark"]`.
- For the automatic theme, it defines them under `@media (prefers-color-scheme: dark)` on `:root:not([data-theme="light"])`.
- The colours of the day types are the same in both themes.

### Scripts

The site works without JavaScript.

| Script | Pages | What it does |
| --- | --- | --- |
| `chrome_page.js` | every page | shows the theme button and switches the theme; closes the burger menu, the user menu and the deletion box of `/admin/accounts` on a click outside or `Échap`; gives the focus to the confirmation field of the deletion box |
| `month_sheet_page.js` | the activity page of an editable month | copies the « Journée » choice into the half-days, empties it when a half-day of the row is changed by hand, repaints the cells |

## PUBLIC PAGES

| Page | Route | Content | Links and buttons in the main content |
| --- | --- | --- | --- |
| Home | `GET /` | a title block, `h1` « Timesheet » and a lead line, then the section « Ce que permet l'application » with four features | « Ouvrir le tableau de bord » when connected; no login button |
| Login | `GET /login` | the form « Connexion » with « Identifiant » and « Mot de passe » | the button « Se connecter » only; a connected browser is redirected (303) to `/dashboard/` |
| Logout confirmation | `GET /logout` | « Déconnexion » and the form | « Se déconnecter » and « Revenir au tableau de bord »; a guest is redirected (303) to `/login` |
| Logged out | after `POST /logout`, « Déconnecter toutes mes sessions », or closing one's own sessions | « Vous êtes déconnecté » and « Votre session est fermée. » | none |
| Not found | any unknown path | « Page introuvable » and « Cette adresse ne correspond à aucune page. » | « Revenir à l'accueil » |
| Refused | 403 | « Accès refusé » with an alert | « Revenir à l'accueil » |
| Other errors | 400, 405, 414, 421, 429, 500 | a title, a line and « (code \<statut>) » ([Refusals](#refusals)) | « Revenir à l'accueil » |

The login and its refusals are described in [Routes](#routes) and [SECURITY.md](SECURITY.md#login-throttling).

## DASHBOARD

### Page header

- Route `/dashboard/`, entry « Tableau de bord » of the main navigation.
- `h1` « Tableau de bord » and the line « Soldes des compteurs au \<date du jour>. ».
- One section only, `h2` « Soldes au \<date du jour> », holding the tiles.

### Counter tiles

- One tile per counter that is enabled, or whose balance today is not zero.
- Each tile is an `<article>` with the counter as `h3`.
- No warning announces a coming loss.

| Value | Label | Definition |
| --- | --- | --- |
| Balance today | the large figure, followed by « à ce jour » | the validated entries plus the provisional entries dated up to today: debits, credits and adjustments |
| Validated balance | « Validé » | the balance at the end of the last validated month |
| Forecast | « Prévision au \<date> » | the balance at the end of the current period of the counter, adding planned leave and future credits; the date never goes beyond 31/12/2099 |

The three values form the position of the counter. A negative value is shown in red.

### Tile « Total »

- The last tile, « Total », counts the days entered in the current year.
- It counts from 1 January, or from the tracking start when it is later.
- It counts every code, the days entered in advance included.
- Its unit is « jours saisis », or « jour saisi » under 2.
- It equals the total of the table « Totaux de l'année » of the year page minus its row « Non renseignés ».
- « Depuis le » gives the date it starts from.
- « Dont à venir » gives the part entered after today.
- « Non renseignés » gives the missing days before today.
- With a tracking start in a later year, it shows 0 from that start.

## ACTIVITY PAGE

- Route `/dashboard/activity/{year}/{month}`, entry « Activité » of the main navigation (`/dashboard/activity` redirects to the current month).
- The page holds the entry of one month, and only it: half-days, validation and adjustments.
- `h1` « \<Mois> \<année> » and the line « Saisie des demi-journées, validation et ajustements du mois. ».
- After a save, a validation, a reopening or an adjustment, the server redirects to this page.

The page holds, in this order:

1. the [month toolbar](#month-toolbar);
2. the section « Saisie »;
3. the section « Ajustements de \<Mois> \<année> », when the month has adjustments or is open.

### Month toolbar

The activity page and the month page share the same toolbar:

- a `<nav aria-label="Mois">` holding a list of three links: « Mois précédent », « Mois en cours », « Mois suivant »;
- « Mois précédent » is absent in January 2000, « Mois suivant » in December 2099;
- each link opens the same page for another month: the toolbar of the activity page stays on activity pages, the one of the month page on month pages.

### Section « Saisie »

#### Status of the month

The `h2` « Saisie » carries the status of the month as a badge.

| Status | Condition, in order |
| --- | --- |
| « avant le début du suivi » | the month is before the month of the tracking start |
| « au-delà de l'horizon de planification » | the month is after the planning horizon |
| « validé le \<jj/mm/aaaa> » | the month is validated |
| « ouvert » | otherwise |

The section then holds a legend of the codes and of the state « Non renseigné », and the list of the [anomalies](#anomalies) of the month.

#### Grid

- The grid is a `<table>` with one row per day, captioned « Saisie de \<Mois> \<année> : \<n> j non renseigné(s) ».
- Each row starts with a `<th scope="row">` holding the `<time>` and the label of its day kind: the name of the public holiday, « Week-end » or « Semaine » ([Day kinds](#day-kinds)).
- The columns are « Jour », « Matin », « Après-midi », « Journée » (only when the month is editable) and « Commentaire ».
- Each `<select>` and comment field is named by an `aria-label`: « Matin du 3 novembre », « Après-midi du 3 novembre », « Journée du 3 novembre », « Commentaire du 3 novembre ».
- The lists offer the codes of [Codes offered in a grid](#codes-offered-in-a-grid).
- A month that is not editable shows the same table without fields.

#### Column « Journée »

- When set, « Journée » imposes its code on both half-days; the server applies it.
- When empty, the half-days are kept as chosen.
- `month_sheet_page.js` copies the « Journée » choice into the half-days, so that the half-days shown are the ones saved.
- It empties the « Journée » choice as soon as a half-day of the row is changed by hand.
- It repaints the cells with the colour of their code.

#### Saving

1. « Enregistrer le mois » posts the whole month with the account revision.
2. An empty field clears a half-day.
3. Every tracked day is checked like an entry: offered code, consuming code on a working day only, comment rule.
4. When a problem is found, nothing is saved and the problems are shown on their rows.
5. Otherwise the month is saved and the page comes back with the notice `saved`.

#### Validation and reopening forms

- Below the grid, the form « Valider le mois » ([Validation](#validation)) or, for the last validated month, the form « Rouvrir le mois » with a field « Motif » ([Reopening](#reopening)).
- The validation form explains: « La validation fige les écritures du mois sur les compteurs ; seul le dernier mois validé pourra être rouvert. ».
- The reopening form explains: « Ses écritures seront contre-passées puis le mois redeviendra modifiable. ».

#### Month history

- A collapsed `<details>` « Historique du mois » lists the events of the month.
- Each event is « Validé le \<jj/mm/aaaa> » or « Rouvert le \<jj/mm/aaaa> : \<motif> ».

### Section « Ajustements »

- The `h2` is « Ajustements de \<Mois> \<année> », for instance « Ajustements de Février 2026 ».
- The table « Ajustements du mois » has the columns « Date », « Compteur », « Quantité », « Motif » and, in an open month, « Action » with a button « Supprimer ».
- In an open month, the form « Ajouter un ajustement » follows ([Adjustments](#adjustments)).

### Refusals of the activity page

| Request | Case | Status | Message |
| --- | --- | --- | --- |
| Save | the month is validated, before the month of the tracking start, or beyond the planning horizon | 409 | « Ce mois n'est pas modifiable : il est validé, précède le début du suivi ou dépasse l'horizon de planification de 24 mois. » |
| Save | a tracked day of the month has no comment field in the form (a truncated or hand-made request), with the current revision | 400 | page « Requête invalide »: nothing is saved instead of clearing the missing days |
| Save | invalid entries | 422 | « Certaines saisies sont invalides, rien n'a été enregistré. », and on each row: « Matin : code d'activité non disponible », « Après-midi : « \<libellé> » est décompté et ne peut pas être posé un week-end ou un jour férié », « commentaire : une seule ligne de 500 caractères au plus, sans caractère de contrôle » |
| Validation | a condition is not met | 409 | « Validation impossible : \<raison> » ([Disabled button](#disabled-button)) |
| Reopening | not the last validated month | 409 | « Seul le dernier mois validé peut être rouvert. » |
| Reopening | missing or invalid reason | 409 | « Indiquez le motif de la réouverture (une ligne, 200 caractères au plus). » |
| Adjustment | unknown `action` | 400 | page « Requête invalide » |
| Adjustment | unreadable date | 422 | « Date d'ajustement invalide. » |
| Adjustment | date before the tracking start, in a validated month or beyond the planning horizon | 422 | « Un ajustement se saisit dans un mois ouvert, à partir du début du suivi. » |
| Adjustment | unknown or archived counter | 422 | « Compteur inconnu. » |
| Adjustment | invalid quantity | 422 | « Quantité invalide : un nombre signé non nul entre -366 et 366, par exemple -1,5 ou 7/12. » |
| Adjustment | missing or invalid reason | 422 | « Le motif est obligatoire (une ligne, 200 caractères au plus). » |
| Adjustment deletion | the adjustment belongs to a validated month | 422 | « Cet ajustement appartient à un mois validé. » |
| Adjustment deletion | unknown identifier | 422 | « Ajustement inconnu. » |
| Any of them | the account changed since the form was shown (another tab), or the revision field is stale | 409 | « Les données ont changé entre-temps (autre onglet ?). Rechargez la page avant de recommencer. » |
| Any of them | unreadable revision | 400 | page « Requête invalide » |
| Any of them | the account would grow beyond its [size limits](#size-of-an-account) | 413 | « Les données de ce compte atteignent la taille maximale autorisée. » |
| Any of them | the database file would grow beyond `database.max_bytes` (64 MiB by default), or another write error | 500 | page « Erreur interne » |

A save of the grid, a validation, a reopening and an adjustment check their revision right after the month: a stale page gets 409 even when some of its values are invalid, so that the page shown after a refusal never carries a newer revision than the values it shows.

## MONTH PAGE

- Route `/dashboard/month/{year}/{month}`, entry « Mois » of the main navigation (`/dashboard/month` redirects to the current month).
- The page is for consultation only, like the year page: it holds no form.
- `h1` « \<Mois> \<année> » and the line « Totaux, calendrier, synthèse et relevés des compteurs du mois. ».

The page holds, in this order:

1. the [month toolbar](#month-toolbar);
2. the section « Totaux du mois »;
3. the section « Calendrier »;
4. the section « Synthèse de \<Mois> \<année> »;
5. one section per counter statement.

### Section « Totaux du mois »

The table « Totaux de l'année » of the [year page](#totals-of-the-year), reduced to the month.

### Section « Calendrier »

- The `h2` « Calendrier » carries the status of the month as a badge ([Status of the month](#status-of-the-month)).
- It holds the legend and the [calendar](#calendar) of the year page, reduced to the month.

### Section « Synthèse »

- The [yearly summary](#yearly-summary) reduced to the month.
- Its first row, « Report du mois précédent », gives the balance of each counter on the eve of the first day of the month.
- Then come the row of the month and the row « Total ».

### Counter statements

- One section per counter that is enabled, or that has entries or cancelled entries in the month, in the order of the configuration.
- The `h2` is the label of the counter (« Temps de travail », « Week-ends »...), with the badge « provisoire » while the statement has provisional lines.
- The table is captioned « Relevé \<compteur> de \<Mois> \<année> », for instance « Relevé Congés payés de Février 2026 ».

| Row or column | Content |
| --- | --- |
| first row | opening balance |
| one row per active entry | « Date », « Libellé », « Nature » (the kind), « Débit », « Crédit », « Solde » (running balance) |
| last row | closing balance |
| `<details>` « Écritures annulées par contre-passation (\<nombre>) » | the cancelled entries and their reversals |

## YEAR PAGE

- Route `/dashboard/year/{year}`, entry « Année » of the main navigation.
- The page is for consultation only: it holds no form.
- `h1` « Année \<année> » and the line « Calendrier des demi-journées, synthèse mensuelle, compteurs et totaux par code d'activité. ».

The page holds, in this order:

1. the year navigation;
2. the table « Totaux de l'année »;
3. the calendar;
4. the yearly summary;
5. one section per counter.

### Year navigation

- A `<nav aria-label="Années">` with « Année précédente », « Année en cours », « Année suivante ».
- « Année précédente » is absent in 2000, « Année suivante » in 2099.

### Totals of the year

- Captioned « Jours par code d'activité, week-ends et fériés compris », with the columns « Code d'activité » and « Jours ».
- One row per activity code with its days, zero included. An archived code has a row only when it has days.
- One row « Non renseignés » for the missing days.
- A last row « Total » (`<tfoot>`) sums them.
- The months before the tracking start are not counted.

### Calendar

- One column per day of the month (1 to 31) and two rows per month: the morning above, the afternoon below.
- One `<tbody>` per month, the month as `<th scope="rowgroup">`, the half-day as `<th scope="row">`.
- Each cell is coloured like the grid and shows the identifier of the code entered (`TT`, `WE`, `CP`...), or « ? » for a missing half-day.
- The full text (date, half-day, label of the code, public holiday name, comment) is in an `<abbr title>`.
- A commented day is marked by a line at the bottom of its cell.
- A legend of the codes comes with the table.
- A day that does not exist (30 February, 31 April) is black ([Colours](#colours)).
- Hovering any part of a month highlights the whole month: its name, its morning row and its afternoon row, never one row alone.

### Yearly summary

The section « Synthèse de l'année \<année> » holds one table, January to December.

#### Columns

| Group | Column | Content |
| --- | --- | --- |
| | « Mois » | the month, a link to its month page, and its status |
| one group per counter, headed by its short label | « Débit » | the debits of the month; only for a counter that some code consumes |
| same | « Crédit » | the credits of the month, adjustments excluded: accruals and credits of feeding codes (the « Crédit » of `TT` is the worked days, of `TP` the part-time days, of `ABS` the absences) |
| same | « Ajustement » | the adjustments of the month, signed: initial balance, roundings, zeroings and adjustments entered by the user |
| same | « Solde » | the balance at the end of the month |
| « Jours » | « Ouvrés » | Total minus Fériés |
| same | « Fériés » | public holidays from Monday to Friday |
| same | « Non renseignés » | missing days |
| same | « Total » | days from Monday to Friday |

#### Header and groups

- A first header row gives each group its name, `<th scope="colgroup">`: the short label of the counter, or « Jours » for the indicators.
- The names of the columns come below it.
- Each group is a `<colgroup>`, and a vertical line separates the groups.
- The quantities of the counters are shown with two decimals, the indicators in days.

#### Rows

- The first row, « Report N-1 », gives the balance of each counter on the eve of 1 January, or of the tracking start when it is later, before its adjustments.
- The zeroing of 1 January and the initial balance therefore count in the « Ajustement » of January.
- Each row satisfies:

  ```text
  Solde of the previous row - Débit + Crédit + Ajustement = Solde
  ```

- The last row, « Total » (`<tfoot>`), sums the Débit, Crédit, Ajustement and indicators; its Solde is the balance of the last row.
- The current month carries `aria-current="date"`.

#### Shown counters

| Place | Counters shown |
| --- | --- |
| Dashboard tiles | enabled, or with a non-zero balance today |
| Statements of the month page | enabled, or with entries or cancelled entries in the month |
| Yearly summary and counter sections of the year page | enabled, or with a movement in the year |

The counters come in the order of the configuration.

#### Status of a month in the summary

| Status | Months | Figures |
| --- | --- | --- |
| « validé » | validated months | indicators read from the frozen snapshot |
| « provisoire » | past open months and the current month | computed |
| « prévisionnel » | future months | the forecast of the journal: expected credits, planned leave, roundings and zeroings, consistent with the forecast of the dashboard |
| « hors suivi » | months before the tracking start | none |

### Counter sections

- One section per counter shown, titled by its label.
- A line gives its rule with the effective amount, for instance « 2,08 j par mois, période au 01/06, solde reporté et arrondi à l'entier supérieur »; a counter without carry over ends with « solde remis à zéro ».
- A recap table « Récapitulatif mensuel, \<compteur> » gives one row per tracked month (the months before the tracking start are skipped).
- A collapsed « Grand livre de l'année (\<n> écriture(s)) » lists the active entries of the year, with the columns of a statement. The cancelled entries are not listed there.

| Column of the recap | Content |
| --- | --- |
| « Mois » | the month |
| « État » | its status in the summary |
| « Débit » | the leave taken |
| « Crédit » | the accruals and the credits of feeding codes only, so that adjustments never blur the credits of the rules |
| « Ajustement » | the adjustments, initial balances, roundings and zeroings included, signed: `+0,31`, `-0,24`, `0,00` without sign |
| « Solde » | the balance at the end of the month |

## PROFILE PAGE

- Route `/dashboard/profile`, entry « Mon profil » of the user menu; `h1` « Mon profil ».

### Identity

The section « Identité » lists:

- « Identifiant »: the login;
- « Nom »: the name, or « non renseigné »;
- « Profil »: « Administrateur » or « Utilisateur »;
- « Compte créé le »: the creation date;
- « Dernière connexion »: the date of the last login, or « jamais »;
- « Sessions ouvertes »: the number of open sessions.

### Sessions

- The button « Déconnecter toutes mes sessions » (`action=sessions`) closes every session of the account, the current one included.
- The page « Vous êtes déconnecté » follows.

### Name form

| Rule | Value |
| --- | --- |
| Field | « Nom et prénom » |
| Length | 60 characters at most |
| White space | every run of white space, tabs and line breaks included, is collapsed to one space |
| Empty | allowed: the name becomes « non renseigné » |
| Request | `action=name` |
| Success | 303 to `/dashboard/profile`, notice `name` |

### Password change

- The form « Changer de mot de passe » asks for « Mot de passe actuel », « Nouveau mot de passe » and « Confirmation ».
- Its legend ends with « (obligatoire) » while the password is provisional ([Provisional password](#provisional-password)).
- The current password is checked under the login throttle.
- The new password follows the policy of [SECURITY.md](SECURITY.md#passwords) and differs from the current one.
- Success: every session of the account is closed, a new one opens for the current browser, and the page comes back (303) with the notice `password`.

### Refusals of the profile page

| Case | Status | Message |
| --- | --- | --- |
| unknown `action` | 400 | page « Requête invalide » |
| invalid name | 422 | « Le nom compte au plus 60 caractères, sur une ligne. » |
| too many failed checks from the same address, or on the same login | 429, with `Retry-After` | page « Trop de tentatives » |
| wrong current password | 422 | « Le mot de passe actuel est incorrect. » |
| the two new passwords differ | 422 | « Les deux saisies du nouveau mot de passe diffèrent. » |
| new password equal to the current one or against the policy | 422 | « Le nouveau mot de passe doit différer de l'actuel et compter au moins 12 caractères dont 6 distincts, sans espace au début ni à la fin, et différer de l'identifiant. » |
| the password changed while the current one was being checked | 409 | « Le mot de passe a changé entre-temps : il n'a pas été modifié. » |

## SETTINGS PAGE

- Route `/dashboard/settings`, entry « Mes réglages » of the user menu; `h1` « Réglages ».
- The rules of the settings are in [User settings](#user-settings).

The section « Suivi et compteurs » holds one form:

1. the fieldset « Début du suivi » with the field « Date de début », disabled once a month is validated;
2. the fieldset « Jours fériés » with the list « Calendrier des jours fériés »: first « Par défaut (\<zone>) », naming the default zone of the configuration, then the thirteen zones in the order of [Public holidays](#public-holidays), and the sentence « Un changement s'applique aux mois non validés. »;
3. the table of the counters, one row per counter that is not archived, with the columns « Compteur », « Activé », « Montant propre », « Solde initial » and, last, « Couleur »;
4. the checkbox « Rétablir les couleurs par défaut »;
5. the button « Enregistrer les réglages ».

- The field « Montant propre » shows the default amount and the rhythm as a placeholder.
- The column « Couleur » holds a colour field for each code that feeds or consumes the counter and is not archived, named « Couleur, \<code> : \<libellé> ».
- The column « Solde initial » is disabled once a month is validated.

### Refusals of the settings page

| Case | Status | Message |
| --- | --- | --- |
| unreadable revision of the configuration or of the account | 400 | page « Requête invalide » |
| the configuration changed since the form was shown | 409 | « La configuration a changé entre-temps. Vérifiez puis recommencez. » |
| the account changed since the form was shown | 409 | « Les données ont changé entre-temps. Vérifiez puis recommencez. » |
| invalid values | 422 | « Réglages non enregistrés. », followed by one line per problem |
| the account would grow beyond its size limits | 413 | « Les données de ce compte atteignent la taille maximale autorisée. » |

Both revisions are checked before the values: a stale form gets 409 even with invalid values.

The problems listed under « Réglages non enregistrés. » are:

- « Date de début de suivi invalide. »;
- « Des ajustements sont datés avant cette date de début : supprimez-les d'abord. »;
- « Calendrier des jours fériés invalide. »;
- « \<compteur> : montant d'acquisition invalide. »;
- « \<compteur> : solde initial invalide (entre -366 et 366). »;
- « \<libellé du code> : couleur invalide. ».

On a 422, the page keeps the values posted; a refused value is shown as stored (the tracking start, an initial balance), or empty (an own amount, a zone shown « Par défaut »), or as the default colour.

## DATA PAGE

- Route `/dashboard/data`, entry « Mes données » of the user menu, after « Mes réglages »; `h1` « Données ».
- The page imports a file, exports the data of the account, both in the [import format](#file-format), and erases them; each in a section, in this order.

- The button that chooses the file has the look of the other buttons.
- The section « Importer des données » posts `multipart/form-data` to `POST /dashboard/data/import`: the file « Fichier d'import », the checkboxes « Remplacer les jours déjà saisis » (`replace`) and « Valider les mois importés » (`validate`), and two buttons, « Vérifier » (`action=check`) and « Importer » (`action=import`).
- The import follows the rules of the command `import` ([Import](#import)): the checkboxes stand for `--replace` and `--validate`, « Vérifier » for `--dry-run`. There is no `--from` nor `--to`: the whole file is read.
- It works while the server runs, on the account of the viewer only, with the account revision of the page.
- The file is bounded by `server.max_body_bytes` (256 KiB by default), as every request body; a larger file is imported with the command `import`.
- « Vérifier » writes nothing and shows the report at the top of the page ([Main content](#main-content)): the notice « Vérification réussie : le fichier peut être importé. Rien n'a été enregistré. » with its details, after the warnings if any.
- « Importer » saves the import and comes back (303) with the notice `imported`; on a problem nothing is written, and the alert lists the problems at the top of the page.
- The section « Exporter mes données » explains what the file holds; its button « Exporter » sends `GET /dashboard/data/export` ([Export](#export)).
- The section « Supprimer mes données » comes last ([Deletion of the data](#deletion-of-the-data)).

### Report of an import

The report is spread over the three levels of messages:

| Level | Message | Details |
| --- | --- | --- |
| alert (red) | « L'import est refusé : rien n'a été enregistré. Corrigez le fichier puis recommencez. » | the problems, in the French texts of [Errors](#errors), each with its date written `jj/mm/aaaa` |
| warning (yellow) | « Points à vérifier : » | the warnings « Date de début avancée du \<date> au \<date> » and « Ajustement du \<date> déjà présent, ignoré »; month by month, « Demi-journées non renseignées en \<mois> : \<dates>. »; « Validation arrêtée en \<mois>. \<raison> » when a month cannot be validated, the reason being « Ce mois n'est pas dans le fichier. » or the reason of [Refusals of the activity page](#refusals-of-the-activity-page) |
| notice (green), « Vérifier » only | « Vérification réussie : le fichier peut être importé. Rien n'a été enregistré. » | « Réglages lus : date de début au \<date>, \<n> compteur(s). » when the file has `settings`; « \<n> jour(s) lu(s), de \<mois> à \<mois>. » when it has days; « \<n> ajustement(s) lu(s). » when it has adjustments; with « Valider les mois importés », « Mois validés : de \<mois> à \<mois> (\<n>). » |

The missing half-days and the validation are computed only when there is no problem.

### Deletion of the data

- The section « Supprimer mes données » explains what is erased and asks « Retapez votre identifiant pour confirmer » (`confirm`); its button « Supprimer mes données » is red (class `danger`).
- It posts to `POST /dashboard/data/reset`, on the account of the viewer only.
- The confirmation must be the login exactly, case included.
- It erases the whole tracking of the account: day entries, adjustments, history of the months, ledger entries (validated months included), settings (zone of public holidays, counters, own amounts, initial balances, colours), and the [conversations](#conversations) with the assistant.
- The account starts again like a new one: tracking start on the first day of the current month in `server.timezone`, zone of public holidays, every counter and colour by default.
- The identity, the name, the profile, the password, the sessions and the [subscriptions](#subscriptions), with their shares, are kept.
- It is the only operation that removes validated entries ([Invariants](#invariants)).
- Success: 303 to `/dashboard/data`, notice `erased`.

### Refusals of the data page

| Case | Status | Message |
| --- | --- | --- |
| `action` other than `check` or `import`, or unreadable revision | 400 | page « Requête invalide » |
| body beyond `server.max_body_bytes` | 400 | page « Requête invalide » |
| no file | 422 | « Choisissez un fichier à importer. » |
| file that cannot be decoded (not JSON, unknown field, wrong `format` or `version`, a malformed value such as a quantity or a date, beyond 8 MiB) | 422 | « Le fichier n'est pas un import valide : un objet JSON au format timesheet-import, version 1, sans champ inconnu. » |
| problems of the import | 422 | « L'import est refusé : rien n'a été enregistré. Corrigez le fichier puis recommencez. », with the report |
| deletion: confirmation other than the login | 422 | « Retapez exactement votre identifiant pour confirmer la suppression de vos données. » |
| the data changed since the page was shown, checked before the file and the confirmation | 409 | « Les données ont changé entre-temps (autre onglet ?). Rechargez la page avant de recommencer. » |
| the account would grow beyond its [size limits](#size-of-an-account) | 413 | « Les données de ce compte atteignent la taille maximale autorisée. » |

## AI ASSISTANT

The assistant answers the questions of a user about their own data, in French, with an AI model of a provider chosen by the user, and proposes writes that the user confirms or cancels. It is enabled for an account as soon as the account can use a subscription with a key.

### AI providers

- The providers live in the configuration, `configuration.ai_providers`, seeded by `init` from the [configuration template](#configuration-template). The template suggests four:

| `name` | Label | `format` | `base_url` | `default_model` | Suggested models |
| --- | --- | --- | --- | --- | --- |
| `mistral` | « Mistral » | `openai` | `https://api.mistral.ai/v1` | `mistral-small-latest` | `mistral-large-latest`, `mistral-medium-latest`, `mistral-small-latest`, `magistral-medium-latest`, `open-mistral-nemo`, `codestral-latest` |
| `anthropic` | « Anthropic (Claude) » | `anthropic` | `https://api.anthropic.com/v1` | `claude-sonnet-5-5` | `claude-opus-5-5`, `claude-sonnet-5-5`, `claude-haiku-5-5`, `claude-sonnet-4-5`, `claude-haiku-4-5` |
| `openai` | « OpenAI » | `openai` | `https://api.openai.com/v1` | `gpt-4o-mini` | `gpt-4.1`, `gpt-4.1-mini`, `gpt-4o`, `gpt-4o-mini`, `o4-mini`, `o3` |
| `google` | « Google Gemini » | `gemini` | `https://generativelanguage.googleapis.com/v1beta` | `gemini-2.5-flash` | `gemini-2.5-pro`, `gemini-2.5-flash`, `gemini-2.0-flash` |

- A configuration whose `ai_providers` is `null` or empty, such as a database created before the assistant, offers these four.
- The format names the API: `openai` posts to `<base_url>/chat/completions` with `Authorization: Bearer <key>`; `anthropic` posts to `<base_url>/messages` with `x-api-key` and `anthropic-version: 2023-06-01`; `gemini` posts to `<base_url>/models/<model>:generateContent` with `x-goog-api-key`.
- The server calls only the `base_url` of a provider of the configuration: an https URL without user, query, fragment or final slash. A user never types an address.
- The configuration page shows the providers read-only ([Configuration page](#configuration-page)); they are changed in the database, the server stopped.

### Subscriptions

- A subscription is an API key of a provider with a model and a name, owned by one account. It is managed in the page « Mes souscriptions » (`/dashboard/subscriptions`, `h1` « Mes souscriptions »), the entry of the user menu just below « Mon profil »: the section « Souscriptions » lists those of the account, « Partagées avec moi » those shared with it, and « Ajouter une souscription » holds the form of a new one.
- « Ajouter une souscription » asks for « Modèle », a list of the suggested models grouped by provider, « Autre modèle », which replaces the chosen model at the same provider, « Clef d'API » and « Nom ». Without a name, the subscription is named after the label of the provider and the model, cut at 60 characters.
- The key is sealed with AES-256-GCM by the key of the [key file](#key-file) and never shown again, to anyone: the table lists the name, the provider, the model and the shares.
- « Gérer » opens, for each subscription, the forms to change its name, model or key (an empty « Nouvelle clef d'API » keeps the key), to share it, to withdraw a share, to make it the default (« Utiliser par défaut ») and to delete it.
- The first subscription of the list is the default one of a new conversation.
- An account owns 20 subscriptions at most.
- Its forms post to `/dashboard/subscriptions` with an `action` starting with `subscription-`.
- Success: 303 to `/dashboard/subscriptions` with the notice `subscription`, `subscription-deleted`, `shared`, `unshared`, `subscription-hidden` or `subscription-shown`.

### Sharing

- The owner shares a subscription from « Gérer », field « Identifiant du compte », with another existing account, 50 accounts at most per subscription; « Retirer le partage avec \<login> » withdraws a share. An unknown login is refused.
- With JavaScript, the field suggests the logins as they are typed, like a search: from 2 characters, 300 ms after the last key, the page asks `GET /dashboard/subscriptions/logins?q=<text>` and shows at most 10 logins of other enabled accounts that contain the text, in alphabetical order, as buttons that fill the field (arrows to move, Escape to close). The answer is a JSON array of logins only: no name, no profile. Without JavaScript, the login is typed whole. Every connected user can thus learn the logins of the other accounts ([SECURITY.md](SECURITY.md#ai-assistant)).
- The account it is shared with sees it in the table « Souscriptions partagées avec moi », with its name, its owner, its provider and its model, never its key. It can use it in the assistant, and hide it (« Ne plus proposer ») or offer it again (« Proposer de nouveau »); it cannot change, share or delete it.
- A subscription of a disabled owner cannot be used.
- The conversations of a shared user stay on their own account: the owner never sees them.
- Withdrawing a share, deleting the subscription or deleting the account of either side ends the use at once: the conversations that used it can be read, not continued.

### Refusals of the subscriptions page

| Case | Status | Message |
| --- | --- | --- |
| unknown `action` | 400 | page « Requête invalide » |
| subscription unknown, or not owned by the account for a change | 404 | page « Page introuvable » |
| model unknown, or « Autre modèle » not 1 to 100 printable characters without space | 422 | « Choisissez un fournisseur et un modèle de la liste, ou un autre modèle de 1 à 100 caractères imprimables sans espace. » |
| subscription name over 60 characters or on several lines | 422 | « Le nom compte au plus 60 caractères, sur une ligne. » |
| API key missing at the creation, over 512 characters or with a control character | 422 | « Indiquez la clef d'API : une ligne de 512 caractères au plus. » |
| 20 subscriptions already | 422 | « Vous avez atteint le nombre maximal de souscriptions. » |
| share with an unknown login, one's own login or a login that already has access | 422 | « Partage impossible : indiquez l'identifiant d'un autre compte existant, qui n'a pas déjà accès à cette souscription. » |
| 50 shares already | 422 | « Cette souscription est déjà partagée avec le nombre maximal de comptes. » |
| the subscriptions changed in the meantime | 409 | « Les données ont changé entre-temps (autre onglet ?). Rechargez la page avant de recommencer. » |

### Conversations

- The page `/dashboard/assistant` (`h1` « Assistant ») offers the form « Nouvelle conversation » when the account can use a subscription, with the list « Souscription » when it can use several, and the table of the conversations, the most recent first: « Conversation » (its title, a link to it), « Dernier échange » and « Actions », whose red button « Supprimer » deletes that conversation (`action=delete`, notice `conversation-deleted`); below it, the button « Supprimer toutes les conversations ». Without usable subscription, an alert points to « Mes souscriptions ».
- The page `/dashboard/assistant/{id}` (`h1` the title of the conversation) lists the exchanges: each question with its time, then the answer, the model that gave it, or the reason of its failure. The form « Poursuivre » asks the next question with the same subscription; « Supprimer cette conversation » deletes it.
- The title of a conversation is its first question on one line, cut at a word before 60 characters with « … ».
- An account keeps `assistant.conversations` conversations, 50 by default, of `assistant.exchanges` exchanges, 20 by default: beyond, the oldest conversation, or the oldest exchange of the conversation, is dropped. When the account record would exceed `limits.account_bytes`, the oldest conversations are dropped too.
- The conversation of another account is not found (404).

### Questions and answers

- A question is 1 to `assistant.question_bytes` bytes, 4000 by default, of text following the [free-text rule](#free-text-rule) on each line; line breaks are kept.
- `action=ask` records the question, redirects (303) to the conversation and starts the answer in the background: the page reloads itself every 3 seconds (`<meta http-equiv="refresh">`) and shows « Réponse en cours de préparation… » until the answer arrives.
- One answer at a time is prepared per account: a question asked meanwhile is refused (409).
- The answer is the Markdown text of the model, its control characters removed and cut at 64 000 bytes.
- An answer interrupted by a stop of the server shows « La réponse a été interrompue : posez la question à nouveau. »; a failure of the provider shows its reason ([Refusals of the assistant pages](#refusals-of-the-assistant-pages)), never the body of its answer.
- Without JavaScript, the forms work as is; with it, Enter sends the question, Shift and Enter adds a line, and the button is disabled after sending.

### Agent and skill

- The system prompt is the skill, a Markdown file embedded in the program (`src/internal/assistant/assistant.skill.md`), followed by a context: today, the login and the name, the tracking start, the planning horizon and the activity codes. The skill describes the application, its vocabulary, the tools and how to answer: in French, with Markdown tables, without inventing a figure.
- The history sent is the last `assistant.exchanges` exchanges of the conversation, questions and answers only.
- The agent calls the model, runs the tools it asks for, and calls it again with their results, `assistant.max_rounds` times at most, 4 by default; a last call asks for an answer without tool.
- Each call to the provider lasts `assistant.timeout` at most, 1 minute by default; it asks for `assistant.max_tokens` tokens at most, 4096 by default; the answer of the provider is read up to 4 MiB; a redirection is not followed.

### Tools

Every tool works on the account of the question, whatever its arguments; it reads the data of the moment of the call.

| Tool | Arguments | Gives |
| --- | --- | --- |
| `read_balances` | none | the [positions](#counter-tiles) of the counters |
| `read_month` | `month` (`YYYY-MM`) | state, editability, half-days and comments, adjustments, anomalies and indicators of the month |
| `read_statement` | `counter`, `month` | the [statement](#counter-statements) of the counter for the month |
| `read_year` | `year` | the yearly summary, month by month: debits, credits, adjustments and balances of each counter, working days, public holidays, missing days |
| `read_configuration` | none | the activity codes, the counters and the zone of public holidays |
| `read_settings` | none | the user settings and every adjustment with its identifier |
| `propose_days` | `month`, `days` (`date`, `am`, `pm`, `comment`) | a proposal to enter, change or clear half-days of an open month |
| `propose_validation` | `month` | a proposal to [validate](#validation) the month |
| `propose_reopening` | `month`, `reason` | a proposal to [reopen](#reopening) the last validated month |
| `propose_adjustment` | `date`, `counter`, `amount`, `reason` | a proposal of an [adjustment](#adjustments) |
| `propose_adjustment_removal` | `id` | a proposal to delete an adjustment of an open month |
| `export_data` | none | the [export](#export) of the account saved as a [temporary download](#temporary-downloads): its link, its file name and the time it expires |

A result is JSON of 64 000 bytes at most. The tools never reach the profile, the password, the settings, the subscriptions or the administration.

### Actions to confirm

- A `propose_` tool writes nothing. It checks the change with the rules of the pages (editable month, [codes offered](#codes-offered-in-a-grid), conditions of a validation or a reopening, bounds of an adjustment); a refusal goes back to the model, which explains it.
- An accepted proposal becomes the pending action of the conversation, with its summary in French and the revision of the account; a new proposal replaces it.
- The page shows it in the section « Action à confirmer », with « Confirmer » and « Annuler ».
- « Confirmer » (`action=confirm`) checks the revision, checks the change again on the current data and writes it like the page would: the event of the write is logged with `by=assistant`, the notice is that of the page (`saved`, `validated`, `reopened`, `adjusted` or `unadjusted`), and the conversation records « Je confirme l'action proposée. » with the outcome.
- « Annuler » (`action=cancel`) drops it; the conversation records the cancellation; notice `cancelled`.
- A validated month is never rewritten: a proposal on it is refused, and a confirmation after the data changed is refused (409).

### Temporary downloads

- `export_data` encodes the export of the account, the file of « Mes données » in the import format ([Export](#export)), and keeps it in the [directory of the downloads](#directory-of-the-downloads) under a random identifier of 32 hexadecimal characters.
- Its link is `/dashboard/assistant/downloads/{id}`; the model gives it in its answer, where it is the only relative link kept.
- The link works only for the connected account that asked for it: for any other account, as for an unknown identifier, it gives 404.
- It answers the file as an attachment named `timesheet-<login>-<date>.json`, the date of its creation, with `Cache-Control: no-store`.
- A file lives `assistant.download_ttl`, 30 minutes by default: after it, the link gives 410, page « Lien expiré », and the file is removed.
- Every `assistant.download_sweep`, 5 minutes by default, and at the start of the server, a job removes the expired files and the temporary files left by an interrupted write.
- An account keeps 20 files at most: a new one removes the oldest.
- « Supprimer mes données » and the deletion of the account remove its files at once.

### Rendering of the answers

- The answer is parsed on the server into paragraphs, headings, lists, tables, code blocks, quotes and rules, with bold, italic, code and links inline; everything is escaped by the templates, and raw HTML is shown as text.
- The headings start at `h4` under the `h3` of the exchange and never skip a level.
- A table gets the caption « Tableau \<n> » and headers `th scope="col"`.
- Only `http` and `https` links with a host are kept, with `rel="noopener noreferrer"`, and the links `/dashboard/assistant/downloads/<32 hexadecimal characters>` of the temporary downloads, with the attribute `download`; another link is shown as its text.

### Refusals of the assistant pages

| Case | Status | Message |
| --- | --- | --- |
| unknown `action` | 400 | page « Requête invalide » |
| unknown conversation, or no pending action to confirm or cancel | 404 | page « Page introuvable » |
| download unknown, malformed or of another account | 404 | page « Page introuvable » |
| download older than `assistant.download_ttl` | 410 | page « Lien expiré », « Ce fichier n'est plus disponible : demandez un nouvel export à l'assistant. » |
| empty question | 422 | « Écrivez une question. » |
| question too long or with a forbidden character | 422 | « La question est trop longue ou contient des caractères non autorisés. » |
| no usable subscription | 409 | « Aucune souscription IA utilisable : ajoutez-en une dans « Mes souscriptions ». » |
| the subscription of the conversation is no longer usable | 409 | « La souscription de cette conversation n'est plus utilisable : elle a été supprimée, n'est plus partagée ou sa clef est illisible. » |
| an answer is already being prepared | 409 | « Une réponse est déjà en cours de préparation : attendez-la avant de poser une autre question. » |
| the conversations changed in the meantime | 409 | « Les données ont changé entre-temps (autre onglet ?). Rechargez la page avant de recommencer. » |
| the data changed since the proposal | 409 | « L'action ne correspond plus à l'état de vos données : elle n'a pas été appliquée. Annulez-la et demandez une nouvelle proposition. » |
| the proposal is now refused | 409 | « L'action n'a pas été appliquée : » and the refusal |
| the account would exceed its size | 413 | « Les données de ce compte atteignent la taille maximale autorisée. » |

Failures of the provider, shown in place of the answer:

| Case | Message |
| --- | --- |
| status 401 or 403 | « Le fournisseur a refusé la clef d'API (statut \<n>) : vérifiez la souscription dans « Mes souscriptions ». » |
| status 429 | « Le quota ou le débit autorisé par le fournisseur est dépassé (statut 429) : réessayez plus tard. » |
| other status | « Le fournisseur a refusé la requête (statut \<n>) : vérifiez le modèle de la souscription. » |
| `assistant.timeout` reached | « Le fournisseur n'a pas répondu à temps. » |
| network error, unreadable answer | « Le fournisseur est injoignable ou sa réponse est illisible. » |
| provider removed from the configuration | « Le fournisseur de cette souscription n'existe plus dans la configuration. » |

## ADMINISTRATION

### Server page

Route `/admin/`, entry « Serveur » of the administration navigation, right after « Tableau de bord », and target of the entry « Administration » of the header; `h1` « Serveur ».

#### Software

The first section, « Logiciel », comes before every server setting, with the same presentation:

| Fact | Value |
| --- | --- |
| « Nom » | `Timesheet` |
| « Version » | the release number written in the program (`ReleaseVersion` of `BuildInfo`), the same as the first entry of `debian/changelog` |
| « Licence » | `GNU GPL version 2 ou ultérieure` |

No other page shows the version.

#### Server settings shown

- After it, the page shows, read-only, every setting of the [server configuration](#server-configuration), one section with its `h2` and its list of facts per group:

| Group | Facts |
| --- | --- |
| « Réseau » | « Adresse d'écoute », « Origines publiques », « Proxys de confiance », « Fuseau horaire », « Niveau du journal », « Certificat TLS », « Clé TLS », « Version TLS minimale », « Durée HSTS » |
| « Requêtes » | « Lecture des en-têtes », « Lecture de la requête », « Écriture de la réponse », « Connexion inactive », « Arrêt du serveur », « Taille des en-têtes », « Taille du corps », « Longueur de l'adresse » |
| « Base de données » | « Fichier », « Taille maximale » |
| « Sessions » | « Expiration sans activité », « Durée maximale », « Sessions par compte », « Enregistrement des sessions » |
| « Connexion » | « Validité du formulaire », « Premier délai », « Délai maximal », « Fenêtre du budget », « Nouvelles adresses par fenêtre », « Adresses suivies », « Oubli d'une adresse » |
| « Mots de passe » | « Itérations PBKDF2 » |
| « Limites » | « Comptes », « Taille d'un compte », « Ajustements par compte » |
| « Assistant » | « Délai d'un appel au fournisseur », « Tours d'outils par question », « Jetons par réponse », « Conversations par compte », « Échanges par conversation », « Taille d'une question », « Durée d'un lien de téléchargement », « Nettoyage des téléchargements »; the values in effect, after the [environment variables](#environment-variables) |

- A duration is shown with its French unit: `8 h`, `1 min`, `30 s`, `500 ms`, in the shortest form of the file. A size is shown in `Mio` or `Kio` when it is a whole number of them, otherwise in `octets`; the length of the address is always in `octets`.
- A list (« Origines publiques », « Proxys de confiance ») shows one entry per line, each in its own `dd` under the same `dt`.
- An empty list of proxies and a missing certificate pair are shown « aucun ».
- They come from the [server configuration](#server-configuration) read at the start of the server, or from the defaults without `--config`, not from the options of `run`, except « Fichier », which is the database the server opened, from `--database` or `database.path`.
- The line under the `h1` explains how to change them: edit the file or use `timesheet config set`, then restart the server; the options of `timesheet run`, `--database` included, take precedence.

### Accounts page

Route `/admin/accounts`, entry « Comptes » of the administration navigation; `h1` « Comptes ».

#### Table of the accounts

The section « Comptes existants » holds one table, one row per account, with:

- the login, the name, the profile;
- the state, « actif » or « désactivé »; a disabled account's row is dimmed;
- the creation date, the last login and the number of open sessions;
- links to the read-only views of the account;
- the actions, side by side, as icon buttons.

#### Actions of a row

| Action | `title` | `aria-label` | Icon | Disabled when | Effect |
| --- | --- | --- | --- | --- | --- |
| Close the sessions | « Fermer les sessions » | « Fermer les sessions de \<login> » | an arrow leaving a door | the account has no open session | closes every session of the account; on one's own row the current one too, and the page « Vous êtes déconnecté » follows |
| Disable | « Désactiver » | « Désactiver \<login> » | a barred circle | one's own row | disables the account |
| Enable | « Activer » | « Activer \<login> » | a ticked circle | never | enables a disabled account |
| Delete | « Supprimer » | « Supprimer \<login> » | a bin | one's own row | opens the deletion box |

- The icons are inline SVG drawn in the text colour.
- The actions come back to `/admin/accounts` (field `return=list`), with the notice on success and the alert on the same page when they fail.

#### Deletion box

1. « Supprimer » is the `summary` of a `<details class="account-delete">`.
2. Its panel is a modal box: a `<dialog open>` named « Supprimer le compte \<login> », holding the form.
3. The box is fixed in the middle of the window, over a dark backdrop that covers the page, so that the scrolling of the table never clips it.
4. It recalls the account: « Supprimer le compte \<login> et toutes ses données ? ».
5. It asks to retype the login in the field « Retapez « \<login> » », which receives the focus.
6. It offers, side by side, « Supprimer » and, at its right, « Annuler ».
7. « Annuler » is a link back to `/admin/accounts`, so that it works without JavaScript.
8. A click on the backdrop or `Échap` closes the box too.

#### Form « Nouveau compte »

The section « Créer un compte » holds the form « Nouveau compte », posted to `POST /admin/accounts`.

| Field | Rule |
| --- | --- |
| « Identifiant » | a [login](#logins) not yet taken |
| « Profil » | « Administrateur » then « Utilisateur », the latter selected by default |
| « Mot de passe provisoire » | follows the password policy; the account must change it at its first login |

The button « Créer le compte » creates the account, `enabled`, with its tracking start on the first day of the current month, and comes back to `/admin/accounts` with the notice `created`.

#### Refusals of the accounts page

| Case | Status | Message |
| --- | --- | --- |
| invalid login | 422 | « Identifiant invalide : 2 à 32 caractères parmi a-z, 0-9, point, tiret et souligné, commençant par une lettre ou un chiffre. » |
| invalid profile | 422 | « Profil invalide. » |
| provisional password against the policy | 422 | « Mot de passe provisoire trop faible : au moins 12 caractères dont 6 distincts, sans espace au début ni à la fin, différent de l'identifiant. » |
| login already taken | 422 | « Cet identifiant est déjà pris. » |
| `limits.accounts` accounts reached, 200 by default | 422 | « Le nombre maximal de comptes est atteint. » |
| other failure | 422 | « Création impossible. » |

The refusals of the row actions are those of the [account page](#refusals-of-the-account-page), shown on `/admin/accounts`.

### Account page

Route `/admin/accounts/{login}`; `h1` « Compte \<login> »; an unknown login gives 404.

The section « Informations » gives the name (or « non renseigné »), the profile, the state, the creation date, the last login and the number of open sessions. The forms follow.

| Section | Form | Fields | `action` | Success | On one's own account |
| --- | --- | --- | --- | --- | --- |
| « Nom » | « Changer le nom » | « Nom et prénom », same rules as in the [profile page](#name-form) | `name` | 303 back to the account page, notice `name` | shown |
| « Profil » | « Changer le profil » | « Profil »: « Utilisateur » or « Administrateur » | `profile` | closes every session of the account; notice `profile` | hidden |
| « Accès » | « Désactiver le compte » or « Activer le compte » | none | `disable` or `enable` | disabling closes every session of the account; notice `disabled` or `enabled` | hidden |
| « Accès » | « Fermer les sessions », button « Fermer toutes ses sessions », disabled without open session | none | `sessions` | closes every session of the account without preventing a new login; notice `sessions` | hidden |
| « Mot de passe » | « Réinitialiser le mot de passe » | « Mot de passe provisoire » | `reset` | closes every session; the account must change the password; notice `reset` | hidden |
| « Suppression » | « Supprimer le compte et toutes ses données », button « Supprimer définitivement » | « Retapez l'identifiant pour confirmer » | `delete` | deletes the account and its data, closes its sessions; 303 to `/admin/accounts` with the notice `deleted` | hidden |

#### Refusals of the account page

| Action | Case | Status | Message |
| --- | --- | --- | --- |
| any | unknown login | 404 | page « Page introuvable » |
| any | the field `created` is missing or differs from the account: the account was deleted then created again since the form was shown | 409 | « Ce compte a été supprimé puis recréé entre-temps. Rechargez la page avant de recommencer. » |
| any | unknown `action` | 400 | page « Requête invalide » |
| `name` | invalid name | 422 | « Le nom compte au plus 60 caractères, sur une ligne. » |
| `profile` | invalid profile, or one's own profile | 422 | « Profil invalide, ou changement de son propre profil. » |
| `profile` | would demote the last enabled `ADMIN` | 409 | « Le dernier administrateur actif ne peut être ni supprimé, ni rétrogradé, ni désactivé. » |
| `reset` | one's own account | 422 | « Changez votre propre mot de passe depuis votre profil. » |
| `reset` | provisional password against the policy | 422 | « Mot de passe provisoire trop faible : au moins 12 caractères dont 6 distincts, sans espace au début ni à la fin, différent de l'identifiant. » |
| `disable`, `enable` | one's own account | 422 | « Un administrateur ne peut pas désactiver son propre compte. » |
| `disable` | the last enabled `ADMIN` | 409 | « Le dernier administrateur actif ne peut être ni supprimé, ni rétrogradé, ni désactivé. » |
| `delete` | wrong confirmation, or one's own account | 422 | « Retapez exactement l'identifiant pour confirmer ; un administrateur ne peut pas supprimer son propre compte. » |
| `delete` | the last enabled `ADMIN` | 409 | « Le dernier administrateur actif ne peut être ni supprimé, ni rétrogradé, ni désactivé. » |
| `name`, `profile`, `reset`, `disable`, `enable`, `delete` | any other failure | 409 | « Opération impossible. », or the page « Page introuvable » (404) when the account was deleted meanwhile |

### Read-only views of another account

| View | Title | Route |
| --- | --- | --- |
| Balances | « Soldes de \<login> » | `/admin/accounts/{login}/dashboard` |
| Month | « \<Mois> \<année> de \<login> » | `/admin/accounts/{login}/month/{year}/{month}` |
| Year | « Année \<année> de \<login> » | `/admin/accounts/{login}/year/{year}` |

- They show the balances, the calendar with the comments, the adjustments with their reasons, the statements, the summary and the ledger.
- They do not show the settings, and there is no activity page of another account.
- They hold no form that writes.
- Their navigation links stay under the prefix `/admin/accounts/{login}/`, and they belong to the admin family.
- They use the colours of the viewer.
- An unknown login gives 404.

### Configuration page

Route `/admin/settings`, entry « Configuration » of the administration navigation; `h1` « Configuration ».

#### Forms

- The section « Jours fériés » comes first: one form « Calendrier des jours fériés » with the list « Calendrier par défaut » of the thirteen zones and the button « Enregistrer »; a change applies to the open months of the accounts that follow the default zone.
- The section « Codes d'activité » holds one form per code, then the form « Nouveau code d'activité ».
- The section « Compteurs » holds one form per counter, then the form « Nouveau compteur ».
- The last section, « Fournisseurs d'IA », lists read-only the [AI providers](#ai-providers): label, format, address, default model and suggested models. No form changes them.
- Each form works without JavaScript and carries an `action` and the configuration revision.
- The legend of each form is the identifier, with « utilisé » or « archivé » when it applies.
- The button « Supprimer » is shown only for a code or a counter that is not used.
- The fields are listed in [Form fields](#form-fields-of-the-configuration-page).

#### Validation before writing

The whole configuration is checked before it is written:

1. a known default zone of public holidays, or none, which means `fr-metropole`;
2. 1 to 100 activity codes, at most 20 counters;
3. unique identifiers, in `[A-Z][A-Z0-9_]{0,15}`;
4. labels of 1 to 60 characters, short labels of 1 to 6, [free-text rule](#free-text-rule);
5. exactly one of `consumes` and `feeds` per code (« Effet »: « Alimente » or « Décompte »), naming an existing counter (« Compteur »);
6. a usable colour `#rrggbb` on every code;
7. an accrual rhythm « par mois » or « par an », and an amount between 0 and 366;
8. a period start `MM-DD`, 29 February excluded;
9. no change that would make the stored data invalid.

#### Changing, archiving and deleting

| Object | Change | Deletion | Archiving |
| --- | --- | --- | --- |
| Default zone of public holidays | any zone; it applies from the first open month of each account without its own zone | none | none |
| Activity code | label, colour, archived; `consumes` and `feeds` only while the code is not used | only while not used (not present in any day entry) | an archived code is no longer offered; stored entries keep it |
| Counter | label, short label, enabled by default, amount, rhythm, period start, carry over, archived; the new rule applies from the first open month | only while not used and while no code consumes or feeds it | [Disabled and archived counters](#disabled-and-archived-counters) |

- An identifier never changes.
- A code is used when it is present in some day entry.
- A counter is used when it appears in the settings, adjustments, ledger or month rules of some account. A settings entry exists for every counter as soon as an account has saved its settings page once.
- A used code or counter cannot be deleted, only archived.

#### Refusals of the configuration page

| Case | Status | Message |
| --- | --- | --- |
| unreadable revision or unknown `action` | 400 | page « Requête invalide » |
| unknown default zone of public holidays | 422 | « Calendrier des jours fériés inconnu. » |
| no effect chosen for a code | 422 | « Indiquez si le code alimente ou décompte son compteur. » |
| new code with an existing identifier | 422 | « Cet identifiant de code existe déjà. » |
| update or deletion of an unknown code | 422 | « Code inconnu. » |
| effect or counter changed on a used code | 422 | « L'effet et le compteur d'un code déjà utilisé ne sont plus modifiables. » |
| deletion of a used code | 422 | « Ce code est utilisé : archivez-le plutôt que de le supprimer. » |
| unreadable amount | 422 | « Montant d'acquisition invalide. » |
| unreadable period start | 422 | « Début de période invalide (MM-JJ). » |
| new counter with an existing identifier | 422 | « Cet identifiant de compteur existe déjà. » |
| update or deletion of an unknown counter | 422 | « Compteur inconnu. » |
| deletion of a used counter | 422 | « Ce compteur est utilisé : archivez-le plutôt que de le supprimer. » |
| deletion of a counter still fed or consumed by a code | 422 | « Un code d'activité alimente ou décompte encore ce compteur. » |
| the configuration changed since the form was shown | 409 | « La configuration a changé entre-temps. Vérifiez puis recommencez. » |
| the configuration fails the [validation before writing](#validation-before-writing) | 422 | « Configuration refusée : \<détail> », the details being listed after this table |
| the change would make stored data invalid, or changes the effect or the counter of a code that a day entered meanwhile uses (checked again when the configuration is written) | 409 | « Configuration refusée : elle rendrait invalides des données existantes. » |

The details that follow « Configuration refusée : » are:

- « entre 1 et 100 codes d'activité. »;
- « trop de compteurs (20 au plus). »;
- « identifiant invalide ou déjà utilisé (une majuscule, puis jusqu'à 15 majuscules, chiffres ou _). »;
- « libellé court invalide (1 à 6 caractères). »;
- « libellé invalide (1 à 60 caractères sur une ligne). »;
- « rythme d'acquisition invalide (mois ou année). »;
- « montant d'acquisition invalide (entre 0 et 366). »;
- « début de période invalide (MM-JJ). »;
- « un code doit alimenter ou décompter un compteur existant. »;
- « couleur invalide (#rrggbb). »;
- « valeur invalide. ».

The checks of the form itself come before the check of the revision: a form with an invalid value gets its 422 even when the configuration changed meanwhile. The validation of the whole configuration before writing (« Configuration refusée : … ») comes after it: a stale form gets 409 first.

## COLOURS

### Colour of a cell

| Cell | Colour |
| --- | --- |
| entered half-day | the colour of its code ([Codes of the template](#codes-of-the-template)) |
| missing half-day, before today | surface with a warning border (« Non renseigné ») |
| upcoming half-day, from today | surface (« À venir ») |
| untracked half-day, before the tracking start | surface (« Hors suivi ») |
| day that does not exist, in the year calendar (30 February, 31 April) | black `#000000`, in both themes |

### Configuration and user colours

| Level | Page | Rule |
| --- | --- | --- |
| Configuration | [configuration page](#configuration-page) | the colour of each code, mandatory |
| User preference | [settings page](#settings-page) | a colour per code, in the row of its counter; « Rétablir les couleurs par défaut » clears them |

- Any `#rrggbb` value is accepted and stored in lower case.
- No colour is refused for its contrast, since the text colour follows it.
- A user colour equal to the colour of the configuration is not stored.
- The user colour of an archived code, or of a code whose counter is archived, is kept; neither is shown in the settings page.
- A user colour for an unknown code is refused at load.
- The user colours of a deleted code are removed with it.

### Colour sheet

- `/dashboard/colors.css` serves the colours of the viewer as custom properties `--code-<id>`.
- Each has a text colour `--code-<id>-ink`, black `#000000` or white `#ffffff`, whichever contrasts most.
- Every colour reaches at least 4.58:1 with one of them, above [AA](#glossary).
- A rule `.code-<id>` per code applies both.
- A read-only view uses the colours of the viewer, not those of the account viewed.

### Colour is never the only information

The label of the code (grid) or its identifier (year calendar) is always written in the cell.

## ROUTES

| Route | Method | Access | Role |
| --- | --- | --- | --- |
| `/` | GET | public | home page ([Public pages](#public-pages)) |
| `/login` | GET | public | login form « Connexion »; a connected browser is redirected (303) to `/dashboard/` |
| `/login` | POST | public | login: 303 to `/dashboard/` on success, 401 « Identifiant ou mot de passe incorrect. » on failure, 429 « Trop de tentatives » with `Retry-After` while the address or the login is blocked |
| `/logout` | GET | connected | confirmation page; a guest is redirected (303) to `/login` |
| `/logout` | POST | connected | closes the session; page « Vous êtes déconnecté » |
| `/dashboard/` | GET | connected | [dashboard](#dashboard): balances today |
| `/dashboard/activity` | GET | connected | 303 to the activity page of the current month |
| `/dashboard/activity/{year}/{month}` | GET | connected | [activity page](#activity-page) of that month |
| `/dashboard/activity/{year}/{month}` | POST | connected | saves the grid of that month |
| `/dashboard/activity/{year}/{month}/validate` | POST | connected | validates the month ([Validation](#validation)) |
| `/dashboard/activity/{year}/{month}/reopen` | POST | connected | reopens the last validated month ([Reopening](#reopening)) |
| `/dashboard/month` | GET | connected | 303 to the month page of the current month |
| `/dashboard/month/{year}/{month}` | GET | connected | [month page](#month-page): totals, calendar, summary and statements |
| `/dashboard/year/{year}` | GET | connected | [year page](#year-page): totals, calendar, summary and counters |
| `/dashboard/adjustments` | POST | connected | adds (`action=add`) or deletes (`action=delete`, `id`) an adjustment; 303 to the activity page of its month; any other `action` gets 400 |
| `/dashboard/profile` | GET | connected | [profile page](#profile-page) |
| `/dashboard/data` | GET | connected | [data page](#data-page) |
| `/dashboard/data/export` | GET | connected | downloads the data of the account in the import format ([Export](#export)) |
| `/dashboard/data/import` | POST | connected | checks or imports a file in the import format ([Data page](#data-page)) |
| `/dashboard/data/reset` | POST | connected | erases the whole tracking of the account ([Deletion of the data](#deletion-of-the-data)) |
| `/dashboard/profile` | POST | connected | `action=name` saves the name; `action=sessions` closes every session of the account; `action=password` changes the password; any other `action` gets 400 |
| `/dashboard/subscriptions` | GET | connected | page « Mes souscriptions » ([Subscriptions](#subscriptions)) |
| `/dashboard/subscriptions` | POST | connected | `action=subscription-add`, `subscription-edit`, `subscription-delete`, `subscription-default`, `subscription-share`, `subscription-unshare`, `subscription-hide` or `subscription-show`; any other `action` gets 400 |
| `/dashboard/subscriptions/logins` | GET | connected | JSON array of at most 10 logins of other enabled accounts containing `q`, 2 to 32 characters; `[]` otherwise ([Sharing](#sharing)) |
| `/dashboard/assistant` | GET | connected | [assistant page](#conversations): new conversation and list of the conversations |
| `/dashboard/assistant` | POST | connected | `action=ask` asks a question, in the conversation `conversation` or in a new one; `action=confirm` or `cancel` applies or drops the pending action of `conversation`; `action=delete` deletes `conversation`; `action=delete-all` deletes every conversation; any other `action` gets 400 |
| `/dashboard/assistant/{id}` | GET | connected | [conversation page](#conversations); 404 for a conversation of another account |
| `/dashboard/assistant/downloads/{id}` | GET | connected | a [temporary download](#temporary-downloads) of the account; 404 for an unknown one or one of another account, 410 once expired |
| `/dashboard/settings` | GET | connected | [settings page](#settings-page) |
| `/dashboard/settings` | POST | connected | saves the user settings |
| `/dashboard/colors.css` | GET | connected | [colour sheet](#colour-sheet) of the viewer |
| `/admin/` | GET | ADMIN | [server page](#server-page) |
| `/admin/accounts` | GET | ADMIN | [accounts page](#accounts-page) |
| `/admin/accounts` | POST | ADMIN | creates an account |
| `/admin/accounts/{login}` | GET | ADMIN | [account page](#account-page) |
| `/admin/accounts/{login}` | POST | ADMIN | `action=name`, `profile`, `reset`, `disable`, `enable`, `sessions` or `delete`; any other `action` gets 400; with `return=list` the answer comes back to `/admin/accounts`; `sessions` on one's own account shows « Vous êtes déconnecté » |
| `/admin/accounts/{login}/dashboard` | GET | ADMIN | read-only balances of the account |
| `/admin/accounts/{login}/month/{year}/{month}` | GET | ADMIN | read-only month page of the account |
| `/admin/accounts/{login}/year/{year}` | GET | ADMIN | read-only year page of the account |
| `/admin/settings` | GET | ADMIN | [configuration page](#configuration-page) |
| `/admin/settings` | POST | ADMIN | changes the configuration: `action=public-holidays-update`, `activity-add`, `activity-update`, `activity-delete`, `counter-add`, `counter-update` or `counter-delete`; any other `action` gets 400 |
| `/static/{path...}` | GET | public | embedded CSS, JavaScript and SVG |

### Access

- Public: any profile, a guest included.
- Connected: `USER` and `ADMIN`.
- ADMIN: `ADMIN` only.
- `HEAD` is served like `GET`.
- `{year}` is 2000 to 2099 on four digits and `{month}` `01` to `12` on two digits; another value or form gives 404.

### Refusals

The refusals below come in the order in which the server checks them: a request refused by an earlier row never reaches a later one. The HTTP chain that applies them is described in [ARCHITECTURE.md](ARCHITECTURE.md#http-chain).

| Order | Status | Case | Answer |
| --- | --- | --- | --- |
| 1 | 431 | request headers beyond the limit (`server.max_header_bytes`, 8 KiB by default; the standard library accepts about 4 KiB more) | plain text from the standard library, without the security headers and without a log line |
| 1 | 400 | malformed request line or headers | plain text from the standard library, without the security headers and without a log line |
| 2 | 500 | a page fails unexpectedly, at any later step | page « Erreur interne », « Une erreur inattendue s'est produite. Réessayez plus tard. » |
| 3 | 421 | `Host` header outside the accepted origins | page « Hôte inconnu », « Ce serveur ne répond pas pour ce nom d'hôte. » |
| 4 | 414 | URL beyond `server.max_url_bytes` (2048 bytes by default) | page « Adresse trop longue », « L'adresse demandée dépasse la taille autorisée. » |
| 5 | 403 | unsafe request whose `Origin` is missing, `null` or not an accepted origin (another host or another scheme), or whose `Sec-Fetch-Site` is present and neither `same-origin` nor `none` | page « Accès refusé », « Cette requête vient d'une autre origine et a été refusée. » |
| 6 | 404 | unknown path | page « Page introuvable » |
| 6 | 405 | known path, another method (`OPTIONS` included), with the header `Allow` | page « Méthode non autorisée », « Cette adresse n'accepte pas cette méthode. » |
| 7 | 303 to `/login` | connected route without a valid session; a POST after the session expired is not saved | none |
| 8 | 403 | `USER` on a route reserved to `ADMIN` | page « Accès refusé », « Cette page est réservée aux administrateurs. » |
| 9 | 303 to `/dashboard/profile` | account with a provisional password, outside the exempt routes ([Provisional password](#provisional-password)) | none |
| 10 | 400 | form that cannot be read, a body beyond `server.max_body_bytes` (256 KiB by default) included | page « Requête invalide », « La requête ne peut pas être traitée. » |
| 11 | 403 | POST whose CSRF token does not match its session; login posted without the login cookie of its form or with another token (the cookie lasts `login.form_lifetime`, one hour by default, from the last display of the form in that browser; the browser then drops it) | page « Accès refusé », « Le formulaire a expiré ou n'est pas valide. Rechargez la page et recommencez. » |
| 12 | 429 | login, or check of the current password in the profile page, while the address or the login is blocked by the throttle; with `Retry-After` | page « Trop de tentatives », « Patientez quelques instants avant de réessayer. » |
| 13 | 401 | wrong login or password, disabled account | the login form with « Identifiant ou mot de passe incorrect. » |
| 14 | 404 | invalid year or month, unknown login under `/admin/accounts/` | page « Page introuvable » |
| 15 | 409 | form of an account whose field `created` is missing or differs from the account | the account page or the accounts page again, with « Ce compte a été supprimé puis recréé entre-temps. Rechargez la page avant de recommencer. » |
| 16 | 400 | unknown `action`, unreadable revision | page « Requête invalide » |
| 17 | 409, 413, 422 | refusals of a page | the page again, with an alert: [activity page](#refusals-of-the-activity-page), [settings page](#refusals-of-the-settings-page), [profile page](#refusals-of-the-profile-page), [subscriptions page](#refusals-of-the-subscriptions-page), [data page](#refusals-of-the-data-page), [assistant pages](#refusals-of-the-assistant-pages), [accounts page](#refusals-of-the-accounts-page), [account page](#refusals-of-the-account-page), [configuration page](#refusals-of-the-configuration-page) |
| 18 | 500 | write error, the database file beyond `database.max_bytes` (64 MiB by default) included | page « Erreur interne » |

Every error page offers « Revenir à l'accueil ». The answers of rows 2 to 18 carry the security headers ([SECURITY.md](SECURITY.md#pages)) and a line in the access log ([Logs](#logs)).

### After a successful POST

- The server redirects (303) to a fixed internal path, with a `notice` code chosen from a fixed table ([Notices](#notices)).
- The login redirects to `/dashboard/` without a notice.
- `POST /logout`, `action=sessions` of the profile page and `action=sessions` on one's own account answer 200 with the page « Vous êtes déconnecté ».

## NOTICES

A successful action that redirects (303) keeps the code of its notice in the session of the browser, never in the URL:

1. the page that follows shows it, at the top, as a notice `<p role="status">`, in green, then forgets it: a reload or another tab does not show it again;
2. another session of the same account does not see it;
3. a password change keeps it in the new session that it opens;
4. a parameter `?notice=` in a URL is ignored.

The code `change-password` is not kept: the profile page shows it as a warning, in yellow, as long as the password is provisional. The notice is held in memory only, not in the [session file](#session-file).

| Code | Message | Shown after |
| --- | --- | --- |
| `saved` | « Le mois a été enregistré. » | a save of the grid |
| `validated` | « Le mois a été validé : ses écritures sont figées. » | a validation |
| `reopened` | « Le mois a été rouvert : ses écritures ont été contre-passées. » | a reopening |
| `adjusted` | « L'ajustement a été enregistré. » | an adjustment added |
| `unadjusted` | « L'ajustement a été supprimé. » | an adjustment deleted |
| `settings` | « Les réglages ont été enregistrés. » | a save of the settings page |
| `password` | « Le mot de passe a été changé. » | a password change in the profile page |
| `name` | « Le nom a été enregistré. » | a name change, profile page or account page |
| `created` | « Le compte a été créé. » | an account created |
| `disabled` | « Le compte a été désactivé : il ne peut plus se connecter. » | an account disabled |
| `enabled` | « Le compte a été réactivé. » | an account enabled |
| `sessions` | « Les sessions du compte ont été fermées. » | the sessions of another account closed |
| `deleted` | « Le compte a été supprimé. » | an account deleted |
| `profile` | « Le profil a été changé. » | a profile changed |
| `reset` | « Le mot de passe provisoire a été enregistré. » | a password reset |
| `configuration` | « La configuration a été enregistrée. » | a change of the configuration |
| `imported` | « L'import a été enregistré. » | an import from the data page |
| `erased` | « Vos données ont été supprimées : le suivi recommence au premier jour du mois. » | a deletion of the data from the data page |
| `subscription` | « La souscription IA a été enregistrée. » | a subscription added, changed or made the default |
| `subscription-deleted` | « La souscription IA a été supprimée. » | a subscription deleted |
| `shared` | « La souscription IA a été partagée. » | a subscription shared |
| `unshared` | « Le partage de la souscription IA a été retiré. » | a share withdrawn |
| `subscription-hidden` | « La souscription partagée n'est plus proposée dans l'assistant. » | a shared subscription hidden |
| `subscription-shown` | « La souscription partagée est de nouveau proposée dans l'assistant. » | a shared subscription offered again |
| `cancelled` | « L'action proposée a été annulée : rien n'a été écrit. » | a pending action of the assistant cancelled |
| `conversation-deleted` | « La conversation a été supprimée. » | a conversation deleted |
| `conversations-deleted` | « Toutes les conversations ont été supprimées. » | every conversation deleted |
| `change-password` | « Choisissez un nouveau mot de passe avant de continuer. » | any request of an account with a provisional password, `GET` included ([Provisional password](#provisional-password)) |

## FORM FIELDS

Every form posts `application/x-www-form-urlencoded` data, except the import of the data page, which posts `multipart/form-data`. Every form of a connected page carries the CSRF token of the session in the hidden field `csrf`. A checkbox is checked when its value is `on`.

### Login and logout

| Form | Route | Field | Control | Content |
| --- | --- | --- | --- | --- |
| Login | `POST /login` | `csrf` | hidden | the token of the login cookie `ts_login` |
| Login | `POST /login` | `login` | « Identifiant », text, 32 characters | the login |
| Login | `POST /login` | `password` | « Mot de passe », password, 256 characters | the password |
| Logout | `POST /logout` | `csrf` | hidden | the token of the session |

### Fields of the activity page

| Form | Route | Field | Control | Content |
| --- | --- | --- | --- | --- |
| Grid | `POST /dashboard/activity/{year}/{month}` | `revision` | hidden | the account revision |
| Grid | same | `am:YYYY-MM-DD` | « Matin du \<jour> \<mois> », list | a code identifier, or empty |
| Grid | same | `pm:YYYY-MM-DD` | « Après-midi du \<jour> \<mois> », list | a code identifier, or empty |
| Grid | same | `day:YYYY-MM-DD` | « Journée du \<jour> \<mois> », list | a code for both half-days, or empty to keep them |
| Grid | same | `comment:YYYY-MM-DD` | « Commentaire du \<jour> \<mois> », text, 500 characters | the comment; mandatory for every tracked day of the month |
| Validation | `POST /dashboard/activity/{year}/{month}/validate` | `revision` | hidden | the account revision |
| Reopening | `POST /dashboard/activity/{year}/{month}/reopen` | `revision` | hidden | the account revision |
| Reopening | same | `reason` | « Motif », text, 200 characters | the reason |
| Add an adjustment | `POST /dashboard/adjustments` | `revision`, `action` | hidden | the account revision, `add` |
| Add an adjustment | same | `date` | « Date », date from the first day of the month (or the tracking start when later) to its last day, the last day by default | `YYYY-MM-DD` |
| Add an adjustment | same | `counter` | « Compteur », list | a counter identifier |
| Add an adjustment | same | `amount` | « Quantité signée », text, 32 characters, placeholder « -1,5 ou 7/12 » | the quantity |
| Add an adjustment | same | `reason` | « Motif », text, 200 characters | the reason |
| Delete an adjustment | `POST /dashboard/adjustments` | `revision`, `action`, `id` | hidden | the account revision, `delete`, the identifier of the adjustment |

`YYYY-MM-DD` is the date of the row. The rows before the tracking start have no field.

### Fields of the settings page

| Field | Control | Content |
| --- | --- | --- |
| `revision` | hidden | the account revision |
| `configuration` | hidden | the configuration revision |
| `start` | « Date de début », date from 2000-01-01 to 2099-12-31, disabled once a month is validated | the tracking start |
| `public_holidays` | « Calendrier des jours fériés », list | the identifier of a zone, or empty for the default zone |
| `enabled:<ID>` | « \<compteur> activé », checkbox | the counter is enabled |
| `amount:<ID>` | « Montant propre, \<compteur> », text, 32 characters | the own amount, or empty for the default amount |
| `initial:<ID>` | « Solde initial, \<compteur> », text, 32 characters, disabled once a month is validated | the initial balance; empty means 0 |
| `color:<CODE>` | « Couleur, \<code> : \<libellé> », colour | the colour of the code |
| `reset-colors` | « Rétablir les couleurs par défaut », checkbox | clears every user colour |

`<ID>` is the identifier of a counter, `<CODE>` the identifier of a code.

### Fields of the profile page

| Form | Field | Control | Content |
| --- | --- | --- | --- |
| Sessions | `action` | hidden | `sessions` |
| Name | `action` | hidden | `name` |
| Name | `name` | « Nom et prénom », text, 60 characters | the name, empty allowed |
| Password | `action` | hidden | `password` |
| Password | `current` | « Mot de passe actuel », password, 256 characters | the current password |
| Password | `password` | « Nouveau mot de passe », password, 12 to 256 characters | the new password |
| Password | `confirm` | « Confirmation », password, 12 to 256 characters | the new password again |

### Fields of the subscriptions page

| Form | Field | Control | Content |
| --- | --- | --- | --- |
| Subscription | `action` | hidden | `subscription-add`, `subscription-edit`, `subscription-delete`, `subscription-default`, `subscription-share`, `subscription-unshare`, `subscription-hide` or `subscription-show` |
| Subscription | `id` | hidden | the identifier of the subscription; `<owner>:<identifier>` for `subscription-hide` and `subscription-show` |
| Subscription | `model` | « Modèle », list grouped by provider | `<provider>\|<model>` |
| Subscription | `custom_model` | « Autre modèle », text, 100 characters | a model of the same provider, empty allowed |
| Subscription | `key` | « Clef d'API » or « Nouvelle clef d'API », password, 512 characters | the key; empty keeps the key on a change |
| Subscription | `name` | « Nom », text, 60 characters | the name, empty allowed |
| Share | `login` | « Identifiant du compte », text, 32 characters, with the suggested logins below it when JavaScript runs | the login to share with or to withdraw |

### Fields of the assistant pages

| Form | Field | Control | Content |
| --- | --- | --- | --- |
| Question | `action` | hidden | `ask` |
| Question | `question` | « Question », text area | the question, 1 to `assistant.question_bytes` bytes |
| Question | `subscription` | « Souscription », list, only for a new conversation with several usable subscriptions | `<owner>:<identifier>`; absent, the first usable one |
| Question, action, deletion | `conversation` | hidden | the identifier of the conversation; empty for a new one |
| Action | `action` | buttons « Confirmer » and « Annuler » | `confirm` or `cancel` |
| Deletion | `action` | hidden | `delete` or `delete-all` |

### Fields of the data page

| Form | Field | Control | Content |
| --- | --- | --- | --- |
| Import | `revision` | hidden | the account revision of the page |
| Import | `file` | « Fichier d'import », file, `.json` | the file to import |
| Import | `replace` | « Remplacer les jours déjà saisis », checkbox | replaces the entries already made at the dates of the file |
| Import | `validate` | « Valider les mois importés », checkbox | validates the imported months afterwards |
| Import | `action` | button « Vérifier » or « Importer » | `check` or `import` |
| Deletion of the data | `revision` | hidden | the account revision of the page |
| Deletion of the data | `confirm` | « Retapez votre identifiant pour confirmer », text, 32 characters | the login, retyped |

### Fields of the administration

| Form | Route | Field | Control | Content |
| --- | --- | --- | --- | --- |
| « Nouveau compte » | `POST /admin/accounts` | `login` | « Identifiant », text, 32 characters | the login |
| « Nouveau compte » | same | `profile` | « Profil », list | `ADMIN` or `USER` |
| « Nouveau compte » | same | `password` | « Mot de passe provisoire », password, 12 to 256 characters | the provisional password |
| Every form of an account | `POST /admin/accounts/{login}` | `action` | hidden | `name`, `profile`, `disable`, `enable`, `sessions`, `reset` or `delete` |
| Every form of an account | same | `created` | hidden | the creation time of the account, in seconds since 1970-01-01 UTC: a form shown before the account was deleted then created again under the same login, at another second, is refused |
| Every form of a row of `/admin/accounts` | same | `return` | hidden | `list`: the answer comes back to `/admin/accounts` |
| « Changer le nom » | same | `name` | « Nom et prénom », text, 60 characters | the name |
| « Changer le profil » | same | `profile` | « Profil », list | `USER` or `ADMIN` |
| « Réinitialiser le mot de passe » | same | `password` | « Mot de passe provisoire », password, 12 to 256 characters | the provisional password |
| Deletion | same | `confirm` | « Retapez l'identifiant pour confirmer » or « Retapez « \<login> » », text, 32 characters | the login, retyped |

### Form fields of the configuration page

All the forms post to `POST /admin/settings` and carry `revision`, the configuration revision.

| Form | Field | Control | Content |
| --- | --- | --- | --- |
| Public holidays | `action` | button « Enregistrer » | `public-holidays-update` |
| Public holidays | `public_holidays` | « Calendrier par défaut », list | the identifier of the default zone |
| Code | `action` | button « Enregistrer » or « Supprimer »; hidden in « Nouveau code d'activité » | `activity-update`, `activity-delete`, `activity-add` |
| Code | `id` | hidden; « Identifiant », text, 16 characters, in « Nouveau code d'activité » | the identifier |
| Code | `label` | « Libellé », text, 60 characters | the label |
| Code | `effect` | « Effet », list « Alimente » or « Décompte » | `feeds` or `consumes` |
| Code | `counter` | « Compteur », list | the counter fed or consumed |
| Code | `color` | « Couleur », colour | the default colour |
| Code | `archived` | « Archivé », checkbox, not in « Nouveau code d'activité » | the code is archived |
| Counter | `action` | button « Enregistrer » or « Supprimer »; hidden in « Nouveau compteur » | `counter-update`, `counter-delete`, `counter-add` |
| Counter | `id` | hidden; « Identifiant », text, 16 characters, in « Nouveau compteur » | the identifier |
| Counter | `label` | « Libellé », text, 60 characters | the label |
| Counter | `short` | « Libellé court », text, 6 characters | the short label |
| Counter | `amount` | « Montant par défaut », text, 32 characters | the default amount |
| Counter | `every` | « Rythme », list « par mois » or « par an » | `month` or `year` |
| Counter | `period` | « Début de période (MM-JJ) », text, 5 characters | the period start |
| Counter | `carry` | « Solde reporté (arrondi à l'entier supérieur) », checkbox | the balance is carried over |
| Counter | `enabled` | « Activé par défaut », checkbox | enabled by default |
| Counter | `archived` | « Archivé », checkbox, not in « Nouveau compteur » | the counter is archived |

## COMMAND LINE

The options `--config=FILE.json`, `--database=FILE.json` and, for `init`, `--template=FILE.json` are left out below.

```text
timesheet init
timesheet run [--listen=...] [--origin=...] [--trusted-proxy=...] [--tls-cert=... --tls-key=...]
timesheet config show [KEY]
timesheet config set KEY VALUE [KEY VALUE...]
timesheet user add LOGIN [--profile=USER|ADMIN] [--provisional]
timesheet user profile LOGIN USER|ADMIN
timesheet user passwd LOGIN
timesheet user del LOGIN
timesheet user list
timesheet session clear
timesheet import LOGIN FILE.json [--from=YYYY-MM] [--to=YYYY-MM] [--dry-run] [--replace] [--validate]
timesheet help | version
```

### Syntax

- Options are written `--name=value` (or `--name` for a flag), anywhere on the line, before or after the arguments.
- An argument cannot start with `-`: every word starting with `-` is read as an option.
- An option that does not apply to the command is refused.
- No path has a default. Without `--config`, every setting takes its default; with it, the settings come from that file. The database comes from `--database`, or else from `database.path` of that file, and is asked for only by a command that opens it, when neither gives it. `config show` and `config set` need `--config`, `init` needs `--template`. A relative path is relative to the current directory.
- Without a command, the program prints the help and exits with 0. `-h` and `--help` print it too.

### Common options

| Option | Value | Default |
| --- | --- | --- |
| `--database=<file>` | the database file; not empty, not ending with `/`, its file name not `.` nor `..` and not ending with `.lock`, `.sessions` or `.tmp` in any case | `database.path` of the [server configuration](#server-configuration); none when the file names no database, or without `--config` |
| `--config=<file>` | the [server configuration](#server-configuration); not empty, not ending with `/` | none: without it, every setting takes its default; required by `config show` and `config set` |
| `--log-level=<level>` | the lowest level written on the standard error: `debug`, `info`, `warn` or `error`, in any case, with an optional offset such as `info+2` | `info`; `server.log_level` of the server configuration for `run` |

- A database file name cannot be `.` or `..`, nor end with `.lock`, `.sessions` or `.tmp` in any case, the suffixes of the lock, session and temporary files written next to it ([Lock](#lock)).
- A relative database path, given by `--database` or by `database.path`, is relative to the current directory; the commands turn it into an absolute path before opening the file.
- From the sources, the database of the repository is given explicitly: `--database=var/lib/timesheet/timesheet-database.json`, with `--config=etc/timesheet/timesheet.json`, whose `database.path` is the database of the package.
- The unit and the commands of the packages give `--config=/etc/timesheet/timesheet.json` ([OPERATIONS.md](OPERATIONS.md#two-installations)); in the container image, its entry point adds it to every command ([OPERATIONS.md, Container](OPERATIONS.md#container)).
- A command that opens the database while neither `--database` nor `database.path` gives it is a usage error (exit 2): `timesheet: <command>: no database: give --database or set database.path in --config`. So are `config show` or `config set` without `--config` (`timesheet: config show: --config is required`), and `init` without `--template`.

### Exit codes

| Code | Meaning | Cases |
| --- | --- | --- |
| 0 | success | the command did its work; `help`, `version`, no command, `-h`, `--help` |
| 1 | error | an invalid or out-of-bounds setting value for `config set`, or two linked values that do not agree; an invalid server configuration, for every command that reads it, or a missing file given by `--config`, except for `config set`; an invalid `--profile` or profile argument, or an invalid `--origin` or `--listen`, which are checked once the database is open; an invalid login; an unknown account; a login taken; the last administrator; a refused import; a weak password or two different passwords; the database missing, existing already for `init`, locked by the server, malformed or insecure; `run` unable to listen or to set up TLS |
| 2 | usage error | an unknown command, option or key of `config show` or `config set`; `config`, `user` or `session` without subcommand; a wrong number of arguments, `config set` without a value for each key included; an option that does not apply to the command; a malformed value of `--from`, `--to`, `--log-level` or `--trusted-proxy`; an empty `--origin=` or `--trusted-proxy=`; `--from` after `--to`; `--tls-cert` without `--tls-key`, or the reverse; an invalid `--database` or `--config`; a command that opens the database while neither `--database` nor `database.path` gives it; `config show` or `config set` without `--config`; `init` without `--template` |
| 128 + signal | password prompt interrupted | 129 for SIGHUP, 130 for SIGINT, 131 for SIGQUIT, 143 for SIGTERM; the terminal echo is restored |

An error is printed on the standard error as `timesheet: <message>`. An error found while reading the options and the arguments is followed by `Try 'timesheet help' for more information.`, except the checks of the months of `import`, which print only their line: `timesheet: invalid --from`, `timesheet: invalid --to` or `timesheet: --from after --to`, and a command that opens the database while neither `--database` nor `database.path` gives it, which prints only `timesheet: <command>: no database: give --database or set database.path in --config`.

### Help and version

- `help` prints the usage in English: `Usage: timesheet COMMAND [ARGUMENT...] [OPTION...]`, then the groups `Commands`, `Common options` and `Options of <command>` for `init`, `run`, `user add` and `import`, then how the password is read and how options are written.
- `version` prints `timesheet <version> (revision <commit>, <date>)`, the release number written in the program followed by the commit and its date, with `, modified` before the closing parenthesis when the tree had uncommitted changes. A binary built outside a git checkout, from an archive, prints only `timesheet <version>`.

### Bootstrap

1. `timesheet init` creates the database from the configuration template.
2. `timesheet user add <login> --profile=ADMIN` creates the first administrator.
3. `timesheet run` starts the server.

The server also starts before step 2, without any account: it then warns, and step 2 is done with the server stopped.

The procedure, with the paths of an installed package, is in [OPERATIONS.md, Install](OPERATIONS.md#install) and [README.md, First start](../README.md#first-start).

The container image does steps 1 and 2 at its first start, before `run`, when the database named by its server configuration does not exist yet: `init`, then `user add <login> --profile=ADMIN --provisional`, with the login of `TIMESHEET_USERNAME`, or else `admin`, and the password of `TIMESHEET_PASSWORD`, or else, when the variable is missing or empty, the default password `Tms@Admin%26`. The account must change its password at its first login ([Provisional password](#provisional-password)). When the database exists, nothing is created; when `init` or `user add` fails, the container stops with exit 1. The procedure is in [OPERATIONS.md, Container](OPERATIONS.md#container).

### Passwords

`user add` and `user passwd` read the password from the first source available:

1. the environment variable `TIMESHEET_PASSWORD`, refused when it is set but empty;
2. otherwise, on Linux, twice on the terminal without echo, after the prompts `Password: ` and `Confirmation: `; the two must match, and an interruption restores the echo;
3. otherwise one line of the standard input (4096 bytes at most); when that input is a terminal or another character device, such as `/dev/null`, on another system than Linux, the command first prints `warning: the password is read with echo on this system; prefer TIMESHEET_PASSWORD or a pipe`.

The password policy then applies ([SECURITY.md](SECURITY.md#passwords)). The help text lists these sources in another order; the order of this list is the one applied.

### Lock and running server

| Command | Takes the lock | Needs the server stopped | Reads the database |
| --- | --- | --- | --- |
| `run` | yes, while it runs | not applicable | yes |
| `init` | yes | yes | creates it |
| `user add`, `user profile`, `user passwd`, `user del` | yes | yes | yes |
| `user list` | no | no | yes |
| `session clear` | yes | yes | yes |
| `import`, `--dry-run` included | yes | yes | yes |
| `config show`, `config set` | no | no | no |
| `help`, `version` | no | no | no |

- Every command but `help` and `version` reads the [server configuration](#server-configuration) first when `--config` is given, and takes the defaults otherwise; the commands that open the database take its path, its size limit, the limits of the accounts and the iterations of the password hashes from it.

- The lock is `<file name>.lock` next to the database ([Database](#database)).
- A command that needs the lock while the server runs fails with exit 1: `the file is used by another process`, followed by `(stop the server or use /admin/accounts)` for the `user` commands and `(stop the server first)` for `session clear` and `import`.
- A command that takes the lock checks the database file and its directory ([Files and directories](#files-and-directories)). `user list` checks only the file.

### `timesheet init`

- Reads the configuration template given by `--template`, a required option, strictly, 1 MiB at most ([Configuration template](#configuration-template)).
- Creates the database with that configuration and no account, mode 0600.
- Refuses an existing database (exit 1).
- Does not create the directory of the database: it must exist.
- Prints `Database created: <file>`, then `Now create an administrator: ` followed by the command that creates the first administrator, with `--database` and, when it was given, `--config`, both absolute and quoted for the shell when they hold another character than a letter, a digit or one of `_-./=:@%+,`.

### `timesheet run`

- Reads the server configuration first, then takes the lock and loads the database.
- Starts even while no enabled `ADMIN` exists, and then logs the warning `no administrator account` with the hint `stop the server, run 'timesheet user add <login> --profile=ADMIN --database=<file>', then start it again`, followed by `--config=<file>` when it was given; these arguments are absolute and quoted for the shell as in the hint of `init` ([Start condition of the server](#start-condition-of-the-server)).
- Restores the sessions of the [session file](#session-file).
- Stops on SIGINT or SIGTERM, letting the requests in progress end for up to `server.shutdown_timeout` (10 seconds by default), then saves the sessions.
- Takes its timeouts and request sizes from the [section server](#section-server) of the server configuration, and the durations of its sessions and of its login throttle from the sections [sessions](#section-sessions) and [login](#section-login).

#### Address and origins

| Setting | Option | Server configuration | Default |
| --- | --- | --- | --- |
| Listening address | `--listen=host:port` | `server.listen` | `127.0.0.1:8080` |
| Accepted origins | `--origin=scheme://host[:port]` | `server.origins` | `http://127.0.0.1:8080`, `http://localhost:8080` |
| Trusted proxies | `--trusted-proxy=address-or-prefix` | `server.trusted_proxies` | the RFC 1918 ranges |

- `--origin` and `--trusted-proxy` can be repeated, and each also takes a comma-separated list.
- An option replaces the matching setting of the file; it cannot empty the list of trusted proxies.
- A page opened under a host name that is not an accepted origin gets 421 « Hôte inconnu ».

#### Derived origins

- With `--listen` and without `--origin`, `server.origins` is ignored: the accepted origins are derived from the listener.
- They are `localhost`, `127.0.0.1`, `::1`, the IPv4 addresses of the interfaces, the host name and `<host name>.local`, with the port of the listener.
- They use `https` when a certificate pair is given, `http` otherwise.

#### HTTPS

- `--tls-cert=<file>` and `--tls-key=<file>`, or `server.tls_cert` and `server.tls_key` in the server configuration, serve HTTPS with a provided certificate pair. The options replace the pair of the file.
- They need https origins: `--origin=https://...`, https origins in the file, or origins derived with `--listen`. The check is made once the options are applied, so that `--origin=https://...` can go with a pair of the file.
- The lowest TLS version is `server.tls_min_version` (1.2 by default); the cipher suites are fixed ([SECURITY.md](SECURITY.md#transport)).
- With https origins, every answer carries `Strict-Transport-Security` with the `max-age` of `server.hsts_max_age` (one year by default).
- With http origins and a pair, `run` refuses to start (exit 1).
- With http origins and no pair, `run` warns at start that passwords and cookies travel in clear.

#### Behind a reverse proxy

1. Give the public URL as origin: `--origin=https://...` or `server.origins`.
2. Give the address of the proxy as trusted proxy: `--trusted-proxy` or `server.trusted_proxies` (IP addresses or prefixes).
3. The proxy must pass the public `Host` unchanged.
4. The proxy must append the client address to `X-Forwarded-For`.

- The header `X-Forwarded-For` is believed only from a trusted proxy.
- The client is then the last address of that header, the one the proxy appended.
- The risks of this set-up are in [SECURITY.md](SECURITY.md#client-address-and-trusted-proxies); the procedure is in [OPERATIONS.md](OPERATIONS.md#behind-a-reverse-proxy-on-another-machine).

### `timesheet config show` and `timesheet config set`

- `config show` prints the server configuration read from the file, as JSON, in the [layout](#layout) of the file, every key included. A missing file is an error. It never shows the options of a running server.
- `config show KEY` prints only the value of one key, raw, on one line: a string without quotes, a duration in its shortest form, a number, a list as values separated by commas, `null` for no value. A script reads a setting this way, for instance `config show database.path`.
- `config set KEY VALUE [KEY VALUE...]` reads each value according to the type of its key, applies all of them, validates the whole configuration, then rewrites the file atomically in its layout, creating its directory (mode 0755) when missing. A file that exists keeps its mode, its owner and its group: a user who cannot give them back to the new file gets an error, and nothing is written. A symbolic link is followed: the file it names is rewritten, the link stays; a link to a missing file is refused with `<file> is a link to a missing file`. A new file gets mode 0644. Nothing is written when one value is refused. A missing file starts from the defaults, without database.
- Linked keys are set in the same command: `server.tls_cert` with `server.tls_key`, `sessions.idle_timeout` with `sessions.max_lifetime`, `login.initial_delay` and `login.forget_after` with `login.max_delay`, `limits.accounts` with `sessions.per_account`. Set one after the other, the first may be refused.
- `config set` prints `<keys> updated in <file>; restart the server to apply it.`, the keys separated by commas.
- Neither reads the database nor needs the server stopped.
- Both need `--config`, since they act on a file; in a package, give `--config=/etc/timesheet/timesheet.json`.
- The keys are those of the [server configuration](#server-configuration), written `<section>.<key>`: `server.listen`, `sessions.idle_timeout`, `limits.accounts`, and so on. An unknown key is a usage error (exit 2).

| Type of the key | Value given to `config set` |
| --- | --- |
| address (`server.listen`) | `host:port`, the port being 0 to 65535 or a service name |
| list (`server.origins`, `server.trusted_proxies`) | values separated by commas, spaces around them ignored; an empty value gives an empty list, which `server.trusted_proxies` accepts and `server.origins` refuses |
| text (`server.timezone`, `server.log_level`, `server.tls_min_version`) | the value as given |
| path (`server.tls_cert`, `server.tls_key`, `database.path`) | a path, or `null` to remove it |
| duration (`server.*_timeout`, `server.hsts_max_age`, `sessions.*`, `login.*`) | a duration such as `500ms`, `30s`, `5m`, `1h30m` or `8h` |
| number (sizes, counts, `passwords.iterations`) | a whole number |

Change `server.origins` together with `server.listen`: a request whose `Host` is not an accepted origin is refused (421). Both can go in the same command, for instance `config set server.listen 0.0.0.0:8080 server.origins http://timesheet.example:8080`.

### `timesheet user add`

- Creates an account `enabled`, with the profile of `--profile` (`USER` by default) and its tracking start on the first day of the current month in `server.timezone`.
- Reads the server configuration first, for the path of the database, the time zone, the limit of accounts and the iterations of the hash.
- Refuses a login already taken (exit 1), before reading the password.
- Reads the password ([Passwords](#passwords)).
- The account does not have to change its password, unless `--provisional` is given: the password is then provisional ([Provisional password](#provisional-password)).
- Prints `Account <login> created (<profile>).`.

### `timesheet user profile`

- Sets the profile `USER` or `ADMIN` of an account.
- Closes its sessions first, even when the change is then refused (last enabled `ADMIN`, exit 1).
- Prints `Profile of <login>: <profile>`.

### `timesheet user passwd`

- Reads a new password, closes the sessions of the account, then sets the password.
- Clears the flag of a provisional password.
- Prints `Password of <login> changed.`.

### `timesheet user del`

- Closes the sessions of the account, then deletes the account and all its data.
- Refuses the last enabled `ADMIN` (exit 1).
- Prints `Account <login> deleted.`.

### `timesheet user list`

- Prints one row per account under the headers `LOGIN`, `PROFILE`, `STATE`, `CREATED`, `LAST LOGIN`, `NAME`.
- The state is `enabled` or `disabled`; it can only be changed from the web.
- The creation and the last login are UTC timestamps (RFC 3339); the last login is `never` before the first.
- Works while the server runs.

### `timesheet session clear`

- Closes every session of every account by emptying the [session file](#session-file).
- Needs the server stopped, like every command that writes.
- Prints `All sessions are closed.`.

### `timesheet import`

Imports days, adjustments and settings into an account ([Import](#import)).

## IMPORT

### What the import reads

- `import LOGIN FILE.json` reads a JSON file of the format `timesheet-import` and writes its content into the account `LOGIN`.
- The same import is offered to every account in its data page ([Data page](#data-page)), in French, without `--from` and `--to`.
- The application reads no spreadsheet. A spreadsheet is converted to this format outside the application: its categories become activity codes, its notes comments, and the hand corrections of a balance that the rules do not explain become adjustments.

### File format

```json
{
  "format": "timesheet-import",
  "version": 1,
  "settings": {
    "start": "2025-01-01",
    "public_holidays": "fr-alsace-moselle",
    "counters": {
      "CP": { "enabled": true, "initial": "1.5" },
      "TPC": { "enabled": true, "amount": "7/12", "initial": "0" }
    }
  },
  "days": { "2025-01-02": { "am": "TT", "pm": "TT", "comment": "Réunion d'équipe" } },
  "adjustments": [ { "date": "2025-06-30", "counter": "CP", "amount": "0.5", "reason": "Alignement sur le bulletin de paie" } ]
}
```

| Field | Required | Content |
| --- | --- | --- |
| `format` | yes | exactly `"timesheet-import"` |
| `version` | yes | exactly `1` |
| `settings` | no | `start`, a date `YYYY-MM-DD`; `public_holidays`, the identifier of a [zone](#public-holidays); `counters`, a map from a counter identifier to `enabled`, `amount` and `initial`, as in the [database](#database) |
| `days` | no | a map from a date `YYYY-MM-DD` to a day record `am`, `pm`, `comment` |
| `adjustments` | no | a list of `date`, `counter`, `amount`, `reason` |

- The file is 8 MiB at most.
- It is decoded strictly: unknown fields and data after the object are refused.
- As for the database, the standard decoder matches field names without regard to case and keeps the last of repeated keys: the fields of two repeated objects and the keys of two repeated maps are merged, but a key repeated within one map, a date or a counter, keeps only its last record.

### Import template

- `share/timesheet/import-template.json` is a small fictitious file of this format, installed next to the configuration template.
- It covers January 2026: every day entered, a public holiday, paid leave, a half-day of `RTT`, an absence, comments, the settings of `CP`, `RTT` and `TPC`, and one adjustment.
- A test imports it, so that it stays valid.

### Options

| Option | Effect |
| --- | --- |
| `--from=YYYY-MM` | skips the days and adjustments before this month; the settings are not filtered |
| `--to=YYYY-MM` | skips the days and adjustments after this month; the settings are not filtered |
| `--replace` | replaces an existing day entry at a date present in the file; other dates are untouched |
| `--dry-run` | goes through every check, the size limits of the account included, and writes nothing; it still takes the lock, so it needs the server stopped |
| `--validate` | validates the imported months afterwards ([Validation of the imported months](#validation-of-the-imported-months)); with `--dry-run`, lists the months that would be validated |

`--from` after `--to`, or a malformed month, is a usage error (exit 2).

### Imported settings

Settings follow the rules of the [settings page](#settings-page):

- only existing counters that are not archived;
- own amounts between 0 and 366, initial balances between -366 and 366;
- `start` and the initial balances can change only while no month is validated; the same values are accepted;
- `start` never after the date of an existing adjustment;
- an absent `initial` keeps the stored initial balance, an absent `enabled` the stored flag;
- an absent `amount` means the default amount;
- a known `public_holidays` becomes the zone of the account, applied from the first open month; an absent one keeps the stored zone, an unknown one is a problem;
- a `null` value counts as an absent one;
- `enabled` and `amount` apply from the first open month.

### Imported days

- Each imported day is checked like an entry of the grid, its comment trimmed the same way.
- Its codes must be offered: an archived code, or a consuming code whose counter is disabled, is refused unless it is already stored on that half-day.
- A consuming code is refused on a weekend or a public holiday, unless it is already stored on that half-day ([Codes offered in a grid](#codes-offered-in-a-grid)).
- The comment follows the [free-text rule](#free-text-rule), 500 characters at most.
- The month must be within the planning horizon and not validated.
- An existing entry is replaced only with `--replace`, even an identical one.
- Empty days are skipped.

### Imported adjustments

- Each imported adjustment is checked like one entered in the activity page ([Adjustments](#adjustments)).
- The date is mandatory: an adjustment without one is refused, whatever `--from` and `--to`.
- The date must fall in an open month of the tracking, not before the tracking start, within the planning horizon.
- The counter must exist and not be archived.
- The quantity is signed, non-zero, between -366 and 366.
- The reason is mandatory, one line, 200 characters at most; it is trimmed.
- It receives an identifier.
- An adjustment identical to a stored one (date, counter, amount, reason) is skipped with a warning, so that a file can be imported again with `--replace`. Identical adjustments count one by one: a file holding the same adjustment twice adds both to an account that has none, and skips both when the account already holds two.

### Moved tracking start

- When the first imported day precedes the tracking start of an account without validated month, the tracking start moves back to that day, with a warning.
- This also overrides a later `start` given in the same file.
- The warning gives the stored tracking start, then the new one.
- A tracking start changed by the file, its `start` and its days together, is refused when a stored adjustment is dated before it.
- When the account has a validated month, such a file is refused.

### Errors

- Every problem is listed, prefixed with `error: `, then the command exits with 1 and nothing is written.
- A value that cannot be decoded, a quantity written with a comma or a date that does not exist (`2026-11-31`) for instance, stops the reading before any check: the command prints the single line `timesheet: activity: invalid import file: <decoding error>`, without date nor field, and exits with 1.
- Problems of days and adjustments carry their date, except an adjustment without one; problems of settings do not.
- A file with no day and no adjustment within `--from` and `--to`, and no `settings`, is refused (`no day to import`). A file holding only `"settings": {}` is accepted.
- The command line writes the messages in English, except the refusals of the grid and the labels of counters, which are the French texts of the pages. The data page writes them in French, and adds « Le fichier ne contient aucun jour à importer » for `no day to import`.

| Problem | Text after `error: ` | Text of the page |
| --- | --- | --- |
| Day beyond the planning horizon | `<date>: beyond the planning horizon` | « \<jj/mm/aaaa> : au-delà de l'horizon de planification » |
| Day in a validated month | `<date>: the month is validated` | « \<jj/mm/aaaa> : le mois est validé » |
| Day already entered, without `--replace` | `<date>: an entry already exists (use --replace)` | « \<jj/mm/aaaa> : une saisie existe déjà (cochez « Remplacer les jours déjà saisis ») » |
| Day refused by the grid rules | `<date>: <problem>`, the problem being the French text of [Refusals of the activity page](#refusals-of-the-activity-page) | « \<jj/mm/aaaa> : \<problème> » |
| File starting before a validated tracking start | `the file starts before the tracking start, already validated` | « Le fichier commence avant la date de début, déjà validée » |
| Adjustment without a date | `adjustment without a date` | « Ajustement sans date » |
| Adjustment outside an open tracked month | `adjustment of <date>: the date must fall in an open month, from the tracking start` | « Ajustement du \<jj/mm/aaaa> : la date doit tomber dans un mois ouvert, à partir de la date de début » |
| Adjustment on an unknown or archived counter | `adjustment of <date>: counter "<ID>" unknown or archived` | « Ajustement du \<jj/mm/aaaa> : compteur "\<ID>" inconnu ou archivé » |
| Adjustment quantity | `adjustment of <date>: a non-zero quantity between -366 and 366 is expected` | « Ajustement du \<jj/mm/aaaa> : une quantité non nulle entre -366 et 366 est attendue » |
| Adjustment reason | `adjustment of <date>: a reason is required, one line of 200 characters at most` | « Ajustement du \<jj/mm/aaaa> : un motif est requis, une ligne de 200 caractères au plus » |
| `start` locked | `tracking start: a month is already validated, it can no longer change` | « Date de début : un mois est déjà validé, elle ne peut plus changer » |
| `start` after an adjustment | `tracking start: adjustments are dated before <date>` | « Date de début : des ajustements sont datés avant le \<jj/mm/aaaa> » |
| Unknown zone in `settings` | `public holidays: zone "<zone>" unknown` | « Calendrier des jours fériés "\<zone>" inconnu » |
| Unknown or archived counter in `settings` | `counter "<ID>" unknown or archived` | « Compteur "\<ID>" inconnu ou archivé » |
| Own amount | `<counter>: invalid accrual amount (between 0 and 366)` | « \<compteur> : montant d'acquisition invalide (entre 0 et 366) » |
| Initial balance | `<counter>: invalid initial balance (between -366 and 366)` | « \<compteur> : solde initial invalide (entre -366 et 366) » |
| Initial balance locked | `<counter>: a month is already validated, the initial balances can no longer change` | « \<compteur> : un mois est déjà validé, les soldes initiaux ne peuvent plus changer » |

### Report

The command prints, in this order:

1. the warnings, prefixed with `warning: ` (`tracking start moved from <date> to <date>`, `adjustment of <date> already present, ignored`);
2. `Settings read: tracking start <date>, <n> counter(s).` when the file has `settings`;
3. `<n> day(s) read, from <month> to <month>.` when the file has days;
4. month by month, within the imported months, `missing entries in <month>: <dates>`: the days before today with a half-day or both left without entry, weekends and public holidays included;
5. `<n> adjustment(s) read.` when the file has adjustments;
6. with `--validate`, `<month> validated.` per month, or `validation stopped at <month>: <reason>`;
7. `Dry run: nothing was written.` with `--dry-run`, otherwise `Import saved.`.

Months are written `YYYY-MM` and dates `YYYY-MM-DD`.

### Validation of the imported months

With `--validate`, and only when days were imported:

1. The open months are validated in order, from the first open month.
2. The validation stops after the last imported month, and before the current month: the current month is never validated, without message.
3. It stops at a month that was not imported (`this month was not imported`) or that cannot be validated (the error of the validation, in English).
4. Each month is recorded as validated on its last day at 23:59:59 in `server.timezone`, its frozen entries created at that time, as if it had been validated then.
5. A month whose history already holds a later event (it was reopened since) is recorded as validated at the time of the import, so that its history stays in order.

After a validation, importing the same file again fails: its days and adjustments fall in validated months. A different `start` or initial balance would also be refused, since they are locked.

### Export

`GET /dashboard/data/export` downloads the data of the viewer's own account in the import format:

- `format` and `version`;
- `settings`: the tracking start, the effective zone of public holidays (the zone of the account or else the default zone), and every counter of the configuration that is not archived, in the order of the configuration, each with its effective values: `enabled` (`true` or `false`, the choice of the account or else the default of the counter), `amount` (the own amount or else the default amount) and `initial`;
- `days`: every stored day entry, by date, with `am`, `pm` and `comment` always written, `null` when empty;
- `adjustments`: every adjustment, open or frozen in a validated month, by date; `[]` when there is none.

Nothing is left implicit: every field is written, and no value depends on the defaults of the configuration. Imported again, `public_holidays`, `enabled` and `amount` therefore become choices of the account: a later change of the default zone or of the default amount no longer applies to it. Archived counters are left out, since the import refuses them.

| Header | Value |
| --- | --- |
| `Content-Type` | `application/json; charset=utf-8` |
| `Content-Disposition` | `attachment; filename="timesheet-<login>-<aaaa-mm-jj>.json"`, the date of today |
| `Cache-Control` | `no-store` |

- The file is indented with two spaces; quantities keep their written form.
- It does not hold the history of the months (validations, reopenings) nor the ledger: they are computed again. Imported into an empty account with « Valider les mois importés », it gives the same balances, the months being validated on their last day. This holds while the history follows the current rules: a code or a counter archived after use, a consuming counter disabled after use, or an adjustment on an archived counter make the import refused; an own amount changed after a validation, or day entries stored before the tracking start (kept when the start moved later), give other balances, the history being computed again with the current settings.
- It can be imported again with the command `import`, or from the data page within `server.max_body_bytes`.

### Source data

The spreadsheet an employee converts, and the import file made from it, are personal data. They never enter the repository: the tests build synthetic import files, and the [example database](#example-database) is anonymized.

## DATABASE

### File

- A single JSON file holds the configuration, the accounts and their data.
- Its path is given by the option `--database`, or else by `database.path` of the [server configuration](#server-configuration), a key without default.
- Its mode is 0600, and it belongs to the account that runs the program.
- `init` creates it ([`timesheet init`](#timesheet-init)).

### Example

```json
{
  "version": 1,
  "revision": 42,
  "configuration": {
    "revision": 3,
    "public_holidays": "fr-metropole",
    "activities": [ { "id": "CP", "label": "Congé payé", "consumes": "CP", "feeds": null, "color": "#e01b24", "archived": false } ],
    "counters": [ { "id": "CP", "label": "Congés payés", "short_label": "CP", "enabled_by_default": true, "accrual": { "every": "month", "amount": "2.08" }, "period_start": "06-01", "carry_over": true, "archived": false } ],
    "ai_providers": [ { "name": "mistral", "label": "Mistral", "format": "openai", "base_url": "https://api.mistral.ai/v1", "default_model": "mistral-small-latest", "models": [ "mistral-large-latest", "mistral-small-latest" ] } ]
  },
  "accounts": [
    {
      "login": "jdupont",
      "name": "Jeanne Dupont",
      "profile": "ADMIN",
      "password": "pbkdf2-sha256$600000$<salt>$<key>",
      "must_change_password": false,
      "state": "enabled",
      "created": "2026-11-08T10:00:00Z",
      "last_login": "2026-11-08T10:05:00Z",
      "revision": 17,
      "settings": { "start": "2025-01-01", "public_holidays": null, "counters": { "CP": { "enabled": true, "amount": null, "initial": "1.5" }, "TPC": { "enabled": true, "amount": "7/12", "initial": "0" }, "RTT": { "enabled": false, "amount": null, "initial": "0" } }, "colors": { "TT": "#c8e0b0" } },
      "days": { "2026-11-03": { "am": "TT", "pm": "TT", "comment": "Formation sécurité" }, "2026-11-04": { "am": "TT", "pm": null, "comment": null } },
      "adjustments": [ { "id": "9f2c…", "date": "2026-11-30", "counter": "CP", "amount": "1.5", "reason": "alignement bulletin de paie" } ],
      "months": { "2026-10": { "status": "validated", "validated_at": "…", "rules": { "ABS": "0", "CP": "2.08", "CSS": "0", "JF": "0", "TP": "0", "TPC": "7/12", "TT": "0", "WE": "0" }, "indicators": { "holidays": "0", "total": "22", "working_days": "22", "missing": "0", "upcoming": "0", "by_counter": { "CP": "4", "TP": "4", "TT": "14", "WE": "9" }, "by_code": { "CP": "4", "TP": "4", "TT": "14", "WE": "9" } }, "history": [ { "at": "…", "action": "validated", "reason": null } ] } },
      "ledger": [ { "id": "a81f…", "month": "2026-10", "date": "2026-10-31", "counter": "CP", "kind": "credit", "amount": "2.08", "label": "Acquisition octobre 2026", "ref": null, "created": "…" } ],
      "assistant": {
        "revision": 4,
        "subscriptions": [ { "id": "5d1e…", "name": "Mistral mistral-small-latest", "provider": "mistral", "model": "mistral-small-latest", "key": "<sealed, base64>", "shared_with": [ "mmartin" ], "created": "…" } ],
        "hidden": [],
        "conversations": [ { "id": "c07a…", "title": "Mon solde de congés ?", "subscription": { "owner": "jdupont", "id": "5d1e…" }, "updated": "…", "exchanges": [ { "question": "Mon solde de congés ?", "answer": "Il vous reste **12,50** jours.", "failure": "", "provider": "mistral", "model": "mistral-small-latest", "asked": "…", "answered": "…" } ], "pending": null } ]
      }
    }
  ]
}
```

The configuration of the example is shortened to one code, one counter and one provider; `rules` holds the effective amount of each counter enabled for the account at the validation (here `RTT` is disabled).

### Conventions of the fields

- Quantities are strings (`"2.08"`, `"7/12"`), dates `YYYY-MM-DD`, months `YYYY-MM`, times RFC 3339 in UTC.
- Every field of the schema is always written.
- A field without value is written `null`, and a `null` is read as an empty value.
- Empty lists and maps are written `[]` and `{}`, except the colours of an account, written `null` while the account never changed them.
- The configuration template follows the same rules.

| Field | `null` when |
| --- | --- |
| `name` | no name is given |
| `last_login` | the account never logged in |
| `consumes`, `feeds` of a code | the code does not consume, or does not feed, a counter |
| `am`, `pm` of a day | the half-day is not entered |
| `comment` of a day | no comment is entered |
| `amount` of a counter setting | no own amount is set |
| `enabled` of a counter setting | the account did not choose |
| `ref` of a ledger entry | the entry cancels nothing |
| `reason` of a month event | the event is a validation |
| `validated_at`, `rules`, `indicators`, `history` of a month | the month was never validated |
| `colors` of the settings | the account never changed a colour |
| `public_holidays` of the settings | the account follows the default zone of the configuration |
| `ai_providers` of the configuration | the database predates the assistant: the four providers of the template are offered |
| `assistant` of an account | the account never used the assistant |
| `pending` of a conversation | no action waits for a confirmation |
| `answered` of an exchange | the answer is being prepared, or was interrupted by a stop of the server |

- `public_holidays` of the configuration is the identifier of a [zone](#public-holidays); an older database without it, or with an empty one, follows `fr-metropole` ([Migration notes](#migration-notes)).
- `state` is mandatory: `enabled` for an active account, `disabled` for a disabled one.
- `profile` is `USER` or `ADMIN`.
- `status` of a month is `open` or `validated`; the `action` of a month event is `validated` or `reopened`.
- `kind` of a ledger entry is `debit`, `credit`, `adjustment` or `reversal`.
- `key` of a subscription is the API key sealed with AES-256-GCM by the [key file](#key-file), bound to the owner and the identifier of the subscription, in base64; `failure` of an exchange is empty unless the answer failed.

### Schema version

- `version` is the version of the schema. Version 1 is the current one.
- A database of a newer version is refused (`the database was written by a newer version`): an older binary cannot read it.
- Any other version is refused as malformed.
- The layout of version 1 changed without a change of version: see [Migration notes](#migration-notes).

### Revisions

| Revision | Raised by | Checked by |
| --- | --- | --- |
| `revision` | every write | nothing: it counts the writes |
| `configuration.revision` | every change of the configuration | the forms of the configuration page and of the settings page |
| `accounts[].revision` | every write of one account: grid, settings, adjustments, validation, reopening, import; also a change of profile, state or password, from the web or the command line | the forms of that account, and the pending actions of the assistant |
| `accounts[].assistant.revision` | every change of the subscriptions, the hidden shares or the conversations of one account | the forms of the subscriptions and of the assistant pages |

A write of another account never refuses one's own form.

### Validation at load

The whole file is validated when it is loaded. An invalid database stops the program (exit 1) with the JSON path of the error; a value that cannot be decoded, such as an invalid quantity, is reported with the decoding error.

| Object | Checks |
| --- | --- |
| File | strict decoding: unknown fields and data after the object refused; the standard decoder matches field names without regard to case and keeps the last of repeated keys: the fields of two repeated objects and the keys of two repeated maps are merged, but a key repeated within one map keeps only its last value; `version` as in [Schema version](#schema-version); `database.max_bytes` at most (64 MiB by default) |
| Configuration | the rules of [Validation before writing](#validation-before-writing) |
| Accounts | logins valid and unique; profiles, states, names (one line, 60 characters at most, no surrounding space); password hashes with 1 000 to 1 200 000 iterations |
| Settings | tracking start; a known zone of public holidays or `null`; counters that exist; own amounts and initial balances within their bounds; colours `#rrggbb` of codes that exist |
| Days | known codes; comments following the [free-text rule](#free-text-rule) |
| Adjustments | an existing counter; unique non-empty identifiers; a date; a non-zero amount between -366 and 366; a non-empty reason following the free-text rule |
| Ledger | an existing counter; a valid kind; unique non-empty identifiers; a date within the month of the entry; a label following the free-text rule, 1000 characters at most |
| Reversals | each cancels one earlier entry that is not itself a reversal, of the same counter, month and date, with the opposite amount, at most once |
| Months | status `open` or `validated`; history actions and reopening reasons; a validated month has its validation time and indicators; month rules name existing counters |
| Consistency | validated months follow each other without gap from the month of the tracking start; every entry of a month that is not validated is a reversal or cancelled by one |
| AI providers | 20 at most; unique names `[a-z0-9-]{1,16}`; a label; a known format; an https `base_url` without user, query, fragment or final slash; a default model and 30 suggested models at most, of 1 to 100 printable characters without space |
| Assistant | 20 subscriptions at most, with unique identifiers, a name, a provider of the configuration, a model, a sealed key, a creation time, and 50 shares at most with other existing accounts; 100 hidden shares of other accounts at most; 200 conversations at most, with unique identifiers, a title, a subscription reference, an update time, 100 exchanges at most whose texts follow the free-text rule on each line, and a pending action of a known size |
| Round trip | the file, encoded again, can be decoded again |

### Writing

- The database is kept in memory.
- It is rewritten atomically and synchronously on each change: temporary file `<name>.<digits>.tmp` next to it, `fsync`, rename, `fsync` of the directory.
- Every write checks that the encoded file can be decoded again before replacing it; the load checks the same. A value that could not be read back never reaches the disk.
- A write that would make the file larger than `database.max_bytes` (64 MiB by default) is refused.

### Files and directories

- The database file must be a regular file, not a symbolic link, belonging to the account that runs the program, with no access for group or other. A path naming a directory or another file that is not regular is refused before the lock is taken, so that nothing is written or removed next to it.
- Its directory must exist, belong to the same account, and not be writable by others unless sticky. A directory writable by its group gives a warning.
- These checks apply to every command that takes the lock; `user list` checks the file only.
- Leftover temporary files are removed only by the holder of the lock.
- The threats these checks answer are in [SECURITY.md](SECURITY.md#files).

### Key file

- The file `<file name>.key`, next to the database, holds the key that seals the API keys of the subscriptions: 32 random bytes in base64, on one line.
- It is created, mode 0600 and atomically, at the first subscription saved, and never rewritten; it follows the checks of the database file.
- Without it, or with another one, the keys cannot be opened: the subscriptions must be given their key again. Back it up with the database, and keep the two copies apart when a stolen backup of the database alone must not reveal the keys.
- A database name cannot end with `.key`.

### Directory of the downloads

- The directory `<file name>.downloads`, next to the database, holds the [temporary downloads](#temporary-downloads) of the assistant, one file `<id>-<login>.json` each, mode 0600, written atomically.
- It is created with mode 0700 at the first download; a directory that gives a right to the group or others is refused, and the download fails.
- Its files never need a backup: they expire after `assistant.download_ttl`. It can be deleted, the server stopped.
- A database name cannot end with `.downloads`.

### Lock

- An exclusive lock `<file name>.lock`, next to the database, allows one writer at a time: `timesheet-database.json.lock` for `timesheet-database.json`.
- The commands that take it are listed in [Lock and running server](#lock-and-running-server).
- A command refuses to write while the server runs.

### Backup and restore

- A backup is a copy of the single file; writes are atomic, so a copy is always whole.
- The server must be stopped before a restore: it keeps the database in memory and would overwrite the file.
- The copy is put back with mode 0600, owned by the account that runs the program, in an existing directory owned by that account and not writable by others unless sticky.
- A restored backup brings back the accounts, passwords, profiles and states of its date. Passwords changed since must be changed again, and accounts disabled or deleted since must be disabled or deleted again.
- The sessions of the [session file](#session-file) stay open: `session clear` closes them. A session of a deleted or disabled account, or created before its account, is refused at its next request.
- The procedure is in [OPERATIONS.md](OPERATIONS.md#backup-and-restore).

## SESSION FILE

- The sessions are saved, hashed, in a file named after the database: `<name>.sessions` next to `<name>`, so `timesheet-database.json.sessions` for `timesheet-database.json`, as the lock is `<name>.lock`.
- Its mode is 0600, and it is 8 MiB at most. The bound of 16 384 sessions on `limits.accounts` × `sessions.per_account` keeps it below ([Section sessions](#section-sessions)).
- A request that only refreshes the last use of its session writes it at most once per `sessions.save_interval` (1 minute by default).
- It lets the sessions survive a restart of the server.
- A session file that cannot be read at start, a file with a wrong mode included, is replaced by an empty one, with the warning `sessions not restored`.
- The events that close sessions are in [Closing of sessions](#closing-of-sessions).
- Its content, its writing and the expiry of the sessions are described in [SECURITY.md](SECURITY.md#sessions).

## SERVER CONFIGURATION

The settings of the server live in their own file, apart from the database. Its path has no default: the option `--config` gives it, `etc/timesheet/timesheet.json` from the sources, `/etc/timesheet/timesheet.json` for the packages. Without `--config`, every setting takes its default; `database.path` has no default, so the database is then given by `--database`. The file shipped in the repository, installed as is by `make install` and by the packages, holds the defaults, with the database of the package:

```json
{
  "server": {
    "listen": "127.0.0.1:8080",
    "origins": [
      "http://127.0.0.1:8080",
      "http://localhost:8080"
    ],
    "trusted_proxies": [
      "10.0.0.0/8",
      "172.16.0.0/12",
      "192.168.0.0/16"
    ],
    "timezone": "Europe/Paris",
    "log_level": "info",
    "tls_cert": null,
    "tls_key": null,
    "tls_min_version": "1.2",
    "hsts_max_age": "8760h",
    "read_header_timeout": "5s",
    "read_timeout": "15s",
    "write_timeout": "30s",
    "keep_alive_timeout": "1m",
    "shutdown_timeout": "10s",
    "max_header_bytes": 8192,
    "max_body_bytes": 262144,
    "max_url_bytes": 2048
  },
  "database": {
    "path": "/var/lib/timesheet/timesheet-database.json",
    "max_bytes": 67108864
  },
  "sessions": {
    "idle_timeout": "8h",
    "max_lifetime": "24h",
    "per_account": 64,
    "save_interval": "1m"
  },
  "login": {
    "form_lifetime": "1h",
    "initial_delay": "1s",
    "max_delay": "5m",
    "budget_window": "1m",
    "new_addresses_per_window": 100,
    "tracked_addresses": 4096,
    "forget_after": "1h"
  },
  "passwords": {
    "iterations": 600000
  },
  "limits": {
    "accounts": 200,
    "account_bytes": 1048576,
    "adjustments": 1000
  },
  "assistant": {
    "timeout": "1m",
    "max_rounds": 4,
    "max_tokens": 4096,
    "conversations": 50,
    "exchanges": 20,
    "question_bytes": 4000,
    "download_ttl": "30m",
    "download_sweep": "5m"
  }
}
```

### Layout

The file is written by `config set` (and printed by `config show`) in a fixed layout, which the shipped file follows byte for byte:

- an indentation of 2 spaces;
- one key per line in each object, one element per line in each array;
- `[]` for an empty array;
- the keys in the order of the example;
- a final line break.

A file edited by hand may use any layout: the next `config set` rewrites it in this one.

### Values

- A duration is a string read like `300ms`, `30s`, `5m`, `1h30m` or `8h`: a whole number of milliseconds. It is written in its shortest form, the largest of the units `h`, `m`, `s` and `ms` that divides it exactly: `60s` is written `1m`, `1h30m` is written `90m`.
- A size is a number of bytes.
- `null` stands for no value. It is the value of `tls_cert` and `tls_key` without a certificate pair. For any other key, `null` is read like an absent key.

### Section server

| Key | Format and bounds | Meaning | Default | Option that replaces it |
| --- | --- | --- | --- | --- |
| `listen` | `host:port`, the port being 0 to 65535 or a service name | the listening address | `127.0.0.1:8080` | `--listen` |
| `origins` | `scheme://host[:port]`, all `http` or all `https`, at least one, no path, query or user | the accepted origins: the public URLs of the server | `http://127.0.0.1:8080`, `http://localhost:8080` | `--origin` |
| `trusted_proxies` | IP addresses or prefixes | the proxies whose `X-Forwarded-For` is believed; an empty list trusts none | the RFC 1918 ranges | `--trusted-proxy` |
| `timezone` | an IANA time zone, not `Local` | the zone of [today](#today-and-time-zone) | `Europe/Paris` | none |
| `log_level` | as `--log-level` | the lowest level of the logs of `run` ([Logs](#logs)) | `info` | `--log-level` |
| `tls_cert`, `tls_key` | two paths, or both `null` | the certificate pair of [HTTPS](#https) | `null` | `--tls-cert`, `--tls-key` |
| `tls_min_version` | `1.2` or `1.3` | the lowest TLS version accepted with a certificate pair | `1.2` | none |
| `hsts_max_age` | a duration from 24 h to 17 520 h (2 years) | the `max-age` of `Strict-Transport-Security`, sent with https origins | `8760h` (1 year) | none |
| `read_header_timeout` | a duration from 1 s to 10 min | the time to read the headers of a request | `5s` | none |
| `read_timeout` | same | the time to read a whole request | `15s` | none |
| `write_timeout` | same | the time to write an answer | `30s` | none |
| `keep_alive_timeout` | same | the time an idle connection is kept open | `1m` | none |
| `shutdown_timeout` | a duration from 1 s to 60 s, under the 90 s that systemd waits by default | the time left to the requests in progress when the server stops | `10s` | none |
| `max_header_bytes` | 4096 to 1 048 576 | the size of the request headers; the standard library accepts about 4 KiB more | `8192` | none |
| `max_body_bytes` | 262 144 to 8 388 608 | the size of a request body; the floor holds the largest legitimate form, the grid of a month of 31 days with comments of 500 characters | `262144` | none |
| `max_url_bytes` | 1024 to 65 536 | the length of the URL of a request | `2048` | none |

### Section database

| Key | Format and bounds | Meaning | Default | Option that replaces it |
| --- | --- | --- | --- | --- |
| `path` | `null`, or a file: not ending with `/`, its file name not `.` nor `..` and not ending with `.lock`, `.sessions` or `.tmp` in any case; a relative path is relative to the current directory | the [database file](#database), with its lock and session files next to it | `null`: no default; `/var/lib/timesheet/timesheet-database.json` in the shipped file | `--database` |
| `max_bytes` | 1 048 576 to 1 073 741 824 | the size of the database file, read or written ([Writing](#writing)) | `67108864` (64 MiB) | none |

### Section sessions

| Key | Format and bounds | Meaning | Default |
| --- | --- | --- | --- |
| `idle_timeout` | a duration from 5 min to 7 days | a session without request for this long is closed | `8h` |
| `max_lifetime` | a duration from `idle_timeout` to 30 days | a session is closed this long after the login | `24h` |
| `per_account` | 1 to 1000, and `limits.accounts` × `per_account` at most 16 384 | the open sessions of one account; a new login closes the oldest one beyond | `64` |
| `save_interval` | a duration from 5 s to 1 h | a request that only refreshes the last use of its session writes the [session file](#session-file) at most once in this time; logins, logouts and closings write it at once | `1m` |

- The durations apply to the sessions restored at start too: they are compared when the session is used.
- The bound of 16 384 sessions keeps the session file under its 8 MiB, so that the sessions can always be restored.

### Section login

| Key | Format and bounds | Meaning | Default |
| --- | --- | --- | --- |
| `form_lifetime` | a duration from 1 min to 24 h | the life of the login cookie `ts_login`, counted from the last display of the form in the browser | `1h` |
| `initial_delay` | a duration from 100 ms to 10 s, at most `max_delay` | the wait after a first failure, of the address and of the login; it doubles at each further failure | `1s` |
| `max_delay` | a duration from 1 s to 1 h | the longest wait of an address or a login, and the wait of an address beyond the budget | `5m` |
| `budget_window` | a duration from 10 s to 1 h | the window of the global budget | `1m` |
| `new_addresses_per_window` | 1 to 10 000 | the new addresses that may fail in a window before the next ones wait `max_delay` | `100` |
| `tracked_addresses` | 16 to 65 536 | the addresses followed at once, and the logins followed at once | `4096` |
| `forget_after` | a duration from `max_delay` to 7 days | an address or a login without failure for this long starts again from `initial_delay` | `1h` |

The throttle itself is described in [SECURITY.md](SECURITY.md#login-throttling).

### Section passwords

| Key | Format and bounds | Meaning | Default |
| --- | --- | --- | --- |
| `iterations` | 600 000 to 1 200 000 | the PBKDF2-SHA256 iterations of a new password hash | `600000` |

The existing hashes keep their own count; any count from 1 000 to 1 200 000 is accepted at login ([SECURITY.md](SECURITY.md#passwords)).

### Section limits

| Key | Format and bounds | Meaning | Default |
| --- | --- | --- | --- |
| `accounts` | 1 to 10 000 | the accounts of the database ([Accounts page](#accounts-page)) | `200` |
| `account_bytes` | 65 536 to 16 777 216 | the size of one account record as compact JSON ([Size of an account](#size-of-an-account)) | `1048576` (1 MiB) |
| `adjustments` | 1 to 100 000 | the adjustments of one account | `1000` |

These limits apply only to a write that grows the database: a lowered limit never makes an existing database unreadable, it only refuses the next account, or the next save that would grow an account beyond it.

### Section assistant

| Key | Format and bounds | Meaning | Default |
| --- | --- | --- | --- |
| `timeout` | duration, 5s to 5m | the longest call to an AI provider | `1m` |
| `max_rounds` | 1 to 8 | the rounds of tools of one answer, before a last call without tool | `4` |
| `max_tokens` | 256 to 32 000 | the tokens asked for each call | `4096` |
| `conversations` | 1 to 200 | the conversations kept per account, the oldest dropped | `50` |
| `exchanges` | 1 to 100 | the exchanges kept per conversation, and sent to the model as history | `20` |
| `question_bytes` | 100 to 16 000 | the size of one question | `4000` |
| `download_ttl` | duration, 1m to 24h | the life of a [temporary download](#temporary-downloads); `TIMESHEET_ASSISTANT_DOWNLOAD_TTL` replaces it | `30m` |
| `download_sweep` | duration, 10s to 1h | the period of the job that removes the expired downloads; `TIMESHEET_ASSISTANT_DOWNLOAD_SWEEP` replaces it | `5m` |

A lowered `conversations` or `exchanges` applies at the next question of a conversation; the conversations already kept stay readable.

### Reading

- Every command given `--config` reads the file first, except `help` and `version`: `run`, `init`, `user`, `session clear` and `import` before opening the database, `config show` and `config set` to print or change it. Without `--config`, they take the defaults.
- The file is read strictly: unknown fields and data after the object are refused, and it is 1 MiB at most. As for the database, the standard decoder matches field names without regard to case and keeps the last of repeated keys: the fields of two repeated objects and the keys of two repeated maps are merged, but a key repeated within one map keeps only its last value.
- An invalid file stops the command (exit 1) with the path of the faulty setting, for instance `server.listen: an address host:port is expected` or `sessions.idle_timeout: a duration from 5m to 168h is expected`.
- A file given by `--config` and missing is an error (exit 1), `the server configuration does not exist: <file>`, except for `config set`, which creates it.
- In a file, `server.listen`, `server.origins` and `server.timezone` are mandatory, an absent `server.trusted_proxies` trusts no proxy, and an absent `database.path` names no database. Every other absent key takes its default.
- The file shipped in the repository holds the defaults, with `database.path` set to `/var/lib/timesheet/timesheet-database.json`, the database of the packages.

### Priority

- `--listen` replaces `server.listen`.
- `--origin` replaces `server.origins`.
- `--listen` without `--origin` ignores `server.origins`: the origins are derived from the listener ([Derived origins](#derived-origins)).
- `--trusted-proxy` replaces `server.trusted_proxies`; it cannot empty the list.
- `--tls-cert` and `--tls-key` replace `server.tls_cert` and `server.tls_key`.
- `--database` replaces `database.path`.
- `--log-level` replaces `server.log_level` for `run`.
- Without option, the file applies; without file, the defaults, the database coming from `--database`.
- A change applies at the next start of the server, or at the next command.

### Environment variables

`run` reads two environment variables after the file, which replace its keys:

| Variable | Replaces | Default of the key |
| --- | --- | --- |
| `TIMESHEET_ASSISTANT_DOWNLOAD_TTL` | `assistant.download_ttl` | `30m` |
| `TIMESHEET_ASSISTANT_DOWNLOAD_SWEEP` | `assistant.download_sweep` | `5m` |

- A variable that is absent or empty leaves the key of the file, or its default.
- A value is read like the key and checked with its bounds: a malformed or out-of-bounds value stops `run` (exit 1) with `invalid environment variable` and the path of the key, for instance `assistant.download_ttl: a duration such as 500ms, 30s, 5m or 8h is expected`.
- The server page shows the values in effect; `config show` shows the file, without the variables.
- The other commands do not read them.

### Editing

- The file holds no secret and is never written by the server.
- It is created readable by anyone (mode 0644), keeps the mode, the owner and the group given to it, and can be edited by hand.
- `config set` rewrites it after validating the new values ([`timesheet config show` and `timesheet config set`](#timesheet-config-show-and-timesheet-config-set)).
- The server page shows the settings read at the start of the server, read-only ([Server settings shown](#server-settings-shown)).

### Default trusted proxies

- The default trusted proxies are the private ranges of RFC 1918.
- A reverse proxy on a private network is therefore believed about its client without further setting.
- A server that listens on the loopback only receives no request from these ranges.
- The server configuration of the container image trusts no proxy: it listens on every interface of the container, and Docker forwards the connections to a published port from a private address.
- Narrow the list to the address of the proxy.

## CONFIGURATION TEMPLATE

- `share/timesheet/database-template.json` is the initial `configuration` section: codes, counters and default colours ([Codes of the template](#codes-of-the-template), [Counters of the template](#counters-of-the-template)).
- It is read by `init` only, given by `--template`, strictly, 1 MiB at most, and validated like the configuration.
- Editing it afterwards has no effect on an existing database: codes, counters and default colours are then edited in the [configuration page](#configuration-page).
- The settings of the server are not in the database: see [Server configuration](#server-configuration).

## EXAMPLE DATABASE

- `share/timesheet/database-example.json` is a complete database, to try the application with the data of nearly three years. It is installed next to the configuration template.
- Its configuration is the one of the [configuration template](#configuration-template).
- It holds one account: login `demo`, name « Compte Démo », profile `ADMIN`, password `demonstration`.
- Its tracking start is 2023-11-27; every day is entered from that date to 2026-12-31.
- The months from November 2023 to October 2026 are validated, each on its last day at 23:59:59 in `Europe/Paris`; the later months are open.
- `CP` and `RTT` are enabled with their default amounts, `TPC` is disabled; the complementary part-time days of the history are entered as `RTT`; three adjustments complete the balances of `CP`.
- It was made from a real history, anonymized: the identity is replaced, the comments are reduced to « Vacances », the names of the public holidays and « Absence », and the reasons of the adjustments are neutral. The calendar is kept.
- To use it, copy it with the mode 0600 into a directory of your own and give the copy with `--database` ([OPERATIONS.md](OPERATIONS.md#example-database)).
- Its password is public: it is never served on a network nor used as a real database.
- A test loads it: same configuration as the template, the account `demo` and its password, months validated from November 2023 to October 2026. A change of the [database](#database) schema updates it.

## MIGRATION NOTES

### Server settings moved out of the database

The settings of the server once lived in the database, in a section `configuration.server` with the keys `addr`, `origins`, `trusted_proxy` (one IP address, or `null`) and `timezone`. An earlier version moved them to the [server configuration](#server-configuration) file, without raising the schema `version`, which is still 1.

| Symptom | Cause |
| --- | --- |
| Every command that loads the database fails with `store: malformed database: json: unknown field "server"` (exit 1) | the database still holds `configuration.server` |
| `run` logs `cannot load <database>` with that error and exits with 1 | same |
| A server configuration with the key `trusted_proxy` fails with `json: unknown field "trusted_proxy"` (exit 1) | the key is now the list `trusted_proxies` |

Procedure:

1. Stop the server and back up the database file.
2. Copy `addr`, `origins` and `timezone` from `configuration.server` into the server configuration, by hand or with `config set`.
3. Write the former `trusted_proxy` as the list `trusted_proxies`: `["<address>"]`, or `[]` when it was `null`. A `null` former value trusted no proxy, while the default of a missing file trusts the RFC 1918 ranges.
4. Delete the object `server` from `configuration` in the database, keeping its mode 0600 and its owner.
5. Check that the database loads, for instance with `user list --config=<file>`, then start the server.

### Renamed shipped files

The same version renamed the shipped files, renamed again later:

| Former name | Current name |
| --- | --- |
| `share/timesheet/timesheet-template.json` | `share/timesheet/database-template.json` |
| `share/timesheet/timesheet-database-template.json` | `share/timesheet/database-template.json` |
| `share/timesheet/timesheet-import-example.json` | `share/timesheet/import-template.json` |
| `share/timesheet/timesheet-import-template.json` | `share/timesheet/import-template.json` |

A script or an `--template` option that names a former file fails with `no such file or directory`.

### Former session file

- Sessions are saved in `<database file name>.sessions` ([Session file](#session-file)).
- A file `timesheet-sessions.json` left by an older version is no longer read: its users log in again.
- Delete it, as it holds the CSRF tokens of its sessions.

### Listening address renamed

The key `addr` of the server configuration became `listen`, and the option `--addr` of `run` became `--listen`. The same version added the keys of the other sections ([Server configuration](#server-configuration)); their defaults are the former fixed values, so a file without them keeps the former behaviour.

| Symptom | Cause |
| --- | --- |
| Every command that reads the server configuration, `run` and `config show` included, fails with `json: unknown field "addr"` (exit 1) | the file still holds `addr` |
| `timesheet: flag provided but not defined: -addr` (exit 2) | a script or a unit still passes `--addr` |
| The installation of the Debian package warns `timesheet: /etc/timesheet/timesheet.json cannot be read, no database created (see the migration notes of the specifications)` | the file kept by `dpkg` still holds `addr` |

Procedure:

1. Stop the server.
2. Rename the key `addr` into `listen` in the server configuration, `/etc/timesheet/timesheet.json` for the package. `dpkg` keeps a configuration file changed by hand, so a modified file is not replaced by the new one.
3. Replace `--addr` by `--listen` in the scripts and in the drop-ins of the unit.
4. Check the file with `timesheet config show --config=<file>`, then start the server.

### Database path in the server configuration

The path of the database moved from the default of `--database` into the key `database.path` of the server configuration, whose default is `/var/lib/timesheet/timesheet-database.json`.

- Debian package: nothing to do then. A file without the key took that default, which is the database of the package; since then the key has no default: such a file names no database, and the commands that open it without `--database` are refused ([No default path and no hidden companion file](#no-default-path-and-no-hidden-companion-file)). A drop-in that still passes `--database` keeps working, since the option comes first. The unit of the package now passes only `--config=/etc/timesheet/timesheet.json`: the database comes from that file.
- From the sources: a command run without `--database`, with the configuration of the repository, now opens `/var/lib/timesheet/timesheet-database.json` instead of `var/lib/timesheet/timesheet-database.json` under the current directory, and fails with `store: the database does not exist, run init first` or with a permission error naming that path. Give `--database=var/lib/timesheet/timesheet-database.json`, or start the server with `bin/timesheet.sh`, which passes it.
- The installation script of the package reads the path with `timesheet config show database.path --config=/etc/timesheet/timesheet.json`. When the file cannot be read, it warns and creates no database; fix the file, then run `dpkg-reconfigure timesheet`.

### Zones of public holidays

The zones of public holidays added the key `public_holidays` to the configuration, to the settings of each account and to the import format, without raising the schema `version`, which is still 1.

- Nothing to do: a database without `configuration.public_holidays` follows the zone `fr-metropole`, the former calendar, and an account without `settings.public_holidays` follows the default zone. At the next write, an account gets `"public_holidays": null`, and the configuration `"public_holidays": ""`, still read as `fr-metropole`, until an administrator saves a default zone in the configuration page.
- An older binary refuses a database written since, with `json: unknown field "public_holidays"`: restore the backup taken before the upgrade, or delete the keys by hand.
- An import file with `public_holidays` is refused by an older binary for the same reason.

### No default path and no hidden companion file

The paths lost their defaults, and the files written next to the database lost their leading dot, without raising the schema `version`, which is still 1.

| Before | Now |
| --- | --- |
| `--config` defaulted to `etc/timesheet/timesheet.json` | without `--config`, the defaults apply; `config show` and `config set` require `--config` |
| `init` read `share/timesheet/database-template.json` by default | `init` requires `--template` |
| `database.path` defaulted to `/var/lib/timesheet/timesheet-database.json` | `database.path` has no default: the database comes from `--database` or from `database.path`, and a command that opens it without either is refused |
| a missing server configuration gave the defaults | a file given by `--config` and missing is an error, except for `config set`, which creates it without database |
| `.<name>.lock`, `.<name>.sessions`, `.<name>.<digits>.tmp` | `<name>.lock`, `<name>.sessions`, `<name>.<digits>.tmp`, and since the assistant `<name>.key` |
| a database name could not start with a dot | a database name cannot end with `.lock`, `.sessions`, `.key`, `.downloads` or `.tmp` |

- A script or a unit that opens the database without `--database`, with a configuration that names none or without `--config`, fails with `timesheet: <command>: no database: give --database or set database.path in --config` (exit 2): add `--database`, or `database.path` with `config set database.path <file>`. The unit of the packages gives `--config`, and the file shipped in the repository and in the packages names the database.
- The former session file `.<name>.sessions` is no longer read: its users log in again. Stop the server, then delete it, with the former lock `.<name>.lock`; the session file holds the CSRF tokens of its sessions.
- The former lock `.<name>.lock` and the new `<name>.lock` do not exclude each other: a former server still running does not stop a command of the new version, and its next write would erase what that command wrote. Stop the server before upgrading or rebuilding, and start it again only with the new version.

### Login throttle and cookies

The table of the login throttle got a lower maximum, the cookies a prefix under https, and the container image a server configuration that trusts no proxy, without raising the schema `version`, which is still 1.

- A server configuration with `login.tracked_addresses` above 65 536 is refused with `login.tracked_addresses: an integer from 16 to 65536 is expected` (exit 1): edit the key in the file by hand, 65536 at most; `config show` and `config set` refuse the file until then.
- The volume of the server configuration of a container created by an earlier image keeps that configuration, which trusts the RFC 1918 ranges: run `config set server.trusted_proxies ''` in the container, then restart it ([OPERATIONS.md, Container](OPERATIONS.md#back-up-and-upgrade-the-container)).
- With https origins, the cookies are named `__Host-ts_session` and `__Host-ts_login`: the sessions opened before the upgrade are no longer recognised, and their users log in again. The former cookies expire with the browser session.

### AI assistant

The assistant added `ai_providers` to the configuration and to the configuration template, `assistant` to each account, the key file `<name>.key` and the section `assistant` of the server configuration, without raising the schema `version`, which is still 1.

- Nothing to do: a database without `ai_providers` offers the four providers of the template, and an account without `assistant` has no subscription. At the next write, the configuration gets `"ai_providers": null` and each account `"assistant": null`. A server configuration without the section `assistant` takes its defaults.
- An older binary refuses a database written since, with `json: unknown field "ai_providers"` or `"assistant"`, and a server configuration with the section `assistant`, with `json: unknown field "assistant"`: restore the backup taken before the upgrade, or delete the keys by hand.
- A database that already ends with `.key` or `.downloads` must be renamed.
- The keys `assistant.download_ttl` and `assistant.download_sweep` and the directory `<name>.downloads` came with the temporary downloads: a server configuration without them takes their defaults.
- The assistant calls the providers over https: open the outbound flows to their addresses, or set `HTTPS_PROXY`.

### Rule for future changes

A change of the layout of the database raises its `version` and makes the program read or convert the former version, or adds a note to this section with its symptom and its procedure. The same applies to the server configuration and to the import format.

## LOGS

### Line format

- The server writes its logs on the standard error, one line per event.
- A line is a list of `key=value` pairs starting with `time`, `level` and `msg`.
- `--log-level` sets the lowest level written: `info` by default. For `run`, the default is `server.log_level` of the [server configuration](#server-configuration); the lines written before the file is read use `info`.
- The commands of the command line write their warnings in the same format.

### Access log

Every request that reaches the handlers gives one line `msg=request` at the level `info`, written when the answer is complete, whatever the answer: pages, static files, redirections, refusals of the guards, errors.

```text
time=2026-11-13T16:40:12.345+01:00 level=INFO msg=request client=203.0.113.9 peer=10.0.0.1 user=jdupont method=POST host=timesheet.example path=/dashboard/activity/2026/11 protocol=HTTP/1.1 status=303 bytes=0 duration=5.150547ms agent="Mozilla/5.0 …"
```

| Field | Content |
| --- | --- |
| `client` | the address of the client: the one given by a trusted proxy in `X-Forwarded-For`, otherwise the address of the connection |
| `peer` | the address of the connection: the proxy, behind one |
| `user` | the login of the session when the request reached a page that reads the session; `-` otherwise |
| `method`, `host`, `path`, `protocol` | the method, the `Host` header, the path without its query string, the protocol version |
| `status` | the status code |
| `bytes` | the size of the body written by the page (a `HEAD` counts it although nothing is sent) |
| `duration` | the time taken, as a Go duration (`370.824µs`, `5.150547ms`) |
| `agent` | the `User-Agent` header, cut at 200 bytes |

- A successful login is logged under its login, a failed one under `-`: the login typed is never logged.
- `user` is also `-`, even with a valid session, for static files, `GET /login`, 405 answers and the refusals made before routing (403 origin, 414, 421).
- Requests refused by the parser of the standard library (headers beyond the limit, 431; malformed request, 400) are answered in plain text without a log line.

### Events

| Message | Level | Fields | When |
| --- | --- | --- | --- |
| `starting` | INFO | `version` | `run` starts |
| `listening` | INFO | `address`, `origins`, `database` | the server is ready; `database` is the absolute path of the database file |
| `no administrator account` | WARN | `hint` | the database has no enabled `ADMIN` at start |
| `serving plain HTTP: …` | WARN | | the origins are `http` |
| `the TLS key is readable by other users: give it mode 0600 or 0640` | WARN | `path` | the key of the TLS pair gives a right to others, or a right to its group beyond reading |
| `stopped` | INFO | | the server stopped |
| `server stopped` | ERROR | `error` | the server failed while serving |
| `cannot load <path>`, `cannot lock <path>`, `cannot listen on <address>`, `invalid origins`, `invalid trusted proxies`, `TLS needs https origins`, `TLS setup failed`, `cannot prepare <path>`, `cannot build the site` | ERROR | `error` | `run` cannot start; it exits with 1 |
| `sessions not restored` | WARN | `error` | the session file cannot be read at start |
| `cannot save the sessions` | ERROR | `error` | the session file cannot be written |
| `unreadable session file replaced by an empty one` | WARN | `error` | `user profile`, `user passwd` or `user del` met an unreadable session file |
| `orphan temporary file removed` | WARN | `file` | a temporary file left by an interrupted write was removed |
| `directory is writable by its group` | WARN | `directory` | the directory of the database is group-writable |
| `file written but its directory is not synchronised` | WARN | `error` | the `fsync` of the directory failed after a write |
| `request` | INFO | [Access log](#access-log) | every request |
| `login` | INFO | `login` | a successful login |
| `login failed` | WARN | `address` | a failed login |
| `login throttled` | WARN | `address` | a login refused by the throttle |
| `logout` | INFO | `login` | `POST /logout` |
| `month saved` | INFO | `login`, `month` | a save of the grid |
| `month validated` | INFO | `login`, `month` | a validation |
| `month reopened` | INFO | `login`, `month` | a reopening |
| `adjustments changed` | INFO | `login`, `month`, `change` (`adjusted` or `unadjusted`) | an adjustment added or deleted |
| `settings saved` | INFO | `login` | a save of the settings page |
| `name changed` | INFO | `login`, `by` | a name change, profile page or account page |
| `password changed` | INFO | `login` | a password change in the profile page |
| `import saved` | INFO | `login`, `days`, `adjustments`, `validated` | an import from the data page; the fields are counts |
| `data exported` | INFO | `login` | a download of the data from the data page |
| `data erased` | INFO | `login` | a deletion of the data from the data page |
| `password change refused` | WARN | `login` | a wrong current password, or a concurrent change (409) |
| `all sessions closed` | INFO | `login` | « Déconnecter toutes mes sessions » |
| `account created` | INFO | `login`, `profile`, `by` | an account created from the web |
| `account creation refused` | WARN | `login`, `error` | a creation refused at the database (login taken, `limits.accounts` reached) |
| `profile changed` | INFO | `login`, `profile`, `by` | a profile changed from the web |
| `password reset` | INFO | `login`, `by` | a password reset |
| `account disabled`, `account enabled` | INFO | `login`, `by` | an account disabled or enabled |
| `sessions closed` | INFO | `login`, `by` | the sessions of an account closed from the administration |
| `account deleted` | INFO | `login`, `by` | an account deleted from the web |
| `configuration changed` | INFO | `action`, `by` | a change of the configuration |
| `key file created` | INFO | `file` | the first subscription created the [key file](#key-file) |
| `subscriptions changed` | INFO | `login`, `change` (the notice code) | a subscription added, changed, deleted, made the default, shared, unshared, hidden or shown |
| `assistant asked` | INFO | `login` | a question to the assistant |
| `assistant answered` | INFO | `login`, `provider`, `model`, `input_tokens`, `output_tokens`, `proposal` | an answer of the assistant; `proposal` tells whether it proposed an action |
| `assistant failed` | WARN | `login`, `provider`, `model`, `error` | a failure of the provider; `error` names the status or the network error, never the key nor the body of the answer |
| `conversation deleted`, `conversations deleted` | INFO | `login` | one or every conversation deleted |
| `data exported` | INFO | `login`, `by=assistant` | an export prepared by the tool `export_data` |
| `assistant download` | INFO | `login` | a temporary download read |
| `downloads removed` | INFO | `count` | the job removed expired downloads |
| `invalid environment variable` | ERROR | `error` | `run` cannot start: a variable of [Environment variables](#environment-variables) is invalid; it exits with 1 |
| `cannot save the download`, `cannot read the download` | ERROR | `error` | a temporary download could not be written or read |
| `cannot save the name`, `cannot save the calendar`, `cannot save the settings`, `cannot save the configuration`, `cannot export the data`, `cannot import the data`, `cannot save the subscriptions`, `cannot save the conversations`, `cannot save the answer` | ERROR | `error` | a write failed (500), or the answer of the assistant could not be kept |
| `handler panic` | ERROR | `path`, `panic` | a page failed unexpectedly (500) |
| `template rendering failed` | ERROR | `error` | a page could not be rendered |
| messages of the standard HTTP server | WARN, DEBUG for TLS handshake errors | | errors of connections |

- `by` is the login of the administrator who acted; for `month saved`, `month validated`, `month reopened` and `adjustments changed`, `by=assistant` marks a pending action of the assistant confirmed by the user.
- `address` is the key of the login throttle, not the address of the client: an IPv4 address as `/32`, a public IPv6 address as its `/64`, another IPv6 address as `/128`, and `::/128` for a request from a trusted proxy without a readable `X-Forwarded-For` ([SECURITY.md](SECURITY.md#login-throttling)).
- A password change refused by the throttle (429) logs no event.
- The commands of the command line log no event of their own.

### Never logged

- passwords;
- the login typed at a failed login;
- session identifiers;
- CSRF tokens;
- query strings;
- request bodies;
- the content of the days: codes, comments, reasons;
- the API keys of the subscriptions;
- the questions to the assistant, its answers and the data its tools read;
- the `User-Agent` beyond its first 200 bytes.

## SECURITY REQUIREMENTS

The security model, its reasons and its accepted risks are owned by [SECURITY.md](SECURITY.md). This table lists the requirements and where they are described.

| Topic | Requirement | Details |
| --- | --- | --- |
| Sessions | kept in memory and saved hashed in the [session file](#session-file), so that they survive a restart; sliding expiry, absolute expiry and number per account set in the [section sessions](#section-sessions), 8 h, 24 h and 64 by default | [Sessions](SECURITY.md#sessions) |
| Cookies | the session cookie `ts_session` and the login form cookie `ts_login`, both `HttpOnly` and `SameSite=Strict`; with https origins, `Secure`, the prefix `__Host-` (`__Host-ts_session`, `__Host-ts_login`) and HSTS of `server.hsts_max_age` | [Sessions](SECURITY.md#sessions) |
| Forms | a CSRF token on every form, and an origin check on every unsafe request | [Forms and origins](SECURITY.md#forms-and-origins) |
| Passwords | PBKDF2-SHA256 with `passwords.iterations` iterations, 600 000 at least and by default; at least 12 characters, 6 distinct, no surrounding space, no control, format or line separator character, 256 bytes at most, different from the login whatever the case and, from the profile page, from the current password | [Passwords](SECURITY.md#passwords) |
| Login | throttled per address and per login, with the delays of the [section login](#section-login) | [Login throttling](SECURITY.md#login-throttling) |
| Pages | strict CSP with Trusted Types | [Pages](SECURITY.md#pages) |
| Logs | no secret in the logs ([Never logged](#never-logged)) | [Logs](SECURITY.md#logs) |
| AI assistant | API keys sealed by the [key file](#key-file) and never shown again; calls only to the addresses of the configuration, over https; tools bound to the account of the question; every write confirmed by the user | [AI assistant](SECURITY.md#ai-assistant) |

## LIMITS

The column « Setting » names the key of the [server configuration](#server-configuration) that sets a limit; its value is then the default. « fixed » marks a limit of the code.

| Limit | Value | Setting | When exceeded | Section |
| --- | --- | --- | --- | --- |
| Request body | 256 KiB | `server.max_body_bytes` | 400 | [Refusals](#refusals) |
| URL | 2048 bytes | `server.max_url_bytes` | 414 | [Refusals](#refusals) |
| Request headers | 8 KiB configured; the standard library accepts about 4 KiB more | `server.max_header_bytes` | 431, plain text | [Refusals](#refusals) |
| Server timeouts | reading the headers 5 s, reading the request 15 s, writing the answer 30 s, idle connection 1 min | `server.read_header_timeout`, `server.read_timeout`, `server.write_timeout`, `server.keep_alive_timeout` | the connection is closed | [`timesheet run`](#timesheet-run) |
| Stop of the server | 10 s for the requests in progress | `server.shutdown_timeout` | they are cut | [`timesheet run`](#timesheet-run) |
| Accounts | 200 | `limits.accounts` | 422 « Le nombre maximal de comptes est atteint. », only for a new account; a database with more still loads | [Accounts page](#accounts-page) |
| Data of one account | 1 MiB of compact JSON, 1000 adjustments | `limits.account_bytes`, `limits.adjustments` | 413, only for a save that grows the account | [Size of an account](#size-of-an-account) |
| Database file | 64 MiB | `database.max_bytes` | every write is refused (500), and a larger file is refused at load, so the effective number of accounts depends on their size | [Writing](#writing) |
| Session file | 8 MiB | fixed; `limits.accounts` × `sessions.per_account` is at most 16 384 | the sessions are not restored at start | [Session file](#session-file) |
| Sessions per account | 64 | `sessions.per_account` | the oldest session is closed | [SECURITY.md](SECURITY.md#sessions) |
| Session expiry | 8 h without activity, 24 h at most | `sessions.idle_timeout`, `sessions.max_lifetime` | the session is closed | [SECURITY.md](SECURITY.md#sessions) |
| Login form | 1 hour from its last display in the browser | `login.form_lifetime` | 403 | [Refusals](#refusals) |
| Login throttle | delay of 1 s doubling up to 5 min per address and per login; 4096 addresses and 4096 logins; 100 new addresses a minute; an address or a login forgotten after 1 h without failure | `login.initial_delay`, `login.max_delay`, `login.tracked_addresses`, `login.new_addresses_per_window`, `login.budget_window`, `login.forget_after` | 429 | [SECURITY.md](SECURITY.md#login-throttling) |
| Import file | 8 MiB | fixed | exit 1 | [Import](#import) |
| Configuration template | 1 MiB | fixed | exit 1 | [Configuration template](#configuration-template) |
| Server configuration file | 1 MiB | fixed | exit 1 | [Server configuration](#server-configuration) |
| Password line on the standard input | 4096 bytes | fixed | the rest is not read | [Passwords](#passwords) |
| Password | 12 characters at least, 6 distinct, 256 bytes at most | fixed | 422, exit 1 | [SECURITY.md](SECURITY.md#passwords) |
| Login | 2 to 32 characters | fixed | 422, exit 1 | [Logins](#logins) |
| Name | 60 characters | fixed | 422 | [Free-text rule](#free-text-rule) |
| Comment | 500 characters, one line | fixed | 422 | [Comments](#comments) |
| Adjustment reason, reopening reason | 200 characters, one line | fixed | 422, 409 | [Free-text rule](#free-text-rule) |
| Label of a code or a counter | 60 characters | fixed | 422 | [Fields of a code](#fields-of-a-code) |
| Short label of a counter | 6 characters | fixed | 422 | [Fields of a counter](#fields-of-a-counter) |
| Identifier of a code or a counter | 16 characters | fixed | 422 | [Fields of a code](#fields-of-a-code) |
| Label of a ledger entry | 1000 characters | fixed | the database is refused at load | [Validation at load](#validation-at-load) |
| Activity codes | 1 to 100 | fixed | 422 | [Configuration page](#configuration-page) |
| Counters | 20 | fixed | 422 | [Configuration page](#configuration-page) |
| Adjustment, initial balance | -366 to 366 | fixed | 422 | [Adjustments](#adjustments) |
| Default and own amounts | 0 to 366 | fixed | 422 | [Fields of a counter](#fields-of-a-counter) |
| Quantity | six decimals, a denominator dividing 63 000 000, magnitude 1 000 000 | fixed | refused as invalid | [Quantities](#quantities) |
| Planning horizon | 24 months after the current one | fixed | 409 | [Supported years and planning horizon](#supported-years-and-planning-horizon) |
| Years | 2000 to 2099 | fixed | 404 | [Supported years and planning horizon](#supported-years-and-planning-horizon) |
| `User-Agent` in the access log | 200 bytes | fixed | cut | [Access log](#access-log) |
| Call to an AI provider | 1 min, 4 MiB of answer | `assistant.timeout`; 4 MiB fixed | the answer fails | [Agent and skill](#agent-and-skill) |
| Rounds of tools | 4, then a last call | `assistant.max_rounds` | the model answers without tool | [Agent and skill](#agent-and-skill) |
| Tokens of one call | 4096 | `assistant.max_tokens` | the answer is cut by the provider | [Agent and skill](#agent-and-skill) |
| Conversations, exchanges | 50 per account, 20 per conversation | `assistant.conversations`, `assistant.exchanges` | the oldest is dropped | [Conversations](#conversations) |
| Question | 4000 bytes | `assistant.question_bytes` | 422 | [Questions and answers](#questions-and-answers) |
| Answer | 64 000 bytes | fixed | cut | [Questions and answers](#questions-and-answers) |
| Answers in preparation | one per account | fixed | 409 | [Questions and answers](#questions-and-answers) |
| Temporary download | 30 min, removed every 5 min, 20 per account, 16 MiB | `assistant.download_ttl`, `assistant.download_sweep`, or their environment variables; 20 and 16 MiB fixed | 410, then removed; the oldest removed | [Temporary downloads](#temporary-downloads) |
| Subscriptions, shares | 20 per account, 50 per subscription | fixed | 422 | [Subscriptions](#subscriptions) |
| API key | 512 characters | fixed | 422 | [Subscriptions](#subscriptions) |
| AI providers | 20, 30 models each | fixed | the database is refused at load | [Validation at load](#validation-at-load) |

## NON-FUNCTIONAL REQUIREMENTS

- One static Go binary, standard library only.
- Hand-written HTML, CSS and JavaScript, no framework.
- The site works without JavaScript.
- Every page can be used with the keyboard.
- Text reaches the [AA](#glossary) contrast level.
- The look is sober and professional ([CODING.md](CODING.md#styles)).

## OUT OF SCOPE

- CSV or PDF export.
- The public holidays and the leave rules of another country than France.
- An interface in another language.
- The local public holidays of the territories of the Pacific, outside the public dataset ([Public holidays](#public-holidays)).
- A recurring part-time pattern, or any pre-filling: part-time days are entered by hand.
- Contract modelling: full-time base, rate, days per year.
- Tracking worked days against a contract.
- Validation of months by a manager or an administrator: users validate their own months.
- Reopening a month other than the last validated one.
- Changing the login of an account.
- Automatic backup.
- Editing the AI providers in the pages, or a provider at an address typed by a user.
- An answer of the assistant streamed as it is written, or a write of the assistant without the confirmation of the user.

## SIMPLIFICATIONS

These are choices of the application. Each one gives the way to reach the expected figure when it matters.

- Paid leave is counted in working days, Monday to Friday: 2.08 days a month, about 25 a year (12 × 2.08 = 24.96). A count in jours ouvrables, Saturdays included (2.5 days a month, 30 a year), is not supported: a consuming code is refused on a Saturday.
- The acquisition period of paid leave is 1 June to 31 May in the template; another period is set with the period start of the counter.
- A counter has a single balance: leave acquired and leave being acquired are merged, where a payslip often shows them apart.
- The carry-over without limit of `CP` and the zeroing of `RTT` and `TPC` on 1 January are choices of the template, not rules the application checks.
- The rounding at the end of a period applies to the carried balance, not to the days acquired over the period. It differs from rounding the acquired days whenever the balance carries a fraction that does not come from the accrual: half-days taken, a fractional initial balance or an adjustment. Correct the difference with an adjustment.
- Counters credit the configured amount at each accrual, whatever the absences: paid leave is not reduced for unpaid absences, and `RTT` is not reduced for absences. A code has no nature beyond its counter.
- Sick leave has no specific accrual and no fifteen-month carry over.
- There are no extra splitting days (jours de fractionnement).
- The number of `RTT` days comes from a company agreement; the template's 8/12 a month (8 days a year) is an example, to adapt in the [configuration page](#configuration-page) or with an own amount.
- `TPC` and the arrangement of the [worked example](#worked-example) are the arrangement of one employer, not a legal scheme.
- Every public holiday is treated as not worked: leave cannot be entered on a public holiday that the employer makes worked, such as Whit Monday as the solidarity day. Enter the day with a code that is not consuming, for instance `JF`, and the leave taken as a negative adjustment of its counter.
- The public holidays of the thirteen zones of the public dataset are built in; another calendar, such as the local public holidays of a territory of the Pacific, is not: enter such a day with a code that is not consuming, for instance `JF`.
- Leave is debited only on the half-days entered with a consuming code. When the employer counts the usual day off of a part-time employee within a week of leave, the employee enters the consuming code on that day too.
- Gaps with the payslip are entered as adjustments.

## PERSONAL DATA

### What is sensitive

- The `ABS` code with its comment, and free comments in general, can reveal health or family events.
- The label of a frozen debit or credit keeps the comment of its day, the comment of an `ABS` day included, until the data or the account is deleted ([Ledger entries](#ledger-entries)).

### Inventory

| Data | Where | Who can read it | Kept until | How removed |
| --- | --- | --- | --- | --- |
| Login, name, profile, state, creation and last login times | database | the person; every `ADMIN` (accounts page); the operator | the account is deleted | deletion of the account |
| Password hash | database | the operator | the next password change, or the deletion | deletion of the account |
| Day entries, comments, adjustments and their reasons, user settings | database | the person; every `ADMIN` (read-only views); the operator | the account is deleted, also while it is disabled | the person edits an open month; deletion of the data; deletion of the account |
| Ledger entries with the comments frozen in their labels | database | the person; every `ADMIN`; the operator | the account is deleted, reopenings included | deletion of the data; deletion of the account |
| Sessions: login, creation and last activity | [session file](#session-file) | the operator | the session closes or expires | logout, `session clear` |
| Access log: client and proxy addresses, login, path, user agent, time | logs on the standard error | the operator | as long as the service manager or the file that collects them keeps them | outside the application |
| Events: logins, failed and throttled logins with their address key, account events with the administrator, configuration changes, saves, validations, reopenings, changes of adjustments, settings and names, with their login | logs | the operator | same | outside the application |
| Subscriptions: name, provider, model, sealed API key, shares | database; the key file next to it | the owner, without the key; the accounts it is shared with, without the key; the operator, who can open the keys with the key file | the subscription or the account is deleted | the owner deletes the subscription; deletion of the account |
| Conversations: questions, answers, pending actions | database | the person; the operator | the oldest beyond `assistant.conversations` and `assistant.exchanges` are dropped; the account is deleted | deletion of one or every conversation; deletion of the data; deletion of the account |
| Temporary downloads: the export of the account, with the comments of its days | directory `<name>.downloads` | the person, through its link; the operator | `assistant.download_ttl`, 30 minutes by default, then the job removes it | expiry; deletion of the data; deletion of the account |
| Questions, the context of the account and the data read by the tools: days, comments, balances, adjustments | the AI provider of the subscription | the provider, under its own terms | as the provider keeps them | outside the application |
| Backups of the database | wherever the operator keeps them | whoever holds them | as long as they are kept | outside the application |
| Import files and the spreadsheets they come from | outside the application | whoever holds them | as long as they are kept | outside the application |
| Files downloaded from the data page « Données »: settings, days with their comments, adjustments with their reasons | the computer of the person | the person, and whoever reaches that computer | as long as they are kept | outside the application |

- The database and the backups are not encrypted, except the API keys, sealed by the key file.
- A question to the assistant sends personal data to the provider chosen by the user, comments of `ABS` days included when the tools read them: the page « Mes souscriptions » says it next to the form of a subscription.
- The reads of an `ADMIN` on another account appear in the access log, with the administrator and the page read.
- The logs never hold the content of a day ([Never logged](#never-logged)). The commands of the command line log no event.
- Deleting an account purges neither the backups nor the logs. Restoring a backup brings back the accounts deleted since, which must then be deleted again.
- Data are kept without automatic retention limit.
- The person downloads their own data from the data page ([Export](#export)); an `ADMIN` cannot download the data of another account.
- The application gives no selective erasure: a comment frozen in a validated month is erased only with the whole tracking ([Deletion of the data](#deletion-of-the-data)) or with the account.
- The person who runs the service is responsible for the legal obligations attached to these data.
- The protection of the files is described in [SECURITY.md](SECURITY.md#data-protection).
