# db-adm-tui v1.4.0

_Release date: 2026-11-20_

_[English](release-notes-v1.4.0.md) | [Русский](release-notes-v1.4.0.ru.md)_

This release completes the Story 9 write surface for PostgreSQL, MySQL/MariaDB,
and MongoDB: native users and credential files, the access matrix, single-row
insert/delete, and focused-cell update with batch writes.

## Highlights

- **Cell update (9d):** `e` edits the focused Data cell or field. SQL supports
  `val`, `NULL`, and `DEFAULT`; MongoDB applies typed `$set` values, including
  extended JSON. Primary keys and `_id` stay read-only.
- **Batch writes (9d):** `Space` selects rows. `-` deletes the current or all
  selected rows; `e` applies the focused column to the whole selection. SQL uses
  one `OR`-joined key predicate and MongoDB uses `$in` with
  `updateMany`/`deleteMany`.
- **Row insert/delete (9a):** SQL column form with bound values and Mongo
  extended-JSON editor; delete by primary key/`_id` with confirmation.
- **Access matrix (9b):** native role membership and object privileges.
- **Users (9c):** create/drop users, masked passwords, `.pgpass`/`.mypass`/
  `.mongo.*` rotation, and optional save-as-profile.
- **Audit:** successful writes record mode, native statement, status, and
  affected count in history.

## Engines

- PostgreSQL 18: inserts, cell updates, batches, roles/ACLs, users, credentials.
- MariaDB 11.4: inserts, cell updates, batches, hostless roles, users, `.mypass`.
- MongoDB 8.2: extended-JSON insert, `$set` updates, batch delete/update, users
  and roles.

## Safety

- Every write requires `r/w`; `r/o` shows the mode-switch hint and never reaches
  the engine.
- One native statement per batch; identifiers are quoted and values are bound.
- Destructive actions confirm the profile, object, and key(s).
- Rows without a stable primary key/`_id`, or with unsafe binary keys, are
  blocked locally.
- Native engine errors stay visible; passwords are redacted from history and
  errors.

## Known limitations

- The row editor changes one cell/column at a time; use the query bar for
  multi-column edits.
- Multi-row inserts are not available; use the query bar or execute several
  statements.
- MongoDB query bar writes remain limited to `$out`/`$merge`.
- Membership `WITH ADMIN OPTION`/`INHERIT` editing is still pending.

## Verification

Live acceptance on the disposable `live-test/` stack passed for all three
engines. See `live-test/ACCEPTANCE-2026-11-20.md` and
`live-test/TUI-TEST-PLAN.md`.

```text
go test ./...
go test -race ./...
go vet ./...
make verify
```
