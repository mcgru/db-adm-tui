# db-adm-tui

Terminal database administration for PostgreSQL, MongoDB, and MySQL/MariaDB,
built to run on the database host inside a closed environment.

This repository currently contains the v1 planning, UX, architecture, spec, and
the first implementation slice: the Go/Bubble Tea TUI shell. The shell is fake:
there are no database drivers, real profiles, imports, or query execution yet.

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

The shell starts on the fake profile picker.

| Key | Action |
|---|---|
| `Enter` | Open the fake workspace for the selected profile |
| `Up` / `Down` | Move the picker or tree cursor |
| `Tab` / `Shift+Tab` | Cycle focus: tree → main → query bar |
| `1` / `2` / `3`, `[` / `]` | Select the Data, Details, or Privileges tab |
| `Ctrl+D` | Disconnect back to the picker |
| `Ctrl+X` then `Ctrl+X` | Execute stub (chord; `Ctrl+X …` shows for two seconds) |
| `Ctrl+X` then `s` / `Ctrl+E` / `Ctrl+W` / `i` | Chord stubs for count, edit, mode, case |
| `?` / `F1` | Help stub |
| `q` | Quit, but only at the picker |
| `Ctrl+C` | Force quit |

Latin and Cyrillic physical key positions resolve to the same actions (`n`/`т`,
`i`/`ш`, `q`/`й`, `m`/`ь`, and the chord finals).

## Checks

```sh
make test    # go test -race ./...
make lint    # go vet ./...
```

Golden views live in `internal/tui/testdata`. Regenerate them after an
intentional layout change:

```sh
go test ./internal/tui -run TestGolden -update
```

The story spec lives at
`_bmad-output/specs/spec-db-adm-tui/stories/1-foundation-and-tui-shell.md`.
