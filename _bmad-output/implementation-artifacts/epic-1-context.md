# Epic 1 Context: db-adm-tui v1

<!-- Compiled from planning artifacts. Edit freely. Regenerate with compile-epic-context if planning docs change. -->

## Goal

Ship a single static Linux binary that lets an engineer working on a database host inspect PostgreSQL, MongoDB, and MySQL/MariaDB through one keyboard-first terminal UI. The product must be safe by default: read-only connections, no persisted passwords, no telemetry, no listening ports, and no ANSI/control injection from database text. Success is a week of real on-host work without falling back to `psql`, `mongosh`, or `mysql` for browsing.

## Stories

- Story 1: Foundation and TUI shell
- Story 2: Connection setup: profiles, import, new connection
- Story 3: Common tree, Security, and object details
- Story 4: Data browsing and search
- Story 5: Query bar, Results, completion, and history
- Story 6: Read-only guard and mode switch
- Story 7: Privilege inspection
- Story 8: Security hardening and release

## Requirements & Constraints

- Connect from a saved profile, an imported `~/.pgpass` or selected `~/.mongo.*` file, or a new no-password URI; resolve the password only at connect time. Saved profiles re-read their source on every connect.
- No password may appear in the profile file, history, argv, error output, or logs. Profiles store non-secret fields plus an optional source pointer. A saved profile with no source prompts once per connect.
- Profiles live in JSON schema v1 under `$XDG_CONFIG_HOME/db-adm-tui/profiles.json`; file `0600`, directory `0700`. `~/.pgpass` follows psql semantics: `0600`, backslash escaping, `*` wildcards, first match, `PGPASSFILE`.
- One live connection per session; disconnect closes it and returns to the picker. Connection loss does the same with an inline row error.
- Engine errors are `domain.Error` values with kinds (`auth`, `permission`, `not_found`, `timeout`, `cancelled`, `guard`, `engine`, `config`); raw driver text and DSNs are redacted at the adapter boundary.
- The UI is 80x24 minimum, Nightshift dark, keyboard-first. Letter shortcuts and chord finals work in both Latin and Cyrillic physical positions.
- Read-only is the default. The guard is a product safeguard, not a security boundary. MQL is limited to `find`, `aggregate`, `count`, and `distinct`.
- History is redacted JSONL + TXT, `0600`, rotated at 10 MB x 5; history failures warn but never block a query.
- `CGO_ENABLED=0`; pure-Go drivers; Linux `amd64`/`arm64`; no daemon, listening port, telemetry, update check, or hidden network call.

## Technical Decisions

- Hexagonal core with a Bubble Tea MVU edge: `domain` is standard-library only; `app` owns session, mode, serialization, and generations; ports are small capability interfaces; only `cmd/db-adm-tui` wires concrete adapters.
- One serialized connection per session. SQL adapters use one `*sql.Conn` from a `sql.DB` with `MaxOpenConns=1`; MongoDB uses one client. Engine calls take a context and a capacity-1 lease.
- Capability bundles return non-nil ports only when supported; `Capabilities` derives from the bundle, never supplied independently. No `Engine` god interface and no `ErrNotImplemented` branching in the TUI.
- External text is sanitized of ANSI/control sequences before rendering. The TUI maps error kinds to copy and never branches on error strings.
- Profiles are file-backed; the adapter owns pgpass/mongo parsing. `ProfileStore` is a port. Credential resolution happens in the profile/app layer, not the TUI.
- Request generations make stale async results droppable; the TUI owns only view state. No mutable globals.
- Stack: Go 1.26+, Bubble Tea v2.1.2, Lip Gloss v2.1.1, Bubbles v2.2.1, pgx stdlib v5.11.0, go-sql-driver/mysql v1.10.2, mongo-driver v2.10.0.

## UX & Interaction Patterns

- Start at the full-screen profile picker: title, subtitle, profile rows, footer keys, count line, viewport scrolling, inline row errors, selected-row fill.
- Picker keys: `n` new, `i` import, `q` quit, `?` help, `Enter` connect. `Ctrl+D` disconnects from the workspace and selects the profile for reconnect.
- New connection is a centered modal with a no-password URI and optional label; reject `user:pass@`; the password comes from a source file or one-time prompt. Import shows discovered entries with checkboxes and source/user/engine.
- Transient status messages clear on the next key or after five seconds; errors persist until the next action. Modal focus traps `Tab`.
- No color-only information: words (`r/o`, `r/w`, `ro`, `rw`, error) carry meaning.

## Cross-Story Dependencies

- Story 1 supplies the Go module, theme, layout, keymap, and fake picker/workspace that story 2 replaces with real profile state.
- Story 2's `ProfileStore`, credential resolution, and connection factory are prerequisites for every engine-backed story (3-8).
- Stories 3-7 depend on a connected session and capability bundle; story 8 hardens and releases the complete product.
- All stories share AD-1..AD-13, `SPEC.md` CAP-1..CAP-10, `DESIGN.md`, and `EXPERIENCE.md`.
