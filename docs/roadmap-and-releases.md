# Next steps and release tagging

_[English](roadmap-and-releases.md) | [Русский](roadmap-and-releases.ru.md)_

## Current state

The repository contains:

- v1 read-only workflows, query bar, privilege inspection, history, and release
  hardening;
- the v2 Story 9b access matrix (membership and object privileges);
- a disposable live fixture stack for PostgreSQL 18, MariaDB 11.4, and MongoDB
  8.2 in `live-test/`;
- live-compatibility fixes for hostless MariaDB roles, MongoDB role discovery,
  and PostgreSQL 18 `MAINTAIN`;
- `.mypass` import and connection-source support for MySQL/MariaDB;
- Story 9c native user creation/drop, masked passwords, credential-file writes
  with rotation, and save-as-profile;
- Story 9a single-row/document insert and delete from the Data tab, with
  primary key/`_id` identity, `r/w` gating, Data refresh, and history audit;
- Story 9d focused-cell update, `Space` row selection, and batch update/delete
  on SQL and MongoDB, with affected counts in history.

Inline multi-column edits and multi-row inserts remain query-bar workflows.

## Acceptance status

All Story 9 slices are implemented and live-verified on the `live-test/` stack:

- 9a insert/delete, 9b access matrix, 9c users, and 9d cell update/batch writes;
- PostgreSQL 18, MariaDB 11.4, and MongoDB 8.2;
- `live-test/ACCEPTANCE-2026-11-20.md` records the checks, commits, and cleanup.

## Recommended next step: release v1.4.0

1. Run the full check suite:

   ```sh
   go test ./...
   go test -race ./...
   go vet ./...
   make verify
   ```

2. Build the release archives:

   ```sh
   make release VERSION=v1.4.0
   ./dist/db-adm-tui_v1.4.0_linux_amd64/db-adm-tui --version
   ./dist/db-adm-tui_v1.4.0_linux_arm64/db-adm-tui --version
   ```

3. Tag and publish with release notes covering 9a–9d, the live acceptance
   result, and the known limitations below.

4. Update the binary at `jojo@uuu:~/db-adm-tui` and keep `~/live-test` in sync.

Do not tag before the acceptance record and README/usage docs match the released
binary.

## Release tagging policy

Use Semantic Versioning with a `v` prefix:

```text
vMAJOR.MINOR.PATCH
v1.1.0
v1.1.0-rc.1
```

- **MAJOR** — incompatible profile/state schema, CLI, or behavior changes.
- **MINOR** — user-visible features such as the access matrix or user editing.
- **PATCH** — backward-compatible fixes with no new feature scope.
- **Pre-release** — `-rc.N` while a feature is code-complete but live
  acceptance or compatibility work is still open.

### Suggested milestones

| Tag | Content | Gate |
| --- | --- | --- |
| `v1.1.0-rc.1` | 9b access matrix, live fixtures, live compatibility fixes | engine probes and local stack pass; TUI live smoke still open |
| `v1.1.0` | accepted 9b | TUI live smoke on all three engines, docs/release notes updated |
| `v1.2.0` | 9c users and credential files | live user creation, rotation, drop, and save-as-profile checks |
| `v1.3.0` | 9a rows/documents | live single-row insert/delete on all engines |
| `v1.4.0` | 9d cell update/batch writes | live update, batch update, and batch delete on all engines |

The current release can ship as one cumulative `v1.4.0`; if a per-feature tag
history is required, tag `v1.3.0` at the 9a commit before tagging `v1.4.0`.

The existing tags `1.0.1` and `v.1.0.0` both point at the same old commit and
use inconsistent naming. Do not rewrite tags that were already published. Use
`v1.1.0` going forward. If those tags were never published, they can be cleaned
up separately to `v1.0.0`/`v1.0.1`.

### When to create a tag

Tag only when all of the following are true:

- the working tree is clean and `main` is up to date;
- `go test ./...`, `go test -race ./...`, `go vet ./...`, and
  `make verify` pass;
- `make release VERSION=vX.Y.Z` produces both Linux archives;
- the binary reports the intended version, commit, and date;
- the README/usage docs and release notes describe the current scope and known
  limitations;
- live smoke checks pass for the engines affected by the release;
- no known blocker or silently deferred acceptance item remains.

Do not tag every commit or a half-finished story. Use commit SHAs for
work-in-progress checkpoints.

### Release checklist

```sh
git checkout main
git pull --ff-only

go test ./...
go test -race ./...
go vet ./...
make verify

make release VERSION=v1.1.0
./dist/db-adm-tui_v1.1.0_linux_amd64/db-adm-tui --version
./dist/db-adm-tui_v1.1.0_linux_arm64/db-adm-tui --version

git tag -a v1.1.0 -m "db-adm-tui v1.1.0"
git push origin main
git push origin v1.1.0
```

Before tagging, run the live acceptance checks in `live-test/`. After tagging,
publish the two archives from `dist/` and include the release notes.

## Release notes outline

```text
## Highlights
- access matrix for role membership and native object privileges
- native user create/drop with credential-file rotation
- row/document insert, focused-cell update, and batch delete/update
- r/o/r/w behavior and native permission errors
- live PostgreSQL 18, MariaDB 11.4, MongoDB 8.2 fixtures

## Engines
- PostgreSQL 18: roles, ACLs, MAINTAIN, bound row writes
- MariaDB 11.4: hostless roles, schema/table grants, batch writes
- MongoDB 8.2: rs0, users, custom/built-in roles, typed $set/batch writes

## Safety
- explicit read-write confirmation
- one native statement per batch
- history audit with mode/status/affected count
- passwords redacted; primary keys and _id protected

## Known limitations
- one cell/column at a time in the row editor; multi-column edits use the query bar
- multi-row inserts are not available; use the query bar or paste multiple statements
- MongoDB query bar stays find/aggregate/count/distinct; writes via $out/$merge
- membership WITH ADMIN OPTION/INHERIT editing is still pending
```
