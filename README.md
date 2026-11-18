# db-adm-tui

Terminal database administration for PostgreSQL, MongoDB, and MySQL/MariaDB,
built to run on the database host inside a closed environment.

The repository now implements the connection layer: a file-backed profile
store, `~/.pgpass` and `~/.mongo.*` import, a new no-password URI form,
connect-time password resolution, and real open/ping/close against the
selected engine. Catalogs, data views, query execution, and privileges remain
preview content until stories 3–7.

## Requirements

- Go 1.26+ (`go.mod` declares `go 1.26.0`; Bubble Tea v2 requires 1.26+).
- A UTF-8 terminal at least **80×24**. Smaller terminals show only
  `Terminal too small: need 80×24. Current WxH.`
- `cgo` is needed for `go test -race`; the shipped binary is built with
  `CGO_ENABLED=0`.

## Build and run

```sh
make build
./bin/db-adm-tui
```

The tool starts on the profile picker.

| Key | Action |
|---|---|
| `Enter` | Connect to the selected profile, submit a form, or import selected rows |
| `Up` / `Down` | Move the picker, import list, or tree cursor |
| `PgUp` / `PgDn` | Scroll the picker or import list |
| `Home` / `End` | Jump to the first or last import row |
| `n` | New connection (`new connection` modal) |
| `i` | Import from `~/.pgpass` and `~/.mongo.*` |
| `Space` | Select an import row |
| `Tab` / `Shift+Tab` | Move focus: workspace zones; fields in a modal |
| `1` / `2` / `3`, `[` / `]` | Select the preview Data, Details, or Privileges tab |
| `Esc` | Cancel an attempt or close the current modal |
| `Ctrl+D` | Disconnect back to the picker |
| `Ctrl+X` then `Ctrl+X` | Execute stub (chord; `Ctrl+X …` shows for two seconds) |
| `Ctrl+X` then `s` / `Ctrl+E` / `Ctrl+W` / `i` | Chord stubs for count, edit, mode, case |
| `?` / `F1` | Help stub |
| `q` | Quit, but only at the picker |
| `Ctrl+C` / `Ctrl+с` | Force quit, even below the 80×24 floor |

Latin and Cyrillic physical key positions resolve to the same actions
(`n`/`т`, `i`/`ш`, `q`/`й`, `m`/`ь`, and the chord finals).

## Profiles

Profiles live at `$XDG_CONFIG_HOME/db-adm-tui/profiles.json` (default
`~/.config/db-adm-tui/profiles.json`). The directory is `0700` and the file is
`0600`. Schema v1 stores only `label`, `engine`, `host`, `port`, `database`,
`user`, an optional source pointer, and `created_at` / `updated_at` /
`last_used`. **No password is ever written to a profile.**

If `profiles.json` is corrupt or unreadable, the picker shows
`Profile storage is unreadable. Showing no profiles.` and stays usable; `n` and
`i` still work. Import and save refuse to overwrite an unreadable store; fix or
remove the file by hand before saving again. The picker source column warns
`ignored`, `missing`, `stale`, or `error` when a source is unusable. One profile
is kept per selected `~/.mongo.*` source file, so saving a new URI that matches
that source refreshes that profile (including its stored non-secret Mongo host
list and options); a changed `~/.pgpass` selector leaves the old profile visible
as `stale` and creates a new one on re-import.

## Import

### PostgreSQL (`~/.pgpass`)

The parser follows psql semantics:

- Each line is `host:port:database:user:password`.
- `PGPASSFILE` is honored; otherwise `~/.pgpass` is used.
- A file with group/other permissions (anything other than owner-only) is
  ignored, exactly as psql does.
- Lines beginning with `#` are comments.
- `\` escapes the next character, including `:`.
- `*` matches any host, port, database, or user; the first matching line wins.
- Each valid line creates or refreshes a profile; the password stays only in
  memory at connect time.
- Malformed lines are skipped with a `warning` tag and never block the valid
  lines.
- A missing file is an empty state, not an error.

### MongoDB (`~/.mongo.*`)

Every `~/.mongo.*` file plus `~/.mongo` is listed, one row per file. Select the
files to import with `Space`, then press `Enter`. Within a file, `#` comment
lines are skipped and the last valid `mongodb://` URI wins. `mongodb+srv://` is
not supported in v1. A malformed line makes that file show an `error` tag naming
the file and it does not import, while other files remain selectable. The source
selector preserves non-secret options such as `authSource`, `replicaSet`, and
`tls`; password keys are stripped before the profile is written.

MySQL/MariaDB has no import source in v1; create those profiles from a URI or a
prompted password.

## New connection

`n` opens a centered form with a no-password URI and an optional label:

```text
postgres://user@host[:port]/db
mysql://user@host[:port]/db
mongodb://user@host[:port]/db
```

The database path is required, the engine's default port is used when omitted,
and an empty label defaults to `database@host`. A URI containing
`user:pass@` or `password=`/`pwd=` is rejected inline with
`Could not connect: the URI must not contain a password.` Query options are not
accepted by the new-connection form in this story; put TLS/auth options in the
selected `~/.mongo.*` source file instead.

On submit, the tool looks for a matching `~/.pgpass` line or `~/.mongo.*` URI.
If one exists, its password is used; otherwise the form reveals a masked
one-time password field. If a matched source password is rejected, the form
offers a one-time password instead and the saved profile becomes prompt-based.
On success the picker offers
`Save <label> as a profile? Enter save · Esc skip`. Saving stores the
non-secret fields and any matched source pointer, never the password.

## Password behavior

- Passwords are resolved from the saved source **on every connect**; changes to
  `~/.pgpass` or the selected `~/.mongo.*` file are respected.
- A saved profile with no source opens a centered masked
  `Password for <profile>` modal. `Enter` connects and `Esc` returns to the
  picker. The value is never stored.
- The source files are only read; the tool never writes them.
- Errors name the profile and source file but never the secret.
- Source-derived text, labels, filenames, and errors are sanitized of ANSI and
  control sequences before rendering.

## Checks

```sh
make test    # go test -race ./...
make lint    # go vet ./...
make build   # CGO_ENABLED=0 static binary in bin/
```

Golden views live in `internal/tui/testdata`. Regenerate them after an
intentional layout change:

```sh
go test ./internal/tui -run TestGolden -update
```

The story spec lives at
`_bmad-output/specs/spec-db-adm-tui/stories/2-connection-setup-profiles-import-new-connection.md`.
