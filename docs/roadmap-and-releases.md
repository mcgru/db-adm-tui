# Next steps and release tagging

## Current state

The repository contains:

- v1 read-only workflows, query bar, privilege inspection, history, and release
  hardening;
- the v2 Story 9b access matrix (membership and object privileges);
- a disposable live fixture stack for PostgreSQL 18, MariaDB 11.4, and MongoDB
  8.2 in `live-test/`;
- live-compatibility fixes for hostless MariaDB roles, MongoDB role discovery,
  and PostgreSQL 18 `MAINTAIN`.

Story 9 is not complete. Story 9c users/credential files and Story 9a
row/document editing are still pending.

## Recommended next step

### 1. Live TUI acceptance for 9b

Use the new `live-test/` stack over SSH and verify the actual TUI, not only the
engine probes:

- connect as `db_admin`, `mdb_admin`, and `mongo_admin`;
- open `Ctrl+X Ctrl+R`, switch both panels with `Tab`/`1`/`2`;
- toggle membership and table/schema/database privileges;
- open `e` details and apply a multi-select privilege diff;
- create and drop a role;
- verify refresh of the matrix, Security lists, Privileges pane, and access
  markers;
- verify `r/o` blocks matrix and write queries, and `r/w` allows them;
- verify history entries contain the mode, native statement, status, and
  affected count;
- repeat a failed operation as a non-admin user and check the native error plus
  `Ctrl+D` hint;
- for MongoDB, verify the replica set, `$out`/`$merge` behavior in `r/w`, and
  that built-in roles are read-only.

Fix anything found before declaring Story 9b accepted.

### 2. Story 9c: users and credential files

Implement create/drop native users with a masked password prompt. The password
must never enter a profile, history, log, argv, or error text.

Credential files are the explicit v2 write exception:

- PostgreSQL: `~/.pgpass` or the active `PGPASSFILE`;
- MySQL/MariaDB: `~/.mypass`;
- MongoDB: `~/.mongo.<USER>-in-<DATABASE>` with a full URI;
- format: `HOST:PORT:DBNAME:USER:PASSWORD`, where `DBNAME=*` matches all
  databases;
- comment the matching old line with `#`, append the new line, preserve
  unrelated content, write atomically, file `0600`, directory `0700`;
- after creation, offer save-as-profile without ever storing the password.

### 3. Story 9a: rows and documents

Implement insert/update/delete only after a stable identity rule is defined.
Without a primary/unique key, either block the operation or require an explicit
full-row predicate. Reuse the existing confirmation, native error, history, and
refresh patterns.

### 4. Release documentation

Update README, usage docs, the story status, and release notes with the final
user/role/row scope and any live-verification limitations.

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
| `v1.2.0` | 9c users and credential files | live user creation and credential-file checks |
| `v1.3.0` | 9a rows/documents | live insert/update/delete on all engines |

If 9b is not live-accepted yet, do not tag `v1.1.0`; tag `v1.1.0-rc.1` or wait.

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
- r/o/r/w behavior and native permission errors
- live PostgreSQL 18, MariaDB 11.4, MongoDB 8.2 fixtures

## Engines
- PostgreSQL 18: roles, ACLs, MAINTAIN
- MariaDB 11.4: hostless roles, schema/table grants
- MongoDB 8.2: rs0, users, custom/built-in roles

## Safety
- explicit read-write confirmation
- one logical statement per execution
- history audit with mode/status/affected count
- passwords redacted

## Known limitations
- users/credential files: Story 9c
- row/document editing: Story 9a
- MongoDB query bar stays find/aggregate/count/distinct; writes via $out/$merge
- membership WITH ADMIN OPTION/INHERIT editing is still pending
```
