# CODING RULES

This document holds the rules that every change of this repository must follow.
It is written for the maintainer, for an AI agent and for a developer who joins the project.
Each rule is enforced at one of three levels: a test run by `make check`, a command run before a commit, or review; the [Summary of the checks](#summary-of-the-checks) says which.
[ARCHITECTURE.md](ARCHITECTURE.md) describes how the code is organised, [SPECIFICATIONS.md](SPECIFICATIONS.md) what it must do, and [SECURITY.md](SECURITY.md) the threats it answers.

## CONTENTS

- [SUMMARY OF THE CHECKS](#summary-of-the-checks)
- [DEPENDENCIES](#dependencies)
- [PACKAGES](#packages)
- [FILES](#files)
- [COMMENTS](#comments)
- [OBJECT-ORIENTED STYLE](#object-oriented-style)
- [NAMING](#naming)
- [SECURITY RULES FOR CODE](#security-rules-for-code)
- [TEMPLATES](#templates)
- [JAVASCRIPT](#javascript)
- [STYLES](#styles)
- [TESTS](#tests)
- [RUNNING ONE TEST](#running-one-test)
- [FORMATTING](#formatting)
- [CHECKS BEFORE A COMMIT](#checks-before-a-commit)
- [CHANGING THE SCHEMA](#changing-the-schema)
- [MAKEFILE](#makefile)
- [PACKAGING](#packaging)
- [DOCUMENTATION](#documentation)
- [COMMITS](#commits)

## SUMMARY OF THE CHECKS

The tests named below live in `src/internal/web/` unless another package is given.

| Rule | Checked by | Section |
| --- | --- | --- |
| No third-party Go module (`src/go.mod` has no `require`) | review | [Dependencies](#dependencies) |
| Whole-line `//` comments in Go are only the allowed ones | `TestGoFilesHaveOnlyAllowedComments` (`source_policy_test.go`, every Go file under `src`, leading tabs included) and the comment command | [Comments](#comments) |
| Trailing and block comments in Go | review | [Comments](#comments) |
| No comment besides the licence header in style sheets, scripts and templates | `TestStylesheetsAvoidImportantAndLiteralSpacing`, `TestJavaScriptHasNoSink`, `TestTemplatesAreSemanticAndInert` (`source_policy_test.go`) | [Comments](#comments) |
| No tab at the start of a line of a Go file | `TestGoFilesHaveOnlyAllowedComments` and the tab command | [Formatting](#formatting) |
| Licence header, with the file name, of templates, style sheets and scripts | `TestWebSourcesStartWithTheLicenceHeader` (`source_policy_test.go`) | [Licence header](#licence-header) |
| Licence header of Go files, the `Makefile` and the scripts; section banners; `End-Of-File` banner | review | [Files](#files) |
| One entity per file | review | [One entity per file](#one-entity-per-file) |
| No `text/template`, no trusted-content type of `html/template` | `TestNoTrustedTemplateContent` (`source_policy_test.go`) | [Security rules for code](#security-rules-for-code) |
| No forbidden JavaScript sink | `TestJavaScriptHasNoSink` | [Security rules for code](#security-rules-for-code) |
| No password, typed login, session identifier or CSRF token in the logs | `TestSecretsNeverReachTheJournal` (`secrets_test.go`) | [Security rules for code](#security-rules-for-code) |
| Only `r.URL.Path` is logged, never `r.URL` or `r.RequestURI` | review | [Security rules for code](#security-rules-for-code) |
| Templates: no `div`, `id`, `for`, inline style, handler, `javascript:`; only the allowed script tags; one `h1` per view | `TestTemplatesAreSemanticAndInert` | [Templates](#templates) |
| Rendered pages: `<html lang="fr">`, `<header>`, `<main>`, `<footer>`, no `div`, one `h1`, no skipped heading level | `TestRenderedPagesHaveASemanticSkeleton` (`semantics_test.go`) | [Templates](#templates) |
| Other semantic rules (order of `head`, elements, `span`, labels) | review | [Templates](#templates) |
| No `!important` in the style sheets | `TestStylesheetsAvoidImportantAndLiteralSpacing` | [Styles](#styles) |
| Contrast of 4.5:1 in both palettes; the two dark blocks identical | `TestBothPalettesKeepReadableContrasts` (`palette_test.go`) | [Palettes](#palettes) |
| No rule resets the ink of every `button` | `TestButtonsKeepTheirInkOnTheAccent` (`palette_test.go`) | [Styles](#styles) |
| Every `day-*` and `code-*` class of the pages has a rule | `TestDayClassesExistInTheStyleSheet` (`semantics_test.go`) | [Styles](#styles) |
| Routes of the code equal the routes of the specifications | `TestRoutesMatchTheSpecifications` (`specifications_test.go`) | [Tests of the documentation](#tests-of-the-documentation) |
| Notice codes have a message and are documented | `TestNoticesMatchTheSpecifications` (`specifications_test.go`) | [Tests of the documentation](#tests-of-the-documentation) |
| Commands of the specifications equal a fixed list and appear in the usage | `TestCommandsMatchTheSpecifications` (`src/internal/app/specifications_test.go`) | [Tests of the documentation](#tests-of-the-documentation) |
| The configuration template equals `NewDefaultConfiguration` | `TestTemplateMirrorsTheDefaultConfiguration` (`src/internal/activity/configuration_test.go`) | [Changing the schema](#changing-the-schema) |
| `etc/timesheet/timesheet.json` is, byte for byte, the encoding of `NewDefaultServerConfiguration` with `database.path` set to the database of the package | `TestShippedServerFileIsTheDefaultConfiguration` (`src/internal/app/server_file_test.go`) | [Changing the schema](#changing-the-schema) |
| The keys of the server configuration equal the keys of the tables of [Server configuration](SPECIFICATIONS.md#server-configuration) | `TestServerKeysMatchTheSpecifications` (`src/internal/app/specifications_test.go`) | [Tests of the documentation](#tests-of-the-documentation) |
| The import template is a valid import file | `TestImportTemplateIsValid` (`src/internal/app/application_test.go`) | [Changing the schema](#changing-the-schema) |
| `ReleaseVersion` equals the first entry of `debian/changelog` | `TestVersionIsTheReleaseOfTheChangelog` (`src/internal/app/application_test.go`) | [Release](#release) |
| Constant format strings | `go vet` | [Tests](#tests) |
| Operator spacing, parentheses, indented `case` | review | [Formatting](#formatting) |
| Documentation kept in sync with the code | review, partly the tests of the documentation | [Documentation](#documentation) |

## DEPENDENCIES

- Direct dependencies are kept to a minimum: Go standard library only, no third-party Go module (`src/go.mod` has no `require`).
- No CSS or JavaScript framework or library, no CDN, no vendored code, no downloaded font: the front end is hand-written HTML, CSS and vanilla JavaScript.
- Any new dependency or framework is first proposed to the author and added only on their explicit decision.
- `src/go.mod` declares `module timesheet` and `go 1.24`, with no `toolchain` line.

## PACKAGES

The roles of the packages and their import direction are owned by [ARCHITECTURE.md, Packages](ARCHITECTURE.md#packages).

- Follow that import direction.
- `web` never imports `app` or `store`: it works on the `web.Repository` interface, implemented by `store.Database` and injected through `SiteDependencies`.

## FILES

### One entity per file

- Each entity lives in its own file named after it in `snake_case` (`month_closing.go`, `ledger_entry.go`, `dashboard_view.go`): every struct, every interface, every record.
- An unexported helper type is allowed in an entity's file only when it serves that entity alone (the records of a view builder, `ledgerWriter` of `Bookkeeper`).
- Exceptions: one `fixtures_test.go` per package for the doubles and helpers shared by several test files, and the platform-specific files.

### Shape of a Go file

Every `.go` file, tests included, has this shape:

```go
/*
 * <name>.go - Copyright (c) 2023-2026 - Olivier Poncet
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

package activity

// ---------------------------------------------------------------------------
// AccrualPeriod declaration
// ---------------------------------------------------------------------------

type AccrualPeriod string

const (
    EveryMonth AccrualPeriod = "month"
    EveryYear  AccrualPeriod = "year"
)

// ---------------------------------------------------------------------------
// AccrualPeriod implementation
// ---------------------------------------------------------------------------

func (p AccrualPeriod) Label() string {
    if p == EveryYear {
        return "par an"
    }
    return "par mois"
}

// ---------------------------------------------------------------------------
// End-Of-File
// ---------------------------------------------------------------------------
```

### Sections of a file

| Section | Holds |
| --- | --- |
| `<Entity> declaration` | The entity type, its constants, its sentinel errors, its interface assertion, and the unexported helper types that serve it alone |
| `<Entity> implementation` | The constructor, then the methods |

| Kind of file | Sections | Banner title |
| --- | --- | --- |
| Entity with a constructor or methods | declaration, implementation | `<Entity> declaration`, `<Entity> implementation` |
| Interface | declaration only | `<Entity> declaration` |
| Data type without constructor or methods (records, `PageModel`, `SiteDependencies`) | declaration only | `<Entity> declaration` |
| `main.go` | one implementation section | `main implementation` |
| Test of an entity | tests | `<Entity> tests` |
| Cross-cutting test | tests | lower-case title naming its subject: `site tests`, `source policy tests`, `secrets tests` |
| `fixtures_test.go` | fixtures | `fixtures` |
| Platform-specific file (`secure_file_unix.go`, `password_reader_linux.go`) | implementation only | `<Entity> implementation (<platform>)` |

Every Go file ends with the `End-Of-File` banner.

### Licence header

#### Text

- The header is the block shown in [Shape of a Go file](#shape-of-a-go-file): copy it from an existing file of the same kind.
- It opens with `/*`, then the line `<file name> - Copyright (c) 2023-2026 - Olivier Poncet`, then the GPL text, the same as in the header of the `Makefile`.
- Each line is prefixed with ` * `; an empty line is ` *`.
- A blank line always separates ` */` from `package`; otherwise the licence becomes the documentation of the package.

#### Build constraint

In a file with a build constraint, `//go:build ...` is the very first line, followed by a blank line and the header.

#### Files that carry it

| Files | Syntax | Checked by |
| --- | --- | --- |
| `.go` files | `/* … */` block, as above | review |
| `Makefile`, `bin/timesheet.sh`, `Dockerfile`, the scripts and the configuration of `.ci/`, `debian/timesheet.service`, `rpm/timesheet.spec` | the same text in `#` comments | review |
| Style sheets (`.css`) and scripts (`.js`) of `internal/web` | the same `/* … */` block at the very top, followed by a blank line | `TestWebSourcesStartWithTheLicenceHeader`, file name included |
| Templates (`.html`) of `internal/web` | the same block inside a template comment, `{{- /* … */ -}}`, on the first lines, which keeps it out of the pages sent | `TestWebSourcesStartWithTheLicenceHeader`, file name included |

`debian/postinst` has section banners but no licence header; `debian/rules`, `.dockerignore` and the skill of the assistant, `src/internal/assistant/assistant.skill.md`, a prompt sent as is to the model, have neither.

### Indentation

- Indentation is 4 spaces, never a tab.
- Only the `Makefile` and `debian/rules` keep tabs, because make requires them.

## COMMENTS

### Rule

- The code is self-explanatory.
- No explanatory comment, no doc comment, no inline or trailing comment, no commented-out code.
- This holds in Go (tests included), JavaScript, CSS, templates and the `Makefile`.
- A name, a constant or a small method says what a comment would have said.

### Comments allowed

| Files | Allowed |
| --- | --- |
| Go | The licence header; the section banners (`// ---…`) and their titles; the directives `//go:build` and `//go:embed` |
| `Makefile`, `bin/timesheet.sh`, `Dockerfile`, the files of `.ci/`, `debian/timesheet.service`, `rpm/timesheet.spec` | The licence header; the section banners with their titles |
| CSS, JavaScript, templates | The licence header and nothing else: no `//`, no other `/* */`, no `<!-- -->`, no other `{{/* */}}`; a style sheet has no section title |

### Checks

This command must print nothing:

```sh
grep -rnE '^\s*//' --include='*.go' src | grep -vE '//go:(build|embed)|// -{10,}$|// .+ (declaration|implementation|tests)( \(.+\))?$|// (End-Of-File|fixtures)$'
```

It accepts five kinds of whole-line comment: the two directives, a banner line, a section title ending with `declaration`, `implementation` or `tests` (with an optional platform in parentheses), and the titles `End-Of-File` and `fixtures`. It does not tell a section title from a comment that ends with the same word: check the titles by review.

| Check | Covers | Left to review |
| --- | --- | --- |
| The command above | Whole-line `//` comments in Go | Trailing and block comments in Go |
| `TestGoFilesHaveOnlyAllowedComments` | The same whole-line `//` comments, and the absence of tabs | |
| `TestStylesheetsAvoidImportantAndLiteralSpacing` | Any `/*` after the header of a style sheet | |
| `TestJavaScriptHasNoSink` | Any `//` or `/* */` after the header of a script | |
| `TestTemplatesAreSemanticAndInert` | `<!--` and `{{/*` after the header of a template | |

## OBJECT-ORIENTED STYLE

### Entities

- Entities are structs with unexported fields.
- They are created by a constructor `NewXxx(deps...)`, which returns a pointer when the struct holds a lock or a state.
- Their behaviour is carried by methods.

### Records

- Data that is persisted or rendered is a record with exported, tagged fields: `activity.Calendar`, `activity.Configuration`, `account.Account`, the view records of `web`.
- A record may carry pure lookup or validation methods (`Configuration.Validate`, `Calendar.IsValidated`).

### Value types

Value types with behaviour (`Quantity`, `Date`, `Month`, `PeriodStart`) have unexported fields and implement `encoding.TextMarshaler` and `encoding.TextUnmarshaler`.

### Dependencies between entities

- A dependency is injected as an interface declared by its consumer (`web.Repository`).
- An interface shared by several packages is declared by the domain (`activity.Clock`).
- Concrete standard library types are accepted: `*slog.Logger`, `io.Reader`, `io.Writer`, `fs.FS`, `*http.Client`.

### Free functions, closures and package variables

| Kind | Allowed |
| --- | --- |
| Free functions | `main`, the constructors `NewXxx`, the `TestXxx` functions, unexported helpers in `_test.go` files |
| Closures | In tests, and as adapters that carry no logic |
| Package-level variables | Sentinel errors `ErrXxx`, constants, interface assertions `var _ I = (*T)(nil)`, the `embed.FS` of `web` exposed by `NewEmbeddedAssets()`, the embedded skill of `assistant` exposed by `NewSkill()` |

- Sort with `slices.SortFunc(xs, r.compare)`, a method as the comparison.
- Declare a command-line option that needs its own parsing with a `flag.Value` type (`OptionList`).

### Quantities and free text

- Quantities of days are exact rationals (`activity.Quantity`), never floats; their denominators divide `QuantityBase`.
- Free text coming from users or files (labels, comments, reasons) is checked with `activity.TextPolicy`.

### main

`main()` does only this:

1. Create the signal context.
2. Build the `Application` with `WithSignals(stop)`; every command but `run` restores the default signal handling, so that one interrupt stops it.
3. Call `Run(ctx)`.
4. Stop the signal handling, also as soon as the context ends, through `context.AfterFunc`.
5. Exit with `os.Exit(code)`.

## NAMING

### Vocabulary

Name the code with these words; their meaning in the domain is in [SPECIFICATIONS.md, Glossary](SPECIFICATIONS.md#glossary).

| Word | Entity it names |
| --- | --- |
| Account | `account.Account` |
| Login | `account.Login` |
| Profile | `account.Profile` |
| Calendar | `activity.Calendar` |
| DayEntry | `activity.DayEntry` |
| HalfDay | `activity.HalfDay` |
| ActivityCode | `activity.ActivityCode` |
| CounterRule | `activity.CounterRule` |
| LedgerEntry | `activity.LedgerEntry` |
| Journal | `activity.Journal` |
| Bookkeeper | `activity.Bookkeeper` |
| MonthClosing | `activity.MonthClosing` |
| Statement | `activity.CounterStatement`, `web.StatementView` |
| Position | `activity.Position`, `web.PositionView` |
| Summary | `activity.Summary`, `web.SummaryView` |
| Adjustment | `activity.Adjustment` |
| Workspace | `web.Workspace` |
| Subscription | `assistant.Subscription`: an API key of an AI provider |
| Conversation | `assistant.Conversation` |
| Pending action | `assistant.PendingAction`, planned by `web.ActionPlanner` |

### Sentinel errors

- Sentinel errors are named `ErrXxx`.
- They live in the file of the entity that returns them (the account errors in `account.go`).
- An error that crosses the `web.Repository` port lives in the file of the record it concerns (`ErrStaleRevision` in `calendar.go`).

## SECURITY RULES FOR CODE

The reasons behind these rules are in [SECURITY.md](SECURITY.md).

| Rule | Checked by |
| --- | --- |
| All HTML goes through `html/template`. Never use `text/template`, nor the trusted-content types of `html/template`: `HTML`, `HTMLAttr`, `JS`, `JSStr`, `URL`, `CSS`, `Srcset` | `TestNoTrustedTemplateContent`, which parses every Go source file under `src/internal` |
| Templates contain no inline script, no inline style, no `style=` and no `on*=` attribute; see [Scripts in a template](#scripts-in-a-template) | `TestTemplatesAreSemanticAndInert` |
| JavaScript uses the DOM API and `textContent` only (forbidden sinks below) | `TestJavaScriptHasNoSink` |
| Secrets are compared in constant time (`subtle.ConstantTimeCompare`); passwords are hashed with PBKDF2 (Password-Based Key Derivation Function 2, `account.PasswordHash`) | review |
| No secret ever reaches the logs: no password, no login typed at a failed login, no session identifier, no CSRF (cross-site request forgery) token | `TestSecretsNeverReachTheJournal`, after a failed and a successful login |
| Only `r.URL.Path` may be logged, never `r.URL` or `r.RequestURI` | review |
| Redirections only target internal paths built by the code (from parsed numbers or validated logins), never a URL taken from the request, with no query string: the `notice` code of the action stays in the session (`Renderer.Redirect`) | review; `TestNoticesFollowTheActionOnceWithoutAnyParameter` for the notice |
| Every state-changing form carries the CSRF token of the session (`csrf`), checked by `AuthGuard`; the login form uses a double-submitted token (the same token in a cookie and in the form) | review |
| Every route of `/dashboard/` and `/admin/` is registered in `site.go` with `guard.Require(user, …)` or `guard.Require(root, …)`, which also checks the `csrf` field of every POST; there is no guard by path prefix, so a route registered without it is open to anyone | review |

The full list of what is never logged is owned by [SPECIFICATIONS.md, Logs](SPECIFICATIONS.md#logs).

Forbidden JavaScript sinks:

- `innerHTML`, `outerHTML`, `insertAdjacentHTML`;
- `document.write`;
- `eval`, `Function(`;
- `javascript:` URLs;
- `setAttribute('style', ...)` and `setAttribute('on...', ...)`.

## TEMPLATES

### Files

| File | Defines |
| --- | --- |
| `templates/layout.html` | The skeleton, the footer, and the shared templates `brand`, `user-menu` and `theme-toggle` |
| `templates/partials/public.html` | `header`, for the public pages |
| `templates/partials/dashboard.html` | `header` and `main-links`, for the pages of a connected user |
| `templates/partials/admin.html` | `header` and `main-links`, for the administration pages |
| One template per view, named in `snake_case` (`admin_settings.html`) | `title`, `content`, and optionally `description` and `scripts` |

The table of pages, routes and templates is in [ARCHITECTURE.md, Pages and the code that serves them](ARCHITECTURE.md#pages-and-the-code-that-serves-them).

### Document skeleton

- The markup is indented by 4 spaces.
- Empty elements are written without ` />`.
- The root is `<html lang="fr">`; `data-theme` is set by `chrome_page.js` only.
- `head` holds, in this order:
  1. `title`;
  2. `meta charset`;
  3. `meta viewport` (`width=device-width, initial-scale=1`);
  4. `meta color-scheme`;
  5. `meta description`;
  6. `meta author`;
  7. `meta http-equiv="refresh"`, only while `PageModel.Refresh` is set (a conversation waiting for its answer);
  8. the icon;
  9. the style sheets.
- `body` holds directly `<header>`, `<main>` and `<footer>`, then the scripts.

### Semantic elements

Pages are as semantic as possible.

| Content | Element |
| --- | --- |
| Title of a view | One `h1` per view |
| Headings | No skipped level |
| Content | Titled `<section>` |
| Lists | `ul`, `ol`, `dl` |
| Forms | `<form>` with `<fieldset>` and `<legend>` |
| Actions | `<button>` |
| Navigation | `<a>` inside `<nav aria-label>` |
| Dates | `<time datetime>` |
| Data | `<table>` with `<caption>`, `<thead>`, `<tfoot>` when the table has totals, and `<th scope>` |
| Current state | `aria-current` |
| Messages | `<p role="status">` or `<p role="alert">` |
| Details after an alert | A plain `<ul>`: a role is only given to an element that accepts it |
| Dialog | `<dialog>` |
| Details | `<details>` and `<summary>` |
| Help | `<abbr title>` |

### Forbidden markup

- No `div` at all.
- `span` only for:
  - the name of a field (`field-name`);
  - the decorative initials of the user menu (`<span class="avatar" aria-hidden="true">`);
  - the name in the brand link, hidden on narrow windows.
- No `id`, and no attribute that names one (`for`, `aria-labelledby`, `aria-describedby`).
- A form field is a `<label class="field">` that wraps its `<span class="field-name">` and its control, or a control named by `aria-label` inside a table.

### Scripts in a template

The only scripts allowed are:

- empty `<script type="module" src="/static/js/<page>_page.js"></script>` tags;
- `<script type="application/json">` blocks holding a single value.

### Checks

| Test | Checks |
| --- | --- |
| `TestTemplatesAreSemanticAndInert` (`source_policy_test.go`) | In the template sources: `div`, `<style`, `style=`, `id=`, `for=`, `on*=` handlers, comments, `javascript:`, script tags other than the two allowed, one `h1` per view |
| `TestRenderedPagesHaveASemanticSkeleton` (`semantics_test.go`) | In the main rendered pages: `<html lang="fr"`, `<header>`, `<main>`, `<footer>`, no `div`, a single `h1`, no skipped heading level |
| review | The order of `head`, the elements of the table above, the use of `span`, the labels |

### Language of the interface

- Pages work without JavaScript; a script only enhances a page.
- Texts of the web pages are in French and never use the em dash, nor an en dash used as a dash.
- The command line (help, prompts, messages, import report) is in English, like the logs and the errors; it keeps in French only the texts it shares with the pages (labels of codes and counters, refusals of the grid).

## JAVASCRIPT

- Scripts are ES modules, loaded by `<script type="module" src="/static/js/<page>_page.js">`.
- One class per file, the file named in `snake_case`.
- An entry point is named `*_page.js` and ends with `new XxxPage(document).start()`.
- That call is the only use of the global `document`.
- Behaviour lives in classes.
- Arrow functions are only adapters for listeners.
- No transpilation, no library.
- Recent browsers only.

## STYLES

### Style sheets

| File | Content |
| --- | --- |
| `static/css/theme.css` | Design tokens as custom properties, base elements, page layout, responsive rules, `prefers-reduced-motion` |
| `static/css/components.css` | The components, in the order below |
| `/dashboard/colors.css` | The colours of the day types chosen by the users, generated from validated values only |

Both static style sheets are hand-written.

#### Order of components.css

1. app bar
2. user menu
3. page header
4. cards
5. messages
6. buttons
7. forms
8. period navigation
9. tables
10. status
11. positions
12. login
13. home
14. footer
15. responsive
16. days
17. month
18. summary
19. calendar
20. facts
21. accounts
22. assistant (subscriptions, login suggestions, exchanges, question, answer, pending action, question form)

The rules of a same subject follow each other, without any title comment: the selectors name what they style. A narrow-window rule of a subject placed after « responsive » stays in a media block at the end of that subject, since a later rule of the same specificity would override it.

### Selectors

| Kind | Rule | Examples |
| --- | --- | --- |
| Elements, structure and states | Target them first | `nav a`, `table caption`, `fieldset legend`, `th[scope="row"]`, `[aria-current="page"]`, `[role="alert"]`, `:disabled`, `:invalid`, `:focus-visible` |
| Classes | Only where the element is not enough, named after what the content is, never after the look or the layout | `month-grid`, `summary-table`, `legend`, `day-missing`, `code-ABS`, `page-header`, `user-menu`, `access`, `positions`, `status` |
| Links that act as buttons | Styled by their place | `.actions a`, `.period-navigation a` |
| Button of a file field | Styled like the buttons, through its pseudo-element | `input[type="file"]::file-selector-button` |
| Derived states | Class prefixed with `is-` | `is-weekend`, `is-holiday`, `is-today`, `is-missing`, `is-provisional`, `is-negative` |

- No `!important`.
- Low specificity.
- Layout with `grid` and `flex` on the semantic elements, no presentational wrapper.
- No rule sets `color: inherit` on every `button`, so that buttons keep their ink on the accent.

### Look

The look is sober and professional:

- a dark slate header bar on top;
- a neutral palette, with a single slate blue accent for links, primary buttons and focus, and desaturated states;
- borders of 1 px;
- the system font stack;
- tabular figures in tables;
- text columns left-aligned, number columns right-aligned, each header aligned like its cells;
- radii of 8 px for controls and 12 px for cards;
- a light `0 1px 3px` shadow on cards, a deeper one on the user menu, inset shadows to mark states;
- a contained width of 1200 px;
- a visible focus.

### Palettes

- The two palettes use the same tokens.
- The light palette is on `:root`.
- The dark palette is under `:root[data-theme="dark"]`, and repeated under `@media (prefers-color-scheme: dark)` on `:root:not([data-theme="light"])`.
- Components use the tokens; a literal colour is allowed only where it is the same in both themes (the brand on the header bar, the banner of the home page).
- `TestBothPalettesKeepReadableContrasts` checks that both palettes keep a contrast of 4.5:1 on their text and background pairs, and that the two dark blocks are identical.

| Token | Light | Dark |
| --- | --- | --- |
| `--background` | `#f4f6f9` | `#11161c` |
| `--surface` | `#ffffff` | `#1a2129` |
| `--surface-raised` | `#ffffff` | `#222b35` |
| `--surface-sunken` | `#eef1f4` | `#151b22` |
| `--ink` | `#1f2933` | `#e6ebf0` |
| `--ink-soft` (secondary text) | `#52606d` | `#9aa7b4` |
| `--border` | `#e1e6ec` | `#2c3640` |
| `--border-strong` | `#c5ced8` | `#3d4955` |
| `--accent` | `#2f5d8a` | `#7fa9d6` |
| `--accent-hover` | `#234766` | `#a3c2e3` |
| `--accent-soft` | `#e6eef6` | `#1f3347` |
| `--on-accent` | `#ffffff` | `#11161c` |
| `--success` | `#276b43` | `#5fb385` |
| `--success-soft` | `#e8f3ec` | `#16291f` |
| `--warning` | `#8a5c00` | `#d9a441` |
| `--warning-soft` | `#fdf4e3` | `#2e2512` |
| `--danger` | `#b42318` | `#f08a80` |
| `--danger-soft` | `#fdecea` | `#3a1c19` |
| `--on-danger` | `#ffffff` | `#11161c` |
| `--bar` (header bar) | `#1f2933` | `#0b0f14` |
| `--bar-ink` | `#e6ebf0` | `#e6ebf0` |
| `--bar-ink-soft` | `#9aa7b4` | `#9aa7b4` |
| `--bar-hover` | `#2c3a47` | `#1c242d` |
| `--bar-accent` | `#7fa9d6` | `#7fa9d6` |
| `--footer` | `#e9edf2` | `#0d1218` |

### Colours of the codes

- The default colours of the codes are owned by [SPECIFICATIONS.md](SPECIFICATIONS.md): `#33d17a`, `#3584e4`, `#9141ac`, `#e01b24`, `#9a9996` and `#ff7800`.
- A code cell takes a black or a white ink, whichever contrasts most with its colour, which keeps every colour above 4.5:1 (WCAG AA level).
- `TestDayClassesExistInTheStyleSheet` checks that every `day-*` class of the pages has a rule in `components.css` and every `code-*` class a rule in `/dashboard/colors.css`.

### Icons

- Icons are hand-made SVG.
- Icon files live under `static/img/icons/`: logo, favicon, theme icons, burger menu.
- An icon that takes the text colour is inline `<svg aria-hidden="true">` in a template (the action buttons of `/admin/accounts` and the login link of the public header).
- Never a `data:` URI.

## TESTS

### Naming and fixtures

- Behaviour is tested by a `_test.go` file named after its entity.
- Cross-cutting tests have their own files: `site_test.go`, `chrome_test.go`, `palette_test.go`, `source_policy_test.go`, `semantics_test.go`, `secrets_test.go`, `specifications_test.go`.
- A package whose test files share doubles or helpers keeps them in one `fixtures_test.go`.

### Doubles

- The clock is injected (`activity.ManualClock`).
- Web tests start from `newTestSite(t)` of `src/internal/web/fixtures_test.go`: accounts `root` (`ADMIN`) and `demo` (`USER`) tracked from September 2026, clock at 2026-11-08 10:00 UTC; `client().login(name)`, `get`, `post`, `page` and `token` drive the pages.
- `html/template` escapes an apostrophe inserted by an action (`{{.Label}}`, a notice) as `&#39;`, while the literal text of a template keeps it: a test that looks for an inserted text with an apostrophe uses the escaped form or another text.
- Web tests use `httptest` and a database in `t.TempDir()`.
- The assistant is tested without network: `newAssistantSite(t, replies...)` starts an `httptest` TLS server that answers the given OpenAI-format replies in order (`openAIText`, `openAICall`), points every provider of the configuration at it and gives its client to the site; `subscribe` and `ask` drive the profile and the assistant pages. The provider clients of `assistant` are tested the same way, one test per format.
- Password hashing uses `account.MinIterations` in tests: `Application.WithIterations` and the field `Iterations` of `SiteDependencies` take precedence over `passwords.iterations`, whose floor is 600 000.
- A test that opens a database always gives its path (`--database` or `store.NewDatabase`): no path has a default, and the configuration of the repository names `/var/lib/timesheet/timesheet-database.json`. Without `--config`, a command takes the default settings, which name no database; `newTestDirectory` also writes `timesheet.json` next to the database, naming it, for the tests that give `--config`.
- `go vet` rejects non-constant format strings: always use a constant format.

### Tests of the documentation

Four tests read [SPECIFICATIONS.md](SPECIFICATIONS.md); update the code, the specifications and these lists together.

| Test | File | Compares | What to update |
| --- | --- | --- | --- |
| `TestRoutesMatchTheSpecifications` | `src/internal/web/specifications_test.go` | The method and pattern of every registered route with the rows of [Routes](SPECIFICATIONS.md#routes) | The specifications only: the routes come from the code |
| `TestNoticesMatchTheSpecifications` | `src/internal/web/specifications_test.go` | A fixed list of notice codes: each must have a message in `Notices` and appear as `` `code` `` in the specifications | Add a new notice code to the list |
| `TestServerKeysMatchTheSpecifications` | `src/internal/app/specifications_test.go` | The keys of `ServerConfiguration` with the first column of the tables `### Section <name>` of [Server configuration](SPECIFICATIONS.md#server-configuration), prefixed with the section | The specifications only: the keys come from the code |
| `TestCommandsMatchTheSpecifications` | `src/internal/app/specifications_test.go` | The `known` list of commands with the `timesheet ` lines of [Command line](SPECIFICATIONS.md#command-line), and checks that the usage lists each | Add a new command to `known`; a new family of subcommands also needs the `switch` of the test (today `config`, `user`, `session`) |

The command list is compared with the specifications and the usage, not with the dispatcher of `Application`: a command added to the code and forgotten in both still passes.

### Data

The spreadsheet the summary is modelled on, and any import file made from personal data, never enter the repository: import tests build synthetic import files.

### What no test checks

- No test checks the licence header of the Go files, of the `Makefile` and of the scripts, nor the section banners: check by hand that every file starts with its header and that every Go file ends with the `End-Of-File` banner.
- The header of the templates, style sheets and scripts is checked by `TestWebSourcesStartWithTheLicenceHeader`.
- `TestRenderedPagesHaveASemanticSkeleton` checks only the paths it lists, and accepts a 403 or a 404 there: add every new page to its list, and check its status in a test of its own.

## RUNNING ONE TEST

The `make` targets set a private Go cache; a direct `go` command uses your own Go cache and toolchain settings unless you prefix it, from the repository root:

```sh
GOCACHE="$PWD/.cache" GOTOOLCHAIN=local go -C src test -race -run TestMonthLifecycle ./internal/web/
GOCACHE="$PWD/.cache" GOTOOLCHAIN=local go -C src test ./internal/activity/
```

- `-run` takes a regular expression on the test names.
- `-race` needs gcc and `CGO_ENABLED=1`, which is the default when gcc is present.

## FORMATTING

### Layout

- Write Go code by hand in the gofmt layout (alignment, braces, blank lines), with three exceptions: the indentation of 4 spaces, the indented `case`, and the operator spacing.
- Never run `gofmt`, `go fmt` or a format-on-save: they put tabs back, squeeze operators and move `case` back.
- If tabs slip in, convert them:

  ```sh
  sed -E -i -e ':a' -e 's/^( *)\t/\1    /' -e 'ta' <files>
  ```

- Never start a line of a multi-line raw string literal with a tab.

### Expressions

These rules hold in Go and JavaScript.

| Rule | Example |
| --- | --- |
| Every binary operator has one space on each side | `m.Days() - 1`, `"/" + name` |
| Unary operators stay attached | `-amount`, `!ok` |
| An operand that is itself a binary expression of another precedence is written in parentheses | `(a * b) + c`, `(err != nil) \|\| ok` |
| Operators of the same precedence chain without parentheses | `a + b - c` |

### switch and select

`case` and `default` clauses are indented one level inside their `switch` or `select`, and their statements one more level:

```go
switch fields[0] {
    case "help":
        documented = append(documented, "help", "version")
    default:
        documented = append(documented, fields[0])
}
```

## CHECKS BEFORE A COMMIT

Run these from the repository root; all must pass before every commit.

| Check | Command | Expected result |
| --- | --- | --- |
| Tests | `make check` (runs `go test -race ./...`, needs gcc) | ends with `=== check ok ===` |
| Vet | `GOCACHE="$PWD/.cache" GOTOOLCHAIN=local go -C src vet ./...` | prints nothing |
| Tabs | `grep -rlP '^\t' --include='*.go' src` | prints nothing |
| Comments | the command of [Comments, Checks](#checks) | prints nothing |

`go vet` is not part of `make check`. Without the prefix, it uses your own Go cache and toolchain settings.

## CHANGING THE SCHEMA

The schema of the database is specified in [SPECIFICATIONS.md, Database](SPECIFICATIONS.md#database).
What the code does at load:

- `store.DatabaseVersion` is `1`; a file with a higher version is refused (`ErrUnsupportedVersion`), any other version is refused as malformed.
- Decoding refuses unknown fields: removing or renaming a field makes the existing databases unreadable.
- A field absent from an existing file decodes to its zero value, then the whole record is validated: the zero value of a new field must pass the validation, or the existing databases are refused.
- There is no migration code.

Steps of a change:

1. Update [SPECIFICATIONS.md, Database](SPECIFICATIONS.md#database) and write how existing databases are handled in [Migration notes](SPECIFICATIONS.md#migration-notes).
2. Change the records in `internal/activity`, `internal/account` or `internal/store` and their validation.
3. When the configuration changes, update `share/timesheet/database-template.json` and `NewDefaultConfiguration` together (`TestTemplateMirrorsTheDefaultConfiguration`).
4. When the server configuration changes, update the settings of `internal/activity` (defaults, bounds, `ServerConfiguration.keys`), regenerate `etc/timesheet/timesheet.json` with `bin/timesheet config set database.path /var/lib/timesheet/timesheet-database.json --config=<new file>`, then copy that new file (`TestShippedServerFileIsTheDefaultConfiguration`), and add the key to its table in [Server configuration](SPECIFICATIONS.md#server-configuration) (`TestServerKeysMatchTheSpecifications`). A new key has a default and bounds; a key renamed, removed or made mandatory needs a migration note, since the file is decoded strictly.
5. When the database schema or the configuration changes, update `share/timesheet/database-example.json` so that it still loads (`TestDatabaseExampleIsValid`).
6. When the import format changes, update the import code, `share/timesheet/import-template.json` (`TestImportTemplateIsValid`) and [SPECIFICATIONS.md, Import](SPECIFICATIONS.md#import).

## MAKEFILE

### Layout

- The licence header comes first.
- Then, between `# ---` banners, the single section `global targets`, and the `End-Of-File` banner.

### Targets

- Each target only runs its script of `.ci/`, from the repository root.
- Every target but `all` ends with `@echo "=== $@ ok ==="`.
- Test targets are named `check*`.

| Target | Script | What it does |
| --- | --- | --- |
| `all` | none | Same as `build` |
| `build` | `make-build.sh` | Compiles a static binary (`CGO_ENABLED=0 go build -trimpath`) straight into `bin/timesheet` |
| `check` | `make-check.sh` | Runs `go test -race ./...` with `CGO_ENABLED=1`; needs gcc |
| `install` | `make-install.sh`, with `DESTDIR` | Copies `bin/timesheet` to `$(DESTDIR)/usr/bin` (mode 755), the configuration template, the import template and the example database to `$(DESTDIR)/usr/share/timesheet` (mode 644), and the server configuration to `$(DESTDIR)/etc/timesheet` (mode 644), replacing the one already there. It does not depend on `build`, so that an installation as root never compiles, and it fails when the binary is missing |
| `clean` | `make-clean.sh` | Runs `go clean` on the packages, empties `.cache/` (compiled packages and test results, so the next build starts from scratch) and removes the binary |
| `docker` | `make-docker.sh` | Builds the container image `timesheet` with `docker build` from the `Dockerfile` of the repository root; it does not run the tests ([Container image](#container-image)) |
| `mrproper` | none | Runs `git clean -f -d -x`, which deletes every untracked and ignored file, the database under `var/lib/timesheet/` included |

### Go environment

The scripts that call `go` (`make-build.sh`, `make-check.sh`, `make-clean.sh`) export their Go variables in a `go environment` section.

| Variable | Value | Reason |
| --- | --- | --- |
| `GOCACHE` | `.cache/` at the root of the repository, ignored by git | The build never writes outside the tree |
| `GOTOOLCHAIN` | `local` | Go never downloads another toolchain |
| `CGO_ENABLED` | `0` for build and clean, `1` for check | A static binary; the race detector needs cgo |

## PACKAGING

### Package

- The packaging lives in `debian/` for the Debian package and in `rpm/` for the RPM package.
- `debian/rules` only calls `dh`, and skips `dh_dwz`, which fails on a Go binary.
- The package takes its files from `make install`: the binary, the configuration template, the import template, the example database and the server configuration `/etc/timesheet/timesheet.json`, the file of the repository as is, which debhelper registers as a configuration file.
- `debian/postinst` reads the database path with `timesheet config show database.path --config=/etc/timesheet/timesheet.json`; it never fails on an unreadable configuration, a configuration without database or a failed `init`, and creates the database only at a plain absolute path inside `/var/lib/timesheet`.
- The unit runs `timesheet run --config=/etc/timesheet/timesheet.json`: the database comes from that file.
- `rpm/timesheet.spec` takes its version from the macro `ts_version`, which the build script gives, and refuses to build without it; it builds with `make`, tests with `make check` and installs with `make install`, the same files as the Debian package, plus the unit `debian/timesheet.service`; it builds no package of debug symbols.
- Its `%post` is `debian/postinst` transposed: the same account, directory and database, the same warnings, and it never fails; its `%preun` and `%postun` are the systemd macros (`%systemd_*`, `%service_*` on openSUSE), which neither enable nor start the service.
- It marks `/etc/timesheet/timesheet.json` `%config(noreplace)`, declares the tools of `%post` as `Requires(post)`, and keeps the names of openSUSE (`go`, `timezone`, `shadow`, `glibc`, `%service_*`) under `%if 0%{?suse_version}`, a variant no build checks.
- How to build the packages is described in [BUILD.md](BUILD.md#build-the-debian-package) and [BUILD.md](BUILD.md#build-the-rpm-package), how to install and run them in [OPERATIONS.md](OPERATIONS.md#debian-package) and [OPERATIONS.md](OPERATIONS.md#rpm-package).

### Container image

- `Dockerfile`, at the repository root, has two stages, `golang:1.27-alpine3.23` then `alpine:3.22`, each image pinned to its version; the Go version of the build stage is a supported release of Go, raised when it leaves support, while `go.mod` keeps `go 1.24`.
- The build stage installs `tzdata`, since `config set` validates `server.timezone` and the binary does not embed the time zones, runs `.ci/make-build.sh`, then `.ci/make-install.sh` with `DESTDIR=/stage`, adds `.ci/timesheet-entrypoint.sh` as `/usr/bin/timesheet-entrypoint.sh`, and sets `server.listen` to `0.0.0.0:8080` and empties `server.trusted_proxies` with `config set`: the image installs the same files as the package.
- The runtime stage installs `tzdata`, since the binary does not embed the time zones, and creates the account `timesheet`, uid and gid 10001; it runs as that account, with the volumes `/var/lib/timesheet` and `/etc/timesheet`.
- `.dockerignore` excludes everything, then lets in only what the build reads: never `var/lib/timesheet/`, which holds a real database, nor `.git`.
- `.ci/timesheet-entrypoint.sh` adds `--config=/etc/timesheet/timesheet.json` to every command; before `run`, when the database is missing, it runs `init` and `user add <login> --profile=ADMIN --provisional`, with the login of `TIMESHEET_USERNAME` or `admin`, and `TIMESHEET_PASSWORD` or the default password, and stops on the first failure ([Bootstrap](SPECIFICATIONS.md#bootstrap)).
- How to build the image is described in [BUILD.md](BUILD.md#build-the-container-image), how to run it in [OPERATIONS.md](OPERATIONS.md#container).

### Building the package

`.ci/ci-build-deb.sh` builds the binary packages in the working tree, uncommitted changes included:

1. In CI only (variable `CI` set), except on the branch `release` (variable `CI_COMMIT_BRANCH`), it first adds a changelog entry versioned `<version of debian/changelog>+<YYYYMMDDhhmmss>`.
2. It prints the version of the Go toolchain with `go version`, and stops when `go` is missing.
3. It runs `dpkg-buildpackage --build=any,all --no-sign`, which runs `make clean`, `make`, the tests (`make check`, hence gcc), then `make install`, and stops if a test fails.
4. `dpkg-buildpackage` writes the packages next to the repository, in its parent directory.
5. The script creates `_dist/` and moves into it the files `timesheet_*.buildinfo`, `timesheet_*.changes` and `timesheet_*.deb`; it fails when one of them is missing. The package of debug symbols, `timesheet-dbgsym_*.deb`, stays in the parent directory, although the `.changes` file lists it.

`.ci/ci-build-rpm.sh` builds the RPM package in the working tree, uncommitted changes included:

1. It stops when `rpmbuild` or `go` is missing, and prints the version of the Go toolchain with `go version`.
2. It reads the version from the first entry of `debian/changelog`, and stops when it cannot; in CI only (variable `CI` set), except on the branch `release` (variable `CI_COMMIT_BRANCH`), it appends `+<YYYYMMDDhhmmss>`; on `release`, the package takes the version of the changelog as is.
3. It packs the working tree, without `.git/`, `.cache/`, `_dist/`, `_rpm/`, `bin/` and `var/`, into `_rpm/SOURCES/timesheet-<version>.tar.gz`: `var/`, which holds a real database, never leaves the tree.
4. It runs `rpmbuild -bb` with `_topdir` set to `_rpm/` and `ts_version` to the version, which unpacks the archive, runs `make`, the tests (`make check`, hence gcc) and `make install`, and stops if a test fails.
5. It creates `_dist/` and moves into it the packages of `_rpm/RPMS/`; it fails when there is none.

### Scripts

| Script | What it does |
| --- | --- |
| `.ci/ci-setup-debian.sh` | Meant for a disposable CI container, run as root: `apt-get update`, `apt-get dist-upgrade -y`, installs `build-essential`, `git`, `devscripts`, `debhelper`, `fakeroot`, `dh-make`, `dh-exec`, `equivs` and `wget`, then `golang` and its dependencies from `trixie-backports`, with `apt-get install -t trixie-backports`, then runs `apt-get autoremove --purge -y` and `apt-get clean`. Do not run it on a workstation |
| `.ci/ci-build-deb.sh` | Builds the Debian packages (above) |
| `.ci/ci-setup-fedora.sh` | Meant for a disposable CI container, run as root: `dnf upgrade -y`, installs `rpm-build`, `gcc`, `make`, `golang`, `systemd-rpm-macros`, `tzdata`, `tar`, `gzip` and `findutils`, then runs `dnf clean all`. Do not run it on a workstation |
| `.ci/ci-build-rpm.sh` | Builds the RPM package (above) |
| `.ci/ci-cleanup.sh` | Runs `make clean`, `debian/rules clean` when `dh` is installed, and removes `_rpm/` |

### Continuous integration

- `.ci/gitlab-ci.yml` has two jobs in the stage `build`, which run on the branches `master` and `release` and keep the packages of `_dist/` as artifacts.
- `build-debian-amd64` runs `ci-setup-debian.sh`, `ci-build-deb.sh`, then `ci-cleanup.sh`, in the image `debian:trixie-backports`, so that the package is built with the Go of the backports, a supported release of Go, while trixie itself ships Go 1.24.
- `build-fedora-amd64` runs `ci-setup-fedora.sh`, `ci-build-rpm.sh`, then `ci-cleanup.sh`, in the image `fedora:latest`, whose Go is a supported release.
- GitLab reads that file only once the CI configuration path of the project is set to `.ci/gitlab-ci.yml`.

### Release

1. Set `ReleaseVersion` in `src/internal/app/build_info.go`.
2. Add the entry of the same version to `debian/changelog`.
3. `TestVersionIsTheReleaseOfTheChangelog` checks that the first entry of the changelog is that version, or that version followed by `+`.
4. Push to the branch `release`: the CI builds the Debian and RPM packages at that version, without the suffix of the other branches.

## DOCUMENTATION

### Keeping it in sync

- Every change of behaviour, screen, label, route, form field, computation rule, database schema or command updates [SPECIFICATIONS.md](SPECIFICATIONS.md) in the same change.
- Every change of code updates, in the same change, all the documentation it makes wrong: [SPECIFICATIONS.md](SPECIFICATIONS.md) first, then [ARCHITECTURE.md](ARCHITECTURE.md), this file, [SECURITY.md](SECURITY.md), the manuals ([USER-GUIDE.md](USER-GUIDE.md), [BUILD.md](BUILD.md), [OPERATIONS.md](OPERATIONS.md)), [README.md](../README.md) and `CLAUDE.md`; the documentation, the specifications above all, stays in sync with every line of code changed.
- A change that leaves the specifications out of sync with the code is incomplete.
- A request that contradicts the specifications first updates them.
- The [tests of the documentation](#tests-of-the-documentation) check the routes, the notice codes, the server configuration keys and the commands.

### Writing rules

- Documentation is written in English.
- Interface texts are quoted in French between « », exactly as on screen.
- Texts printed by the command line are quoted in English between backticks, exactly as printed.
- No em dash, and no en dash used as a dash, in any document or interface text.
- Never name another project: state every rule as the project's own.
- No personal data: examples are fictitious, and the spreadsheet the summary is modelled on is never copied.
- Level-1 and level-2 headings are written in upper case; the tests of the documentation look for `## ROUTES`, `## COMMAND LINE` and `## SERVER CONFIGURATION`, with its `### Section <name>` headings, in the specifications.
- Code, commands, paths, identifiers and values are written in backticks.
- A placeholder in angle brackets outside backticks is escaped, `\<login>`: otherwise Markdown reads it as an HTML tag, hides it, and a placeholder such as `<code>` breaks the rest of the page.
- Cross-references are relative Markdown links with an anchor, such as `[Limits](SPECIFICATIONS.md#limits)`, never a bare document name or "see above".
- Each fact has one owning document; another document states it in one line at most and links to the owner.
- Every code block names its language (`sh`, `text`, `json`, `go`, `make`, `ini`).

## COMMITS

- A commit message is one line, short and explicit.
- It is written in English, in the imperative mood.
- It has no body.
- It has no `Co-Authored-By` line nor any other attribution.
- Commit, push or reset only on an explicit request in the current message: the approval of a plan or the end of a review is not one. Once the changes are done, propose a commit message and wait.
