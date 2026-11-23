# BUILD

This document describes how to build Timesheet from its sources: the program, its tests, the installation of its files, the Debian and RPM packages and the container image.

It is written for whoever builds the software. How the build scripts are written is in [CODING.md](CODING.md#makefile); how to run the result is in [OPERATIONS.md](OPERATIONS.md).

## CONTENTS

- [REQUIREMENTS](#requirements)
- [BUILD THE PROGRAM](#build-the-program)
- [RUN THE TESTS](#run-the-tests)
- [INSTALL THE FILES](#install-the-files)
- [CLEAN](#clean)
- [BUILD THE DEBIAN PACKAGE](#build-the-debian-package)
- [BUILD THE RPM PACKAGE](#build-the-rpm-package)
- [BUILD THE CONTAINER IMAGE](#build-the-container-image)
- [VERSION](#version)

## REQUIREMENTS

| To | You need |
| --- | --- |
| Build the program | Go 1.24 or later, and `make`. |
| Run the tests | `gcc` as well: the race detector needs cgo. |
| Build the Debian package | A Debian system with `build-essential`, `debhelper`, `devscripts`, `golang` and `git`. The CI takes `golang` from `trixie-backports`; trixie alone builds with Go 1.24 ([CODING.md, Continuous integration](CODING.md#continuous-integration)). |
| Build the RPM package | A Fedora system with `rpm-build`, `gcc`, `make`, `golang` (Go 1.24 or later), `systemd-rpm-macros` and `tzdata`, the packages that `.ci/ci-setup-fedora.sh` installs. The spec also has the variants of openSUSE (`go` and `timezone`), which no build checks. |
| Build the container image | Docker, or another builder of `Dockerfile` images, and access to the registry of the images `golang:1.27-alpine3.23` and `alpine:3.22`. |

The program has no dependency besides the Go standard library: nothing is downloaded during the build, except the base images of the container image.

Run every command of this document from the root of the repository.

## BUILD THE PROGRAM

```sh
make
```

- It produces `bin/timesheet`, a static binary that runs without any other file: the pages, the styles and the scripts are embedded in it.
- It ends with `=== build ok ===`.
- The build cache is kept in `.cache/`, at the root of the repository. The first build, and the first build after `make clean`, take a few seconds more.
- Rebuild after every change of the sources, templates and styles included.
- The binary has no default path: `--config` gives the server configuration, `etc/timesheet/timesheet.json` from the sources; without it, every setting takes its default and `--database` gives the database, which has no default path. `init` takes the configuration template with `--template`.

## RUN THE TESTS

```sh
make check
```

- It runs every test with the race detector, and ends with `=== check ok ===`.
- It needs `gcc`.

## INSTALL THE FILES

`make install` copies the built files into a directory tree. It does not build: run `make` first.

```sh
make
make install DESTDIR=/tmp/stage
```

| File | Installed as | Mode |
| --- | --- | --- |
| `bin/timesheet` | `$(DESTDIR)/usr/bin/timesheet` | 755 |
| `share/timesheet/database-template.json` (configuration template) | `$(DESTDIR)/usr/share/timesheet/` | 644 |
| `share/timesheet/database-example.json` (example database) | `$(DESTDIR)/usr/share/timesheet/` | 644 |
| `share/timesheet/import-template.json` (import template) | `$(DESTDIR)/usr/share/timesheet/` | 644 |
| `etc/timesheet/timesheet.json` (server configuration, as is: the database at `/var/lib/timesheet/timesheet-database.json`) | `$(DESTDIR)/etc/timesheet/` | 644 |

- Without `DESTDIR`, the files go under `/usr` and `/etc`: run `sudo make install`.
- It fails when `bin/timesheet` is missing.
- It replaces an `/etc/timesheet/timesheet.json` already there: save your own first.
- It creates neither the service account, nor the database directory, nor the service. The [Debian package](#build-the-debian-package) and the [RPM package](#build-the-rpm-package) do.
- The server configuration names the database `/var/lib/timesheet/timesheet-database.json` (`database.path`): the installed binary given `--config` finds it through the configuration, without `--database`.
- Give the installed binary `--config=/etc/timesheet/timesheet.json` and, for `init`, `--template=/usr/share/timesheet/database-template.json`.

## CLEAN

| Command | Effect |
| --- | --- |
| `make clean` | Removes `bin/timesheet` and empties the build cache `.cache/`. The database is kept. |
| `make mrproper` | Runs `git clean -f -d -x`: deletes every file that git does not track, the database under `var/lib/timesheet/` and any new source file included. |

Never run `make mrproper` without a backup of the database and a commit of your new files.

## BUILD THE DEBIAN PACKAGE

1. On a Debian system with the [requirements](#requirements), from the root of the repository:

   ```sh
   .ci/ci-build-deb.sh
   ```

2. The build runs `make check` and stops if a test fails.
3. The package is written into `_dist/`, at the root of the repository: `timesheet_<version>_amd64.deb` (the architecture is the one of the build machine), with its `.buildinfo` and `.changes` files. Its debug symbols, `timesheet-dbgsym_<version>_amd64.deb`, stay in the parent directory of the repository: delete them by hand when you do not need them.
4. To remove the build leftovers afterwards:

   ```sh
   .ci/ci-cleanup.sh
   ```

- The package is built from the working tree, uncommitted changes included.
- The package installs the server configuration of the repository as is: its `database.path` is the database of the package.
- `<version>` is the first entry of `debian/changelog`; in the CI, except on the branch `release`, it is followed by `+` and the date and time of the build.
- Installing and running the package is described in [OPERATIONS.md](OPERATIONS.md#debian-package).

## BUILD THE RPM PACKAGE

1. On a Fedora system with the [requirements](#requirements), from the root of the repository:

   ```sh
   .ci/ci-build-rpm.sh
   ```

2. The build runs `make check` and stops if a test fails: the tests need an architecture where Go has the race detector, such as `x86_64` or `aarch64`.
3. The package is written into `_dist/`, at the root of the repository: `timesheet-<version>-1.<distribution>.<arch>.rpm`, with the distribution tag and the architecture of the build machine. No package of debug symbols is built.
4. To remove the build leftovers afterwards, the directory `_rpm/` among them:

   ```sh
   .ci/ci-cleanup.sh
   ```

- The package is built from the working tree, uncommitted changes included, without `.git/`, `.cache/`, `_dist/`, `_rpm/`, `bin/` and `var/`: the database of the repository never enters it.
- `<version>` is the first entry of `debian/changelog`; in the CI, except on the branch `release`, it is followed by `+` and the date and time of the build.
- Installing and running the package is described in [OPERATIONS.md](OPERATIONS.md#rpm-package).

## BUILD THE CONTAINER IMAGE

1. From the root of the repository:

   ```sh
   make docker
   ```

   It runs `docker build -t timesheet .` and ends with `=== docker ok ===`.

2. The build runs in two stages:
   - in `golang:1.27-alpine3.23`, which receives the time zones (`tzdata`) that `config set` needs to validate `server.timezone`, `.ci/make-build.sh` and `.ci/make-install.sh` build and install the files into a staging tree, as `make` and `make install` do; `.ci/timesheet-entrypoint.sh` is added as `/usr/bin/timesheet-entrypoint.sh`, and `config set` makes the server configuration listen on `0.0.0.0:8080` and trust no proxy;
   - `alpine:3.22` receives these files, the time zones (`tzdata`) and the system account `timesheet` (uid and gid 10001).
3. The image does not run the tests: run `make check` before.

- The image is built from the working tree, uncommitted changes included. `.dockerignore` sends only the sources, the build and installation scripts, the entry point, the server configuration and `share/timesheet/`: never the database under `var/lib/timesheet/`, nor `.git`, so `version` prints only the release number.
- `docker compose build`, or `docker compose up --build`, builds the same image `timesheet` from `docker-compose.yml`.
- Running the image is described in [OPERATIONS.md](OPERATIONS.md#container), with Docker Compose in [OPERATIONS.md](OPERATIONS.md#with-docker-compose).

## VERSION

```sh
bin/timesheet version
```

It prints the release number, the commit the binary was built from and its date, and `modified` when the tree had uncommitted changes; a binary built from an archive, outside a git checkout, prints only the release number. Releasing a new version is described in [CODING.md](CODING.md#release).
