# Story 9 (v2) — Structured write operations

**Status:** 9b access matrix implemented; 9c users and 9a rows pending
**Scope:** users, access roles/grants, and rows in the current table/collection
**Depends on:** Story 6 read-write mode, Story 7 privilege inspection, Story 8 security/release

## Why this is a v2 story

The v1 PRD explicitly excludes interactive data editing, schema changes, and
privilege assignment; FR-40 also says no `GRANT`, `REVOKE`, membership change,
or user creation is reachable from the Privileges view. This story is therefore
a deliberate v2 expansion, not a v1 continuation. The v1 read-write switch is
the mechanism it builds on; the spec and README must be amended before code.

## Product intent

Make common small writes possible without leaving the TUI:

- add/remove native users;
- add/remove roles and role access to databases, schemas, tables, or
  collections;
- add/remove rows or documents in the selected object.

`+` and `-` are contextual. They never do anything in read-only mode except
name the reason and point to `Ctrl+X Ctrl+W`. Every mutation requires an
explicit confirmation that names the target and the native effect. There is no
silent write.

## Decisions from the product review

- **Order changed:** implement **9b roles/grants first**, then **9c users**, and
  keep **9a rows deferred**. Rows are not part of the first implementation.
- **Read-write gate:** Story 6's switch is the gate. In `r/o`, `+`/`-`/`e`
  mutations report that read-write mode is required and point to
  `Ctrl+X Ctrl+W`; the tool never switches automatically.
- **Engine coverage:** PostgreSQL, MySQL/MariaDB, and MongoDB together.
- **Role actions:** grant/revoke native privileges to a role, create/drop
  roles, and assign/remove roles for users.
- **Editing model:** a selected list value can be changed by cycling values
  with `Space`, or `e` opens a modal editor with the same list; use
  multi-select where the native operation supports several values. The exact
  list per context must stay native.
- **User creation password:** masked prompt, never stored in a profile. The
  password is appended to the matching credential file (never to a directory):
  - PostgreSQL: `~/.pgpass` (or the active `PGPASSFILE`), one line per
    credential in the exact form `HOST:PORT:DBNAME:USER:PASSWORD`, with psql
    backslash escaping and `#` comments.
  - MySQL/MariaDB: `~/.mypass`, using the same
    `HOST:PORT:DBNAME:USER:PASSWORD` format and escaping rules.
  - MongoDB: a separate file per user/database named
    `~/.mongo.<USER>-in-<DATABASE>`. Its content is the full non-shell URI:
    `mongodb://user:password@host1:port1,host2:port2,.../dbname?opt1=val1&opt2=val2`.
  - Credential files are `0600`; parent directories are `0700`. Passwords
    never appear in profiles, history, logs, argv, or errors.
- **Existing credential entry:** comment out the matching old line/file entry
  with `#` and append the new one below it, preserving unrelated entries.
- **v2 security amendment:** v1 said the tool only reads source files. Story 9
  deliberately changes that for credential sources created through the user
  flow; the PRD/README/spec must record the exception and the secure write
  rules before implementation.

## Proposed split

### 9b — Access roles and grants (first)

- Create/drop native roles.
- Grant/revoke privileges to an existing role on a database/schema/table/
  collection (engine-native privilege names only).
- Add/remove role membership for a user/role where the engine supports it.
- Entry points:
  - Security → role: manage membership/role attributes.
  - Database/schema/table context → `+` grant access; `-` revoke a selected
    native grant from the Privileges view.
- Editing:
  - `Space` on a selected editable list value cycles through the native
    choices;
  - `e` opens a modal editor with a multi-select list where the engine allows
    more than one value.
- After success, refresh the Privileges tab and the affected tree node.
- Dropping a role with dependent grants/ownership surfaces the native error;
  the tool does not invent cascade semantics.

### 9c — Users (second)

- Create/drop native users.
- Password on creation is prompted masked and written only to the source file
  rules above; a generated-password option is a possible follow-up.
- "User in database" is engine-specific:
  - PostgreSQL: role is cluster-scoped; database access is represented by
    `CONNECT`/schema/table grants, not a per-database user object.
  - MySQL/MariaDB: user is server-scoped (`user@host`); privileges can be
    scoped to a database/table.
  - MongoDB: user is database-scoped; roles are assigned per database.
- Dropping the currently connected user or a role that owns dependencies is
  blocked by the engine or by an explicit pre-check; the tool never silently
  drops the live session.
- After success, refresh Security and the relevant Privileges view.

### 9a — Rows (deferred)

Keep the previously prepared row/document plan: single-row insert/delete,
SQL column form plus Mongo JSON editor, primary key/`_id` identity, refresh
Data and history. Do not implement it in the first Story 9 slice.


## UI rules

- `+`/`-` only act when focus is on a context that supports the action:
  - tree: principal / role / grant context;
  - main Data/Results grid: current row/document;
  - query bar: literal characters, never actions.
- `+` opens a form or a short action picker; `-` opens a confirmation.
- Confirmation copy names the profile, object, and native action; destructive
  removal uses an explicit `Enter remove · Esc cancel`.
- Read-only mode shows `Blocked: mutation requires read-write mode.` and the
  mode-switch hint. It does not auto-switch.
- Errors are inline, preserve the current view, and name the native operation.
- No nested modals; one mutation at a time; the session lease blocks other
  engine-backed actions while a write is in flight.

## Resolved decisions

- **MySQL/MariaDB credentials:** use `~/.mypass` with the same
  `HOST:PORT:DBNAME:USER:PASSWORD` line format and psql-style escaping.
  Extend the resolver/import path so a created account can be reused.
  A `*` database field matches every database, following the existing
  `~/.pgpass` wildcard semantics.
- **Existing credential entry:** comment out the matching old line with `#`
  and append the new entry immediately below it. Preserve unrelated lines and
  rewrite the file atomically with `0600`. For MongoDB, the per-user/database
  file is created or updated in the same way.
- **Profile after user creation:** offer the existing save-as-profile prompt;
  the source file is written first, the profile remains optional.
- **Audit:** use the existing history JSONL/TXT files, adding the native
  operation, mode, status, and affected count. Passwords and password literals
  remain redacted.
- **Grant flow:** select a role first, then choose the database/schema/table
  target and native privilege.
- **Live databases:** not available; use sqlmock and pure Mongo helpers/seams,
  and record live verification as deferred integration work.

## Matrix UI decision (replaces the modal role editor)

The role UI is two normalized matrices, opened with the reserved
`Ctrl+X Ctrl+R` chord in `r/w` mode:

1. **Principal × Role** — membership. Rows are users and roles; columns are
   roles; toggling a cell grants/revokes membership. `e` opens details such as
   `WITH ADMIN OPTION` / `INHERIT` where the engine exposes them.
2. **Principal/role × Object** — privileges. Rows are roles, plus users for
   PostgreSQL/MySQL so direct grants stay visible; MongoDB has role rows only.
   Columns are database/schema/table/collection scopes from the current
   database/schema, with `/` filtering and horizontal scrolling. A cell shows
   the native privilege summary. `Space` toggles the default privilege for that
   target (`CONNECT`/`USAGE`/`SELECT` for SQL; `find` for MongoDB), while `e`
   opens the full multi-select native privilege list.

`Ctrl+X Ctrl+R` is blocked in `r/o` with the same mode-switch hint as other
mutations. The current account must have native admin rights; the target user's
login/password is irrelevant. A `not permitted` result shows a reconnect hint
(`Ctrl+D`), not an automatic account switch. A read-only “who can administer
roles” helper is a later enhancement.

**Switching matrices:** `Tab` / `Shift+Tab` move between `Principal × Role` and
`Principal/Role × Object`, and `1` / `2` select either matrix directly. The
matrix header shows both names with the active one highlighted; the footer
shows `Tab switch · 1 membership · 2 privileges · Space toggle · e details · / filter · Esc close`.
Each matrix keeps its own cursor, scroll position, and filter while the other is
active. Arrows move inside a matrix; `Tab` is reserved for switching panels
inside this surface, not for moving between cells or workspace zones.

The existing create/drop role dialogs and `+`/`-` actions remain; the modal
grant/assign/revoke forms are replaced by the matrices. The engine/app
`RoleAdmin` API is retained.

## Implementation status

- **9b matrix slice implemented:** Native `RoleAdmin` and `RoleMatrixReader`
  ports; PostgreSQL, MySQL/MariaDB, and MongoDB adapters; app read-write gate
  and history audit; TUI `Ctrl+X Ctrl+R` full-pane matrix with membership and
  privilege panels, per-panel cursor/filter, `Space` toggle, `e` multi-select
  details, `/` filter, and `Esc` close. Role create/drop remain in the tree.
  Tests use sqlmock, pure Mongo helpers, and TUI fakes.
- **Matrix data is structured:** membership cells and privilege cells come from
  native catalogs; rendered Privileges-tab text is not parsed for edits.
  PostgreSQL reads role/ACL catalogs, MySQL reads role edges and the
  information-schema privilege tables, and MongoDB reads `usersInfo` and
  `rolesInfo`. Built-in MongoDB roles appear as membership columns but are not
  editable privilege rows.
- **Known 9b limitations:** PostgreSQL `PUBLIC` grants are read but not shown as
  a matrix row, and unsupported native privileges (for example PostgreSQL
  `MAINTAIN`) remain visible but are not toggleable. Membership details show
  `WITH ADMIN OPTION`/`INHERIT` as a later enhancement rather than editing them.
  MongoDB role creation is scoped to the current database. Live databases are
  not available, so native command acceptance is deferred.
- **Next:** 9c users, then the deferred 9a rows.

## Risks and constraints

- PostgreSQL roles are cluster-wide; MySQL users are server-wide; MongoDB users
  are database-scoped. A single cross-engine "user in database" abstraction is
  false and must not be invented.
- Native role/privilege syntax differs substantially; the UI must keep native
  terms and cannot offer a uniform privilege matrix yet.
- Row deletes need a stable identity; tables without a primary/unique key need
  a conservative block or an explicit full-row predicate.
- SQL forms must handle defaults, nullability, generated/identity columns,
  JSON/binary values, and quoting; Mongo forms must preserve document shape.
- Destructive user/role drops can fail because of ownership or dependent
  grants; there is no cross-engine cascade policy.
- Every mutation must stay behind the read-write switch and confirmation; the
  guard is a product safeguard, not a security boundary.
- No live database is available in the current environment; the first
  implementation needs an integration plan or must explicitly defer live
  verification.

## Acceptance sketch

- In r/o, `+`/`-` never reach the engine and show the mode-switch hint.
- In r/w, the user can insert and delete one row/document with confirmation,
  and the Data view refreshes with the affected count.
- In r/w, the user can create/drop a role and grant/revoke one native
  privilege, with refresh and inline native errors.
- In r/w, the user can create/drop a user with a masked password that never
  appears in history, profiles, logs, or errors.
- Every mutation is recorded with mode, profile, native operation, status, and
  affected count; passwords and password literals are redacted.
- The tree, Privileges tab, and Data tab never show stale data after a
  successful mutation.
