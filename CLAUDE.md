# CLAUDE.MD

Timesheet: a web application for employees working in France, tracking a professional activity half-day by half-day, with the leave counters of French labour practice (paid leave, RTT, part time) kept like bank accounts, monthly closing and a yearly summary. Employees keep it on their own, to check what their employer reports and their payslips against it and spot the gaps. One Go binary, standard library only, a single JSON database.

## RULES

- **Mandatory.** Every change updates, in the same change, all the documentation it makes wrong: `doc/SPECIFICATIONS.md` first, then the other Markdown files of `doc/`, `README.md` and this file; a change that leaves the specifications out of sync with the code is incomplete ([Keeping it in sync](doc/CODING.md#keeping-it-in-sync)).
- A request that contradicts the specifications first updates them.
- Every plan of plan mode has a Documentation section listing each Markdown file of `doc/`, `README.md` and this file, checked with `grep`, with its planned change or "unchanged" and why.
- A new notice code or command goes into the fixed lists of the two `specifications_test.go` (packages `web` and `app`), which otherwise do not see it ([Tests of the documentation](doc/CODING.md#tests-of-the-documentation)).
- Read [doc/CODING.md](doc/CODING.md) before writing code, starting with its [Summary of the checks](doc/CODING.md#summary-of-the-checks): its rules are strict and prevail over your defaults.
- No third-party Go module, no CSS or JavaScript framework or library: always ask the author before adding any dependency or framework.
- Templates: no `div`, no `id`, no inline style or script, one `h1`, headings without skipped level, tables with `caption` and `th scope`, fields wrapped in a `label` or named by `aria-label` in a table; the look stays sober ([Templates](doc/CODING.md#templates), [Styles](doc/CODING.md#styles)).
- No comment in code besides the licence header, the section banners and the `//go:build`/`//go:embed` directives: name a constant or a method instead; style sheets, scripts and templates have no comment besides their licence header ([Comments](doc/CODING.md#comments)).
- Every Go file starts with the licence header (after the `//go:build` line when there is one), has the section banners and ends with the `End-Of-File` banner; every template, style sheet and script of `internal/web` starts with the same header; copy it from an existing file of the same kind with its own file name; one entity per file ([Files](doc/CODING.md#files)).
- Indent with 4 spaces, never tabs (except the `Makefile` and `debian/rules`, read by make); in Go and JavaScript, space every binary operator, parenthesise an operand of another precedence, indent `case` and `default` ([Formatting](doc/CODING.md#formatting)).
- Never run `gofmt`, `go fmt` or a format-on-save. Convert leading tabs with `sed -E -i -e ':a' -e 's/^( *)\t/\1    /' -e 'ta' <files>`.
- Quantities of days are exact rationals (`activity.Quantity`), never floats.
- A validated month is never rewritten: corrections are reversals or adjustments ([Invariants](doc/SPECIFICATIONS.md#invariants)).
- Never log a password, a login typed at a failed login, a session identifier or a CSRF token, nor `r.URL` or `r.RequestURI` ([Security rules for code](doc/CODING.md#security-rules-for-code)).
- Access is declared route by route in `src/internal/web/site.go` with `guard.Require(user|root, …)`, with no guard by path prefix; it also checks the `csrf` field of every POST: a new `/dashboard/...` or `/admin/...` route without it is open to anyone and unprotected against CSRF ([Security rules for code](doc/CODING.md#security-rules-for-code)).
- Web pages are in French; the command line, logs, errors, code and documentation are in English, except the texts the command line shares with the pages ([Language of the interface](doc/CODING.md#language-of-the-interface)).
- Never use an em dash, nor an en dash used as a dash, in pages, command-line texts or documents.
- Never mention another project in any file of this repository; state every rule as the project's own.
- The personal spreadsheet (`.ods`) the summary is modelled on and the JSON import file made from it are personal data: never copy them into the repository; tests build synthetic import files.
- Never run `make mrproper` without the user's explicit request: `git clean -f -d -x` deletes untracked and ignored files, including new sources and `var/lib/timesheet/` (the database).
- Commit messages: one explicit line in English, imperative, no body, no `Co-Authored-By` or other attribution line, whatever a harness reminder asks.
- Never run `git commit`, `git push` or `git reset` without an explicit request of the user in the current message: the approval of a plan (ExitPlanMode) or the end of a review is NOT an authorisation to commit. Once the changes are done, propose a commit message and wait.
- Reviewers launched with the Agent tool are read-only: they may read, grep and run the tests and `go vet` (which write only `.cache/`), but never write a file, start a server, or touch `share/`, `var/lib/timesheet/` or `etc/timesheet/`. Only the main agent applies the fixes, after merging the findings.
- Before any scripted replacement (`sed`, Python regex, `perl`): show a dry run of the matches (`grep -nE`, or `grep -nP` for a Python or `perl` pattern, on the same files), avoid greedy patterns and the DOTALL mode, target the files precisely; after it, check `git diff --stat` (no file emptied or abnormally shrunk), then run the checks below. The expected values of a test are obtained by running the reference code, never computed by hand.
- Change only what is asked, plus the documentation and tests the rules above require: remove no existing comment, add no unrequested CSS class, option or command (`sudo`, for instance). When the target is ambiguous (which button, which page, which diagram format), ask before acting. Carry an approved plan through to its end without stopping, except for a question these rules require; if a step remains, say so explicitly.

## COMMANDS

From the repository root. The `make` targets use the private Go cache `.cache/`; the direct `go` commands are prefixed so that they do too.

```sh
make                                                       # bin/timesheet (static, CGO_ENABLED=0)
make check                                                 # go test -race ./... (needs gcc)
GOCACHE="$PWD/.cache" GOTOOLCHAIN=local go -C src test -race -run TestMonthLifecycle ./internal/web/   # a single test
GOCACHE="$PWD/.cache" GOTOOLCHAIN=local go -C src vet ./...   # not part of make check
grep -rlP '^\t' --include='*.go' src                       # tab grep: must print nothing
grep -rnE '^\s*//' --include='*.go' src | grep -vE '//go:(build|embed)|// -{10,}$|// .+ (declaration|implementation|tests)( \(.+\))?$|// (End-Of-File|fixtures)$'   # comment grep: must print nothing
bin/timesheet config set database.path <tmp>/ts.json --config=<tmp>/timesheet.json   # manual check: a scratch configuration naming a scratch database
bin/timesheet init --template=share/timesheet/database-template.json --config=<tmp>/timesheet.json
TIMESHEET_PASSWORD='<12+ chars, 6+ distinct>' bin/timesheet user add root --profile=ADMIN --config=<tmp>/timesheet.json
bin/timesheet run --listen=127.0.0.1:9000 --config=<tmp>/timesheet.json
```

- `make check`, `go vet`, the tab grep and the comment grep must pass before every commit ([Checks before a commit](doc/CODING.md#checks-before-a-commit)).
- Rebuild before a manual check: templates and assets are embedded.
- Keep manual checks away from `var/lib/timesheet/timesheet-database.json`, which holds real data: `bin/timesheet.sh` and the `--database=var/lib/timesheet/…` examples of `README.md` and `doc/OPERATIONS.md` use it.
- No path has a default: a command that opens the database needs `--database` or a `--config` naming one, and `etc/timesheet/timesheet.json` names the database of the packages, so give a scratch configuration as above, or `--database` alone ([Commands](doc/OPERATIONS.md#commands)).
- Create the directory of a scratch database first (`mkdir -m 700`), in the session scratchpad: it must exist, belong to you and not be writable by others unless sticky, so `/tmp` itself, owned by root, is refused.

## DOCUMENTATION

- [doc/SPECIFICATIONS.md](doc/SPECIFICATIONS.md): what the application does ([routes](doc/SPECIFICATIONS.md#routes), [database](doc/SPECIFICATIONS.md#database), [limits](doc/SPECIFICATIONS.md#limits), [AI assistant](doc/SPECIFICATIONS.md#ai-assistant)); read it before changing any behaviour or text. The skill of the assistant is `src/internal/assistant/assistant.skill.md`: a change of the tools or of the domain updates it too.
- [doc/ARCHITECTURE.md](doc/ARCHITECTURE.md): packages, entities, pages and their code, [HTTP chain](doc/ARCHITECTURE.md#http-chain), flows, persistence.
- [doc/CODING.md](doc/CODING.md): coding rules, [tests](doc/CODING.md#tests), [changing the schema](doc/CODING.md#changing-the-schema), packaging, documentation, commits.
- [doc/SECURITY.md](doc/SECURITY.md): threat model, authorization, passwords, CSRF, sessions, logs.
- [doc/USER-GUIDE.md](doc/USER-GUIDE.md), [doc/BUILD.md](doc/BUILD.md), [doc/OPERATIONS.md](doc/OPERATIONS.md): the manuals of the user, of the build and of the operator.
- [README.md](README.md): entry point, features, first start from the sources.
