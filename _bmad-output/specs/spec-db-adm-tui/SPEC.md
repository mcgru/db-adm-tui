---
id: SPEC-db-adm-tui
companions:
  - ../../planning-artifacts/prds/prd-db-adm-tui-2026-11-18/prd.md
  - ../../planning-artifacts/architecture/architecture-db-adm-tui-2026-11-18/ARCHITECTURE-SPINE.md
  - ../../planning-artifacts/ux-designs/ux-db-adm-tui-2026-11-18/DESIGN.md
  - ../../planning-artifacts/ux-designs/ux-db-adm-tui-2026-11-18/EXPERIENCE.md
sources:
  - ../../planning-artifacts/briefs/brief-db-adm-tui-2026-11-18/brief.md
  - ../../planning-artifacts/briefs/brief-db-adm-tui-2026-11-18/addendum.md
---

> **Canonical contract.** This SPEC and the files in `companions:` are the complete, preservation-validated contract for what to build, test, and validate. Source documents in frontmatter are traceability only.

# db-adm-tui — v1

## Why

Engineers who must work inside closed network perimeters cannot use GUI database clients. Tunnels, jump hosts, and proxies are slow, fragile, or forbidden; the reliable path is to SSH onto the database host and use `psql`, `mongosh`, or `mysql`. That path fragments the work: one shell per engine, no shared tree, no privilege view, and every "quick look" becomes a context switch. db-adm-tui is a small, keyboard-first terminal tool that is already where the data is — one static binary on the host, one tree across PostgreSQL, MongoDB, and MySQL/MariaDB, and a read-only default that makes looking around safe.

## Capabilities

- **CAP-1**
  - **intent:** An engineer can connect from a saved profile, an imported credential source, or a new no-password URI, with the password resolved at connect time.
  - **success:** Saved profiles re-read their source on every connect; `~/.pgpass` and a selected `~/.mongo.*` file create usable profiles; a new URI connects or prompts once; no password appears in the profile file, history, argv, or errors.
- **CAP-2**
  - **intent:** The user can navigate one tree of databases, schemas, tables, collections, views, sequences, users, and roles across all three engines.
  - **success:** Databases expand lazily; PostgreSQL schemas, MySQL tables/views, and MongoDB collections/views appear; Security exposes users and roles; `ro`/`rw`/`—` markers match native checks; views/sequences show the v1 placeholder.
- **CAP-3**
  - **intent:** The user can inspect an object's estimated count, size, indexes, and structure, and compute an exact count.
  - **success:** Estimates are labeled and never shown as `0` when unavailable; size and indexes render; MongoDB structure shows sampled field presence; `Ctrl+X s` replaces the estimate with `exact` or cancels without losing it.
- **CAP-4**
  - **intent:** The user can browse paged rows and documents, filter loaded data, run a server-side regexp on a field, and toggle the global search case mode.
  - **success:** The first page is 100; `m`, the end-of-data gesture, and `Ctrl+A` page correctly; `/` filters loaded rows with `n`/`N`; `Ctrl+F` returns a labeled filtered result; `Ctrl+X i` changes case mode everywhere.
- **CAP-5**
  - **intent:** The user can inspect native grants and memberships for the connected user and for selected users or roles.
  - **success:** Direct and via-role entries are distinct; PostgreSQL `INHERIT FALSE` is marked as requiring `SET ROLE`; MySQL granted/active roles are separate; MongoDB inherited roles/privileges render; `not permitted` and `no privilege` differ; no `GRANT`/`REVOKE` is reachable.
- **CAP-6**
  - **intent:** The user can compose and run one SQL or MQL statement with completion, context help, history, and one Results tab for rows and errors.
  - **success:** SQL and the four MQL commands execute; `Ctrl+Space` completes; `Ctrl+F1` explains; `Ctrl+X Ctrl+E` edits; `Up`/`Down` browse history; an error preserves the statement in Results; arbitrary JavaScript and multiple statements are rejected.
- **CAP-7**
  - **intent:** Every connection starts read-only; write statements are blocked until the user explicitly switches to read-write.
  - **success:** The `r/o` badge is visible; PostgreSQL/MySQL set a read-only session; MongoDB relies on the tool guard; a blocked statement never reaches the engine; `Ctrl+X Ctrl+W` confirms and flips the badge; elevated/root warnings appear.
- **CAP-8**
  - **intent:** The tool never leaks secrets and never lets database text control the terminal.
  - **success:** No password appears in argv, logs, history, or errors; history writes `***`, `0600`, and rotates at 10 MB × 5; ANSI/control sequences from data, pasted text, and filenames are stripped; no telemetry or hidden network call exists.
- **CAP-9**
  - **intent:** The user can operate entirely by keyboard at 80×24 with layout-independent shortcuts and offline help.
  - **success:** Every action has a key; letter shortcuts and chord finals work in Latin and Cyrillic; `F1`/`?` opens the keymap; modals trap focus; terminals below 80×24 show the too-small state; help needs no network.
- **CAP-10**
  - **intent:** The tool ships as one static Linux binary that runs in the foreground on the database host.
  - **success:** `CGO_ENABLED=0` builds for Linux `amd64` and `arm64`; no runtime, daemon, listening port, or service manager is required; the only network calls are to the selected database endpoints.

## Constraints

- v1 supports PostgreSQL 14+, MySQL 8.0+, MariaDB 10.4+, and MongoDB 6.0+.
- One live connection per session; no tunnels, proxies, or GUI client.
- The query bar holds one logical statement over at most three wrapped lines; no multi-statement scripts and no arbitrary JavaScript. MQL is limited to `find`, `aggregate`, `count`, and `distinct`.
- Read-only mode is a product safeguard, not a security boundary; writes happen only after an explicit switch and confirmation.
- Engine errors, permission failures, timeouts, and connection loss render inline and never crash; a lost connection returns to the picker.
- Passwords never persist. Profiles are JSON with non-secret fields and an optional source pointer; `~/.pgpass` follows psql semantics (`0600`, escaping, wildcards, first match, `PGPASSFILE`).
- The UI is 80×24 minimum, dark Nightshift, keyboard-first; `DESIGN.md` and `EXPERIENCE.md` are binding companions.
- Performance: cold start under 300 ms, first page under 1 s, data window capped at 10,000 rows / 64 MB.
- History is JSONL + TXT, `0600`, redacted, rotated at 10 MB × 5 per format.
- The binary is static for Linux `amd64`/`arm64`; no daemon, listening ports, telemetry, or update checks.
- `ARCHITECTURE-SPINE.md` AD-1..AD-13 are binding, including hexagonal boundaries, one serialized connection, capability ports, and the statement guard.

## Non-goals

- Not a GUI, DBeaver replacement, IDE, schema designer, or migration tool.
- No interactive data editing, schema changes, or privilege assignment in v1.
- No full multi-line SQL editor, arbitrary JavaScript, or multi-statement scripts.
- No cross-engine query translation; each engine's dialect is used as-is.
- No backup, restore, monitoring, alerting, or fleet management.
- No telemetry, cloud sync, analytics, saved passwords, or credential vault.
- No shared profiles, SSO, or team administration; no plugin system or additional engines in v1.
- No data views for database views/sequences, no sorting, and no export in v1 (sorting/export deferred to v3).
- No Windows target; Linux `amd64`/`arm64` only.

## Success signal

A week of real on-host work passes without falling back to psql/mongosh for browsing; the first external tester completes a find-record task and an access question unaided; a deliberate credential test finds no password in profiles, history, argv, logs, or error output.

## Assumptions

- The module path is `github.com/mcgru/db-adm-tui`, finalized at repo creation.
- Public release is personal-first, public-ready; the license decision waits until v1 proves itself.
- Help corpus content is authored after the v1 scaffolding; `make update-help` is the generation and validation pipeline.
- CI, signing, and release automation are deferred; the Makefile contract is fixed.

## Open Questions

- Public release posture and project license?
- Who owns the help corpus content, and at what cadence?
- Which CI/signing provider for release automation?
- Final module path at repo creation?
