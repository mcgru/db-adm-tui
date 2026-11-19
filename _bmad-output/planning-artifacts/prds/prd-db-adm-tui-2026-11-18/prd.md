---
title: "PRD: db-adm-tui"
status: final
created: 2026-11-18
updated: 2026-11-18
---

# PRD: db-adm-tui

## 0. Document Purpose

This PRD defines v1 of db-adm-tui for the builder, the first external tester, and the downstream UX, architecture, and story-writing workflows. It builds on the product brief and its addendum at `_bmad-output/planning-artifacts/briefs/brief-db-adm-tui-2026-11-18/` and does not restate them. Vocabulary is fixed in the Glossary, features are grouped with globally numbered functional requirements, and assumptions are tagged inline and indexed at the end. Screen-level interaction details belong to the UX workflow; this document constrains them without prescribing them.

## 1. Vision

db-adm-tui is the database tool that is already where the data is: one statically compiled terminal binary on the database host, reached through the SSH access an engineer already has. It replaces the ritual of tunnels, jump hosts, and GUI clients with one navigable tree of databases, schemas, tables, collections, users, and roles — the same tree whether the engine is PostgreSQL, MongoDB, or MySQL/MariaDB.

It is deliberately not DBeaver. It does not try to be an IDE, a schema designer, or a data editor. It is the tool for the small, urgent question inside a closed environment — what databases exist, what can this user do, where is the row I need, how large is this table — answered in one visit without leaving the terminal. It wins by being small, fast, predictable, comfortable, and honest about what it does not do.

The gap is specific: no surveyed terminal client combines on-server operation, MongoDB, SQL, and native privilege inspection in one static binary. The product speaks each engine's dialect and shows privileges in native terms; what it unifies is movement — one tree, one keyboard model, one set of views.

For v1, read-only is the default: the tool blocks write statements in the query bar until the user confirms a switch to read-write mode, and the UI offers no data editing or privilege changes. v2 extends the same surface with a privilege matrix, audited changes, and optional data edits. [NOTE FOR PM: the read-write switch is a v1 feature; "read-only" means the default mode, not an incapability.]

## 2. Features

### 2.1 Common navigation tree

**Description:** One tree serves every engine. A connection expands into databases; a database expands into schemas where the engine has them, then into tables, collections, and other object nodes. A dedicated Security node holds users and roles so the same navigation works whether the engine keeps them per cluster, per server, or per database. Selecting a table or collection opens context tabs for its data and details. The tree is read-only in v1: it exposes no actions that modify data, grants, or roles.

**Functional Requirements:**

#### FR-1: Connect and list databases

After a profile connects, the tree lists the databases visible to the connected user. Profiles and import are defined in §2.7.

**Consequences (testable):**
- The connection root shows the profile label and the engine.
- Databases the user cannot see are omitted; a permission error while listing shows an inline error, not a crash.
- An empty list shows an explicit empty state.
- A failed connection returns the user to the profile picker with the error attached to the profile.

#### FR-2: Expand a database into objects

The user can expand a database node to see the objects it contains.

**Consequences (testable):**
- PostgreSQL: database → schema → tables, views, and sequences, each as a node.
- MySQL/MariaDB: database → tables and views, plus sequences where the engine exposes them.
- MongoDB: database → collections and views.
- v1 lists database views and sequences as nodes; their detail and data views are deferred. Selecting one shows a `detail view not available in v1` placeholder.
- Children load lazily on expand, not when the connection opens.
- A failed child load leaves the parent expanded with an inline error.

#### FR-3: Security node for users and roles

Every connection exposes a Security node containing the engine's users and roles.

**Consequences (testable):**
- PostgreSQL: cluster roles with attributes and memberships.
- MySQL/MariaDB: users in `user@host` form and roles, with native scope labels.
- MongoDB: users and roles per database; cluster roles are reachable from the admin database.
- v1 is read-only: no action in the Security node modifies a user or role.
- Missing privileges show a `not permitted` state instead of an empty list or a crash.
- The detailed privilege view is defined in §2.8.

#### FR-4: Open details and data from the tree

`Enter` or `Space` on a table or collection opens its context tabs in the main pane.

**Consequences (testable):**
- The main pane opens the context tab for the selected object; a table or collection opens `Data`.
- The `Details` tab shows the statistics and structure defined in §2.2.
- The `Data` tab loads the first page defined in §2.3.
- `Tab` moves focus between the tree, the main pane, and the query bar without losing the current selection.
- `1`/`2`/`3` select a context tab; `[` and `]` cycle tabs.

### 2.2 Object details and statistics

**Description:** Selecting a table or collection fills the Details tab with an estimated count, on-disk size, indexes, and structure; exact count is on demand. MongoDB structure is a sampled field list.

**Functional Requirements:**

#### FR-5: Estimated count

The details pane shows an estimated row or document count for the selected object.

**Consequences (testable):**
- The value is labeled `estimated` with its source (catalog estimate, collection statistics, or equivalent).
- If the engine cannot provide an estimate, the row shows `unavailable`, not `0`.
- The estimate refreshes when the object is selected again.

#### FR-6: Size on disk

The details pane shows the object's on-disk size when the engine exposes it.

**Consequences (testable):**
- Values are human-readable (B, KB, MB, GB).
- PostgreSQL reports total relation size; MySQL/MariaDB reports data plus index length; MongoDB reports storage size and index size.
- If the value is unavailable, the row shows `unavailable`.

#### FR-7: Indexes

The details pane lists the object's indexes.

**Consequences (testable):**
- Each entry shows at least the index name, the columns or fields it covers, and whether it is unique.
- An object with no indexes shows an explicit empty state.
- Index metadata failures degrade to `unavailable` without hiding the rest of the pane.

#### FR-8: Structure

The details pane shows the object's structure in engine-native terms.

**Consequences (testable):**
- PostgreSQL and MySQL/MariaDB show columns with type, nullability, and default where available.
- MongoDB shows a sampled field list derived from the first loaded page, labeled `sampled from N documents`. [ASSUMPTION: sample size follows the initial page of 100 documents.]
- Each sampled field, including nested field paths, shows how many sampled documents contain it and the percentage, for example `present in 87 of 100 sampled documents (87%)`.
- If sampling cannot run, the structure row shows `unavailable`.

#### FR-9: Exact count on demand

`Ctrl+X s` computes the exact count for the selected object.

**Consequences (testable):**
- The key is `Ctrl+X s`, not `Ctrl+S` (terminal flow control).
- The running state is visible and the request can be cancelled.
- On completion the value replaces the estimate and is labeled `exact`.
- The count always covers the whole object; it does not count the current filtered view.
- On failure the estimate remains and the error is shown inline.
- The action is read-only.

### 2.3 Data browsing and search

**Description:** Selecting a table or collection opens a read-only data view with 100-row pages, explicit and end-of-data paging, a bounded full fetch, buffer search, and a server-side regexp filter on the focused field.

**Functional Requirements:**

#### FR-10: Initial data page

When a table or collection is opened, the system loads and displays up to 100 rows or documents.

**Consequences (testable):**
- The first request returns at most 100 rows or documents.
- The status line shows the loaded count and, when available, the estimated total with an `estimated` label.
- An empty object shows an explicit empty state.
- A failed fetch keeps the previous view and shows the error inline.
- The fetch is read-only.

#### FR-11: Explicit next page with `m`

The user can press `m` in the data view to fetch the next page.

**Consequences (testable):**
- The next page contains up to 100 rows or documents and is appended to the existing rows.
- At the end of the data, `m` is a no-op and the status line says `end of data`.
- The loaded count updates after each fetch.

#### FR-12: End-of-data gesture

While the cursor is on the last loaded row, two consecutive `PgDown` presses or three consecutive `Down` presses fetch the next page.

**Consequences (testable):**
- The first press at the end shows a hint: `press PgDown again` or `press Down twice more`.
- The gesture counter resets on any other key, after a fetch starts, or when focus moves.
- The fetch uses the active batch size: 100 normally, 10,000 after a capped full fetch.
- The gesture is disabled while a modal or filter prompt is open.

#### FR-13: Fetch all with a cap

`Ctrl+A` fetches all remaining rows or documents, capped at 10,000 per request.

**Consequences (testable):**
- If the remaining rows fit within the current window (10,000 rows or 64 MB, whichever comes first), all are loaded and the status line reflects the complete set.
- If more remain, exactly 10,000 are loaded and the status line shows the loaded window and that more are available.
- Subsequent `m` or end-of-data gestures fetch 10,000-row batches; the window slides, and rows outside it are re-fetched on scroll.
- The request is read-only and cancellable.
- The window cap is enforced per data view; it does not limit how far the user can page.
- Re-fetching a window uses the query's natural or explicit order; without a stable order, the status line warns that rows may repeat or be skipped.

#### FR-14: Buffer filter with `/`

`/` filters the loaded rows or documents incrementally, in the manner of less.

**Consequences (testable):**
- Typing filters the visible rows to those containing the pattern and highlights the matching text.
- Matching follows the global search case mode in §2.4; the default is case-sensitive.
- The prompt shows the pattern and the match count.
- `Esc` clears the filter and restores all loaded rows.
- The filter applies only to loaded rows; it never queries the server.
- If nothing matches, the view shows an explicit no-match state and the pattern remains editable.

#### FR-15: Match navigation with `n` and `N`

In a filtered view, `n` moves to the next match and `N` to the previous one.

**Consequences (testable):**
- The status line shows the current match position, for example `match 3 of 17`.
- Navigation wraps at the ends; the direction is visible in the status line.

#### FR-16: Server-side regexp filter with `Ctrl+F`

With a cell focused in the data view, `Ctrl+F` opens a regular-expression prompt and filters the whole table or collection by the current column or field.

**Consequences (testable):**
- If the data view does not have focus, the tool first offers to move focus there; on confirmation it proceeds.
- If the focused object is a database view or sequence with no data view in v1, it shows the placeholder.
- The prompt names the current column or field.
- The query is read-only and uses the engine's native regexp operator.
- Case sensitivity follows the global search case mode in §2.4 and is applied with the engine's native flags.
- On success, the view shows only matching rows or documents and a persistent banner: `Filtered by regexp on <field>; N matching rows loaded`, followed by `of ~M total rows (estimated)` when an object estimate is available, or `total unavailable` when it is not.
- Result paging follows FR-10–FR-13.
- `Esc` clears the filter and restores the previous unfiltered view when it is still cached; otherwise it reloads the initial page.
- An invalid regexp shows an inline error and keeps the previous view.
- Match navigation follows FR-15.

#### FR-17: Load status, cancellation, and errors

The data view always communicates what is loaded and whether it is complete or filtered.

**Consequences (testable):**
- The status line shows loaded count, the loaded window range, estimated total when available, filter state, and the global search case mode.
- At narrow widths, mode and case indicators are preserved and counts truncate first.
- A long-running fetch can be cancelled; the previous view remains.
- Connection loss, permission errors, and query timeouts are shown inline without crashing the tool.
- No fetch changes data; all data-view actions are read-only.

### 2.4 Global search behavior

**Description:** Search case sensitivity is a tool-wide mode, not a per-view setting. It applies to every search and filter surface, and its current state is always visible outside the work areas (the tree, details, data view, and query bar) so the user does not have to remember it.

#### FR-18: Global search case mode

`Ctrl+X i` toggles case-insensitive search for all search and filter surfaces.

**Consequences (testable):**
- The default is case-sensitive. [ASSUMPTION: default matches less without `-i`.]
- The second key is plain `i`; `Ctrl+I` is `Tab`.
- The status line shows the current mode, for example `case: sensitive` or `case: insensitive`, outside the work areas.
- The mode applies to the `/` buffer filter and the `Ctrl+F` server-side regexp filter.
- It does not change SQL or MQL execution or command completion.
- For a server-side regexp, the mode is translated into the engine's native flag: PostgreSQL `(?i)`, MySQL `REGEXP_LIKE(..., 'i')`, MariaDB `REGEXP`/`RLIKE` with an inline `(?i)` flag, MongoDB `$options: 'i'`.
- The mode persists for the session.

### 2.5 Query bar

**Description:** A query bar with one logical statement per execution, scoped to the current database and collection. SQL uses the engine's native dialect; MQL is restricted to `find`, `aggregate`, `count`, and `distinct`; results reuse the data view; read-only mode is enforced before execution. The bar may grow to three wrapped lines; it never becomes a multi-line editor.

**Functional Requirements:**

#### FR-19: Context-aware query input

The query bar is bound to the current database and, for MongoDB, the current collection.

**Consequences (testable):**
- SQL accepts one statement per execution; scripts and multiple statements are rejected. The bar may wrap the statement over up to three lines and never becomes a multi-line editor.
- MQL accepts only `find`, `aggregate`, `count`, and `distinct`; arbitrary JavaScript and shell escapes are rejected with a clear error.
- The MQL input does not require a `db.<collection>.` prefix.
- If no database or collection is selected, the bar is disabled with a hint.
- The dialect is the engine's current native dialect; the tool does not translate between engines.

#### FR-20: Execute, cancel, and report

`Enter` executes the statement when the completion popup is closed; `Ctrl+X Ctrl+X` executes from any work area (tree, details, data view, or query bar).

**Consequences (testable):**
- While a query runs, a running indicator and elapsed time are visible; `Esc` cancels.
- On completion, elapsed time and the returned row or document count are shown.
- On error, the engine message is shown inline and the statement text is preserved.
- Every execution is subject to the read-only guard in §2.6.

#### FR-21: Completion with `Ctrl+Space`

`Ctrl+Space` opens a completion popup for the word at the cursor.

**Consequences (testable):**
- Suggestions include dialect keywords and functions, visible databases, schemas, tables, and collections, columns or fields of the current object, and identifiers from statements executed earlier in the current profile.
- MQL suggestions include command names, query operators, and field paths.
- `Up`/`Down` navigate suggestions; `Enter` accepts the highlighted suggestion; `Esc` closes the popup; `Ctrl+Space` reopens it.
- Completion is contextual guidance, not semantic validation; the tool does not reject a statement because a suggestion was ignored.
- `Tab` moves focus and is never used for completion.
- Completion works in both the query bar and the edit modal.
- When no suggestions exist, the status line says `no suggestions`.

#### FR-22: Context help with `Ctrl+F1`

`Ctrl+F1` opens offline context help for the focused element or the word under the cursor, anywhere in the interface.

**Consequences (testable):**
- In text fields, help describes the word under the cursor; in the tree, details, and data view, it describes the focused element.
- Help covers SQL and MQL keywords, functions, operators, commands, and interface elements.
- v1 help covers every keyword, operator, command, and function offered by completion, plus every focusable interface element.
- Help content is regenerated by `make update-help` across all enabled drivers and ships with the binary.
- An unknown word or element shows `no help available`.
- Help opens in a modal or pane and is dismissed with `Esc`.
- Help requires no network access.

#### FR-23: Edit modal with `Ctrl+X Ctrl+E`

`Ctrl+X Ctrl+E` opens the current statement in a modal editor with the cursor at the first editable position.

**Consequences (testable):**
- The modal edits one logical statement; long statements scroll horizontally or wrap visually, but the tool does not become a multi-line SQL IDE.
- `Enter` executes and closes the modal; `Esc` cancels without executing.
- Completion and context help work inside the modal.
- On execution, the query bar shows the executed statement; on cancel, it keeps the original text.

#### FR-24: History with `Up` and `Down`

When the query bar has focus, `Up` and `Down` navigate executed statements, shell-style.

**Consequences (testable):**
- When the completion popup is open, `Up`/`Down` navigate suggestions instead of history.
- In other panes, `Up`/`Down` navigate that pane.
- History is per profile, persisted under the history path, and subject to the FR-43 redaction rule. [ASSUMPTION: history is per profile.]
- A recalled statement can be edited before execution; history is appended only when a statement is executed.
- Consecutive duplicate statements are collapsed. [ASSUMPTION: duplicates are collapsed.]

#### FR-25: Results presentation

Query results reuse the data view.

**Consequences (testable):**
- SQL results render as a table grid with columns and rows.
- MQL results render as a flat table: columns are the union of top-level fields in the loaded documents; nested objects and arrays render as compact JSON; missing fields render blank.
- Columns are ordered `_id` first, then first-seen order across loaded pages; new fields append to the right and existing columns never move.
- The table has no fixed column cap; horizontal scrolling handles wide or heterogeneous documents.
- Paging (`m`, end-of-data gesture, `Ctrl+A`), the `/` buffer filter, and `n`/`N` match navigation apply to query results.
- The server-side regexp filter (`Ctrl+F`) does not apply to arbitrary query results in v1; the query itself expresses the filter.
- Null and empty string are visually distinguishable.
- Result status shows elapsed time and loaded count.

#### FR-26: Query scope and safety

Queries run against the current selection and cannot bypass the read-only guard.

**Consequences (testable):**
- SQL executes against the current database; schema qualification is the user's responsibility.
- MQL commands run against the current collection.
- Multi-statement scripts, shell escapes, and arbitrary JavaScript are rejected.
- The tool does not rewrite the statement except for read-only enforcement.

### 2.6 Read-only guard and mode switch

**Description:** Every connection starts in read-only mode. The tool blocks write statements with a conservative per-engine blocklist before they reach the engine; where the engine supports a read-only session, that session is set as a second layer. Read-only mode is a product safeguard, not a security boundary: a privileged user can bypass it outside the tool. `Ctrl+X Ctrl+W` switches to read-write after confirmation, and a persistent indicator shows the current mode.

**Functional Requirements:**

#### FR-27: Read-only default

Every new connection starts in read-only mode, regardless of the connected user's privileges.

**Consequences (testable):**
- The mode indicator is always visible outside the work areas.
- Closing and reopening a connection resets the mode to read-only.
- The mode belongs to the open connection and is not persisted across sessions.
- Where supported, a read-only session is set as a second layer: PostgreSQL `default_transaction_read_only=on`; MySQL/MariaDB `SET SESSION TRANSACTION READ ONLY`. MongoDB has no session-level read-only.

#### FR-28: SQL write blocklist

In read-only mode, write and admin statements are blocked before they are sent to the engine.

**Consequences (testable):**
- The blocklist covers `INSERT`, `UPDATE`, `DELETE`, `MERGE`, `TRUNCATE`, `CREATE`, `ALTER`, `DROP`, `GRANT`, `REVOKE`, `COMMENT`, `COPY FROM`, `SELECT INTO`, `CALL`, `DO`, `VACUUM`, and `ANALYZE`.
- MySQL/MariaDB additions include `REPLACE`, `LOAD DATA`, `RENAME TABLE`, `SET GLOBAL`, `SET PERSIST`, `LOCK TABLES`, and `UNLOCK TABLES`.
- Write operations nested in CTEs, or inside `EXPLAIN ANALYZE`, are also blocked; `EXPLAIN` of a read statement is allowed.
- `SET TRANSACTION READ WRITE` and disabling read-only session settings are blocked.
- The tool names the reason and points to the mode switch.
- When the parser is in doubt, the statement is blocked.

#### FR-29: MongoDB write blocklist

In read-only mode, MongoDB write and admin commands are blocked before they are sent.

**Consequences (testable):**
- The blocklist covers `insert`, `update`, `delete`, `findAndModify`, `create*`, `drop*`, `collMod`, `createIndexes`, `dropIndexes`, `renameCollection`, `convertToCapped`, `grant*`, `revoke*`, `createUser`, `updateUser`, `dropUser`, `createRole`, `updateRole`, `dropRole`, and cluster or administration commands such as `replSet*`, `shard*`, `shutdown`, `setParameter`, and `compact`.
- Aggregation stages `$out` and `$merge` are blocked, as is `mapReduce` with an output target.
- Read commands from §2.5 and the inspection commands needed by the tree and details panes remain available.
- The tool names the reason and points to the mode switch.

#### FR-30: Mode switch and warnings

`Ctrl+X Ctrl+W` switches between read-only and read-write.

**Consequences (testable):**
- Switching to read-write requires explicit confirmation; switching back to read-only is immediate.
- Connections made with an account that has elevated rights, and tool launches as root, show a warning at connect and when switching to read-write.
- The indicator changes with the mode and is visible outside the work areas.
- The switch does not change the engine's own permissions; it only changes what the tool allows through its input.
- The tool sets a read-only session where supported and clears it only in read-write mode.

### 2.7 Connection profiles and import

**Description:** On startup the tool shows a profile picker with saved profiles, `n` for a new connection, and `i` for import. Profiles store non-secret connection details and an optional source-file pointer; passwords are resolved at connect time or prompted once when no source exists. PostgreSQL profiles import from `~/.pgpass`; MongoDB profiles import from selected `~/.mongo.*` files or `~/.mongo`.

**Functional Requirements:**

#### FR-31: Profile picker

On startup the tool lists saved profiles and lets the user connect with one.

**Consequences (testable):**
- Each entry shows a label, engine, host, and user; it never shows a password.
- The picker offers `n` for a new connection and `i` for import, alongside saved profiles.
- Selecting a profile connects; a failure keeps the user in the picker with the error attached to the profile.
- An empty profile list shows an explicit empty state that points to `n` and `i`.
- The picker is fully keyboard-navigable.

#### FR-32: PostgreSQL import from `~/.pgpass`

The tool imports PostgreSQL profiles from `~/.pgpass`, following psql semantics.

**Consequences (testable):**
- The format is `host:port:database:user:password`; `*` wildcards, `\` escaping, and `#` comments are honored.
- A file with permissions other than `0600` is ignored, as psql does.
- `PGPASSFILE` is honored when set; otherwise `~/.pgpass` is used.
- Each valid line becomes or updates a profile; the password is never copied into the profile.
- Malformed lines are skipped with a warning while valid lines still import.
- A missing file produces a clear empty state, not an error dialog.

#### FR-33: MongoDB import from `~/.mongo.*`

The tool scans for MongoDB connection files and lets the user choose which to import.

**Consequences (testable):**
- Every file matching `~/.mongo.*` is listed, plus `~/.mongo` when it exists.
- The user can select one or more files; each selected file yields one profile.
- The URI format is `mongodb://user:password@host1:port1,host2:port2/dbname?options`.
- Within a selected file, comment lines starting with `#` are ignored and the last parsed URI wins.
- A malformed selected file shows an error naming the file; the user fixes the file.
- The password is never copied into the profile.

#### FR-34: Password resolution at connect

The password is read from the source file when the connection opens.

**Consequences (testable):**
- The tool re-reads the source file on every connect; changes to the file are respected.
- A missing file, a changed selector, or a missing matching line fails the connection with an error naming the source file and profile.
- The password never appears in process arguments, logs, history, or error output.
- The source file is never written by the tool.

#### FR-35: Profile storage and refresh

Profiles store only non-secret details and a source pointer.

**Consequences (testable):**
- The stored fields are label, engine, host, port, database, user, and an optional source-file pointer.
- Re-import refreshes non-secret fields and preserves the profile identity.
- Profiles are stored under `$XDG_CONFIG_HOME/db-adm-tui` (default `~/.config/db-adm-tui`); state and history use `$XDG_STATE_HOME`.
- A profile without a source-file pointer prompts for a password at each connect; the password is never stored.

#### FR-36: Profile safety

Profile handling never exposes secrets.

**Consequences (testable):**
- The UI shows a redacted connection string, never the password.
- A password typed into any field is not persisted.
- Errors name the source file and selector but not the secret value.

#### FR-46: New connection

The user can create a connection at the picker without editing credential files.

**Consequences (testable):**
- `n` at the picker opens a form accepting a no-password URI: `mongodb://user@host[:port]/db`, `postgres://user@host:port/db`, or `mysql://user@host:port/db`.
- A URI containing `user:pass@` is rejected with an inline error; the password is resolved from a source file or prompted once.
- The tool resolves the password from the matching source file; if no match exists, it prompts once and never stores the value. [ASSUMPTION: prompt-once fallback.]
- On success, the connection opens and can be saved as a profile with non-secret details and an optional source pointer.
- A saved profile without a source pointer prompts for the password on each connect.
- Invalid URIs and failed connections show an inline error and keep the form.
- No password is written to the profile, history, or logs.

### 2.8 Privilege inspection

**Description:** Privilege inspection shows native grants and memberships for the connected user and for selected users and roles, with effective checks where the engine provides them. v1 is informational; no `GRANT` or `REVOKE` is offered. It does not build a unified cross-engine privilege model. Realizes UJ-1.

**Functional Requirements:**

#### FR-37: Connected-user privileges

The user can inspect the connected user's native privileges and roles.

**Consequences (testable):**
- PostgreSQL: database `CONNECT`, schema `USAGE` and `CREATE`, table and column privileges, `PUBLIC` grants, role memberships, and `has_*` checks where available.
- MySQL/MariaDB: `SHOW GRANTS`, active roles, and global, database, table, and column grants.
- MongoDB: `connectionStatus` with `showPrivileges`, `usersInfo`, roles, and inherited roles and privileges.
- Each entry is labeled direct or via role where the engine exposes enough information.
- A catalog the user cannot read shows a `not permitted` row instead of hiding the rest of the view.

#### FR-38: User and role detail

Selecting a user or role in the Security node shows its native grants and memberships.

**Consequences (testable):**
- The detail shows the user's or role's native scope: cluster for PostgreSQL roles, server for MySQL users and roles, database for MongoDB users and roles.
- Memberships are listed separately from object grants.
- Missing privileges show `not permitted` and keep the connected user's own view available.
- v1 offers no action that modifies a user or role. Realizes UJ-1.

#### FR-39: Effective versus granted

Where the engine can answer, the view distinguishes granted privileges from effective ones.

**Consequences (testable):**
- PostgreSQL uses `has_*_privilege` and role inheritance; `INHERIT FALSE` memberships are marked as requiring `SET ROLE`.
- MySQL/MariaDB shows granted roles and active roles separately.
- MongoDB uses `inheritedRoles` and `inheritedPrivileges` where available.
- Where effective computation is unavailable, the view shows native grants only with an explicit note.
- No cross-engine privilege vocabulary is invented; native terms are used verbatim.

#### FR-40: Read-only privilege view

Privilege inspection never changes privileges in v1.

**Consequences (testable):**
- No `GRANT`, `REVOKE`, role membership change, or user creation is reachable from the v1 view.
- The v2 privilege matrix builds on this view and is not part of v1. It is the Story 9 access matrix: Principal × Role membership and Principal/Role × Object privileges, opened by `Ctrl+X Ctrl+R` only in read-write mode.
- The matrix keeps native privilege names, audits every change, and shows native permission errors instead of impersonating the target user.
- Attempting a privilege-changing statement through the query bar follows the read-only guard in §2.6.

#### FR-41: Degradation and errors

Partial privilege data is honest about what is missing.

**Consequences (testable):**
- Permission errors are shown inline, per section, without crashing the tool.
- Available sections remain visible when another section fails.
- The view distinguishes `no privilege` from `not permitted to inspect`.

### 2.9 History and state

**Description:** Command history is a local audit trail of what the user executed. It is stored per profile in two formats so it is both machine-readable and pleasant to read. It never contains credentials.

**Functional Requirements:**

#### FR-42: History format and path

Executed statements are written to the history path in JSONL and human-readable TXT.

**Consequences (testable):**
- Both files live in the history directory: `~/.local/state/db-adm-tui/history` by default, or `$XDG_STATE_HOME/db-adm-tui/history` when set.
- Each entry records the profile, timestamp, engine, and statement.
- Files are created with `0600` permissions.
- The JSONL format is one JSON object per line.

#### FR-43: Redaction

History never stores a credential or password literal.

**Consequences (testable):**
- Credentials and connection URIs with passwords are redacted.
- A password literal typed in a statement is written to disk as `***`; the in-memory session may retain it.
- Error output and crash text follow the same redaction rule.

#### FR-44: Rotation

History growth is bounded.

**Consequences (testable):**
- Each format rotates at 10 MB and keeps 5 files.
- Rotation happens on write; no unbounded file grows.
- Rotation never loses the current entry.

#### FR-45: History failure behavior

History is best-effort and never blocks the work.

**Consequences (testable):**
- If the history path is not writable, the tool continues and shows a warning.
- A corrupted history file is skipped with a warning; new entries are still written.
- No history failure prevents a query from running.

## 3. Target User

### 3.1 Jobs To Be Done

- **Functional:** When I am already on a database host inside a closed environment, I want to see what exists and answer a specific question — access, data, size — without tunnels, jump hosts, or a GUI, so the task takes one visit instead of a context switch.
- **Emotional:** I want to be confident that looking around cannot change production.
- **Social:** I want to answer a teammate's access or data question quickly and hand back a precise answer.
- **Contextual:** I am in a terminal, often under time pressure, on a host where I cannot install a runtime or a GUI.

**Secondary:** a DBA or teammate who needs the same view without learning a new tool; v1 is not designed around their workflows, but the tree and panes should be legible to them.

### 3.2 Non-Users (v1)

- GUI-first users who want a desktop client.
- Application developers looking for an IDE, schema designer, or migration tool.
- Anyone who needs interactive data editing, schema changes, or privilege assignment in v1.
- DBAs looking for backup, monitoring, automation, or fleet management.
- Teams needing shared profiles, SSO, or centralized administration.

### 3.3 Key User Journeys

*Lightweight form: this is an internal tool with a single operator role; each journey is a short named scene.*

- **UJ-1. Mcgru answers an access question on a PostgreSQL host.**
  Mcgru is already SSHed into a production host. He starts db-adm-tui, picks the `prod-ro` profile, expands the `orders` database, opens Security, selects the `reporting` role, and sees its native grants: `CONNECT`, `USAGE` on `public`, `SELECT` on three tables. He answers the teammate and exits without changing anything.
  **Edge case:** one membership is `INHERIT FALSE`, so the privilege view marks it as requiring `SET ROLE` instead of counting it as effective.

- **UJ-2. Mcgru finds one customer's document in a closed MongoDB.**
  Mcgru is connected to a MongoDB host. He selects the `customers` collection, presses `/` and types the email; the buffer has no match, so he presses `Ctrl+F` on the `email` field, enters a regexp, and gets a filtered view with the banner that this is not the full data. He reads the document and clears the filter.
  **Edge case:** the regexp is invalid; the tool shows the error and leaves the previous view intact.

- **UJ-3. Mcgru checks a table before a migration.**
  Mcgru selects a 40 GB PostgreSQL table. The details pane shows an estimated row count, total size, and indexes; the estimate is enough to see the table is large but not for the migration plan, so he presses `Ctrl+X s` and gets the exact count. He notes the missing index and exits.
  **Edge case:** the exact count runs long; he cancels it and keeps the estimate.

- **UJ-4. Mcgru runs a read-only check on MySQL.**
  Mcgru connects to a MySQL host, selects the `billing` database, opens the query bar, types `SELECT COUNT(*) FROM invoices WHERE status = 'open'`, executes it, reads the count, and exits. No writes are attempted.
  **Edge case:** the guard misclassifies the statement as a write; it blocks execution and names the reason, and the user can switch to read-write only with confirmation.

## 4. Glossary

- **Active batch size** — the page size currently used by `m` and the end-of-data gesture; see Page.
- **Buffer filter** — `/` filter over loaded rows; never queries the server.
- **Completion** — `Ctrl+Space` suggestions from the current dialect and visible objects.
- **Connection profile** — non-secret connection details plus a source-file pointer; password resolved at connect.
- **Context help** — `Ctrl+F1` offline help for the focused element or word.
- **Data view** — read-only rows or documents for a table, collection, or query result.
- **Elevated rights** — superuser/root or a cluster/server admin role; engine-specific.
- **Engine** — PostgreSQL, MongoDB, or MySQL/MariaDB.
- **Estimated count** — engine approximation, labeled `estimated`.
- **Exact count** — whole-object count on demand, labeled `exact`.
- **Global search case mode** — tool-wide case sensitivity for `/` and `Ctrl+F`.
- **History path** — the directory `~/.local/state/db-adm-tui/history`, or `$XDG_STATE_HOME/db-adm-tui/history` when set.
- **MQL** — restricted MongoDB surface: `find`, `aggregate`, `count`, `distinct`; no arbitrary JavaScript.
- **Native terms** — engine-specific names, not translated.
- **Page** — up to 100 rows/documents; 10,000 after a capped full fetch.
- **Read-only mode** — product guard blocking write statements; not a security boundary.
- **Read-write mode** — mode after `Ctrl+X Ctrl+W` confirmation; writes allowed.
- **Recent identifiers** — identifiers from statements executed earlier in the current profile.
- **Reference host** — 2 vCPU, 4 GB RAM, local SSD, database on the same host, no other load. [ASSUMPTION: used to validate performance targets.]
- **Security node** — tree node exposing users and roles.
- **Server-side regexp filter** — `Ctrl+F` over the whole object on the focused field; result labeled filtered.
- **Selector** — the `~/.pgpass` line or `~/.mongo.*` file chosen at import; used to re-read the password at connect.
- **Source file** — `~/.pgpass`, `~/.mongo.*`, or `~/.mongo`.
- **SQL** — the engine's native SQL dialect.
- **Statement blocklist** — per-engine write/admin statements blocked in read-only mode.
- **v2** — privilege matrix, native command preview, audit trail, optional data edits.
- **v3** — sorting and export; not scoped here.

## 5. Cross-Cutting NFRs

### Performance

- **NFR-1: Cold start.** The binary starts and shows the profile picker in under 300 ms on the reference host. [ASSUMPTION: target; validate on the reference host.]
- **NFR-2: First data page.** The first 100 rows or documents load in under 1 second for a local database on the reference host; slower requests show progress instead of freezing the UI.
- **NFR-3: Input responsiveness.** Keypress feedback appears within 50 ms, and the completion popup opens within 100 ms on the reference host.
- **NFR-4: Large fetches.** A 10,000-row fetch shows progress, keeps the UI responsive, and can be cancelled.
- **NFR-5: Bounded memory.** Each data view keeps a sliding window of loaded rows or documents: at most 10,000 rows or 64 MB, whichever comes first. When the cap is reached, the oldest pages are evicted and re-fetched on scroll. The status line shows the current window range and the estimated total. [ASSUMPTION: exact caps to be validated in architecture.]

### Reliability

- **NFR-6: Error resilience.** Engine errors, permission errors, connection loss, and query timeouts produce inline errors; they never crash the tool or leave a half-rendered view. Reconnecting returns to the profile picker with the error attached.

### Security and privacy

- **NFR-7: Secret handling.** Passwords and connection URIs with passwords never appear in process arguments, logs, history files, crash output, or error messages. History and state files are created with `0600` permissions. Profiles store only non-secret details and a source-file pointer.
- **NFR-8: Terminal safety.** All data rendered from the database is sanitized against ANSI escape injection; database content cannot change the terminal title, clipboard, or output stream.
- **NFR-9: Read-only guard.** No statement matching the per-engine blocklist reaches the engine; statements the parser cannot classify are blocked. The guard cannot detect side effects inside read-looking statements such as `SELECT nextval(...)`, `setval(...)`, `pg_terminate_backend`, `SELECT ... FOR UPDATE`, or `NOTIFY`. It is a product safeguard, not a security boundary.
- **NFR-10: No telemetry or hidden egress.** The tool makes no network calls except to the database endpoints the user selected. There is no telemetry, analytics, update check, or cloud sync.

### Compatibility and deployment

- **NFR-11: Static binary.** The tool ships as a single statically compiled binary with no runtime dependencies, for Linux `x86_64` and `arm64`; other Unix-like hosts are best-effort. [ASSUMPTION: primary target is Linux servers.]
- **NFR-12: Terminal and keyboard.** The UI works at `80x24`, in UTF-8 terminals, without color; every action is reachable from the keyboard; status and mode are conveyed by text and shape, not color alone; every letter shortcut works in both Latin and Cyrillic layouts by physical key position. `Ctrl+S` and `Ctrl+Q` are not bound because terminals use them for flow control.
- **NFR-13: Engine versions.** PostgreSQL 14+, MySQL 8.0+, MariaDB 10.4+, and MongoDB 6.0+ are supported, with graceful degradation where a specific catalog or command is unavailable.
- **NFR-14: No daemon, no ports.** The tool runs in the foreground, opens no listening ports, and stores configuration and state in XDG paths.

## 6. Non-Goals (Explicit)

- Not a GUI and not a DBeaver replacement; no IDE, schema designer, migration tool, or dashboard.
- No interactive data editing, schema changes, or privilege assignment in v1; these are v2.
- No native-command preview (`Ctrl+X Ctrl+R`) in v1; it belongs to the v2 privilege matrix.
- No arbitrary JavaScript, Mongo shell, or multi-statement scripts.
- No cross-engine query translation; each engine's native dialect is used as-is.
- No backup, restore, monitoring, alerting, or fleet management.
- No telemetry, analytics, cloud sync, or update checks.
- No saved passwords or credential vault.
- No shared profiles, SSO, or team administration.
- No plugin system in v1.
- No engines beyond PostgreSQL, MongoDB, and MySQL/MariaDB in v1.
- No detail or data panes for database views and sequences in v1; they appear as tree nodes only.
- No column sorting or data export in v1; both are deferred to v3.
- No custom themes or UI extensions in v1.

## 7. MVP Scope

### 7.1 In Scope

v1 is the union of §2.1–2.9: connection profiles, import, and new connections; the common tree with Security; object details; data browsing and search; global case mode; privilege inspection; the query bar; the read-only guard; and history. NFR-1–14 apply.

### 7.2 Deferred and Out of Scope

- **v2:** privilege assignment matrix, native command preview, audit trail, optional data edits.
- **v3:** column sorting and data export.
- **Unscheduled:** detail and data views for database views and sequences; additional engines; plugin system; shared profiles, SSO, team administration.
- **Not planned:** telemetry, cloud features, saved passwords, credential storage.

## 8. Success Metrics

*Metrics are measured manually or locally; the product does not collect telemetry.*

### Primary

- **SM-1: Privilege answer time.** A privilege question is answered in under 30 seconds, measured over five runs of UJ-1. Validates FR-3, FR-37, FR-38, FR-39.
- **SM-2: Personal weekly use.** The tool is used on at least five of seven days of real on-host work, without falling back to psql or mongosh for browsing or abandoning it out of frustration. Validates the product as a whole.
- **SM-3: External tester success.** The first external tester completes two real tasks unaided in one session: find a record, and answer what a given user can do on a given database. Validates UJ-1 and UJ-2.
- **SM-4: No credential leaks.** A deliberate test finds zero credentials or password literals in history files, logs, process arguments, or error output. Validates NFR-7 and FR-43.

### Secondary

- **SM-5: First page latency.** First 100 rows load in under 1 second on the reference host. Validates FR-10, NFR-2.
- **SM-6: Bounded memory.** Memory stays within the window cap over 100 page fetches and 50 tree navigations. Validates NFR-5.
- **SM-7: Large fetch responsiveness.** A 10,000-row fetch stays cancellable and responsive. Validates FR-13, NFR-4.
- **SM-8: MySQL query check.** A read-only check query on MySQL (UJ-4) completes in one visit and returns a result. Validates FR-19, FR-20, FR-26.
- **SM-9: Connection setup.** Importing from `~/.pgpass` or a selected `~/.mongo.*` file, and creating a new no-password connection, both produce a working session without storing passwords. Validates FR-31–FR-36 and FR-46.

### Counter-metrics (do not optimize)

- **SM-C1: Do not add telemetry to measure success.** Counterbalances SM-1 through SM-9; measurement stays manual and local.
- **SM-C2: Do not relax the read-only blocklist to make a query pass.** Safety outranks convenience; counterbalances SM-2 and SM-3.
- **SM-C3: Do not increase memory caps or add "keep everything" modes to chase fetch-all convenience.** Bounded memory is part of the product; counterbalances SM-5 and SM-6.
- **SM-C4: Do not add features outside v1 scope to look more complete.** Small is the point; counterbalances SM-2 and SM-3.

## 9. Open Questions

1. **Public release posture.** Built for personal use first, kept ready for public release; the decision waits until v1 proves itself in daily use. Owner: author.

### 9.1 Architecture Inputs

These are not product questions; they are decisions handed to architecture:

- Exact parser coverage for the read-only blocklists in FR-28 and FR-29, including nested writes, CTEs, and engine-specific admin statements.
- Profile storage schema and XDG location, beyond the product constraint that profiles never store passwords.
- Stable ordering strategy for window re-fetch when the user's query has no explicit order.

## 10. Assumptions Index

- **§2.2 FR-8** — MongoDB structure samples the first 100 documents.
- **§2.4 FR-18** — case-sensitive is the default search mode.
- **§2.5 FR-24** — query history is per profile, and consecutive duplicates are collapsed.
- **§2.7 FR-35/FR-46** — profiles may omit a source pointer; passwords are then prompted per connect and never stored.
- **§5 NFR-1** — the 300 ms cold-start target is validated on the reference host.
- **§5 NFR-5** — the 10,000-row / 64 MB window caps are validated in architecture.
- **§5 NFR-11** — Linux servers are the primary target; other Unix-like hosts are best-effort.
- **§4 Glossary** — the reference host specification is a validation configuration, not a deployment requirement.