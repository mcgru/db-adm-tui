# Addendum: db-adm-tui brief

Supporting detail that does not belong in the one-to-two-page brief. Downstream consumers: PRD, UX, architecture.

## Landscape evidence (November 2026)

| Project | Stack, license | Engines | Status and notes |
|---|---|---|---|
| lazysql | Go, MIT | PostgreSQL, MySQL, SQLite, MSSQL, ClickHouse | ~4.4k stars, v0.7.0 (Oct 2026), alpha; telemetry in official builds; MongoDB issue #235 open, PR #322 pending |
| harlequin | Python, MIT | DuckDB, SQLite, PostgreSQL, MySQL/MariaDB, ODBC, BigQuery, Trino, Databricks, Cassandra | ~6.5k stars, v2.17 (Oct 2026); requires Python; Ctrl+Enter issues in some terminals (#71); no MongoDB |
| sqlit | Python, MIT | ~25 SQL DBMS | ~5.0k stars, active (Oct 2026); Python dependencies; no MongoDB; fuzzy filtering over millions of rows |
| dblab | Go, MIT | PostgreSQL, MySQL, SQLite, Oracle, MSSQL | ~3.2k stars, v0.55.2 (Nov 2026); single binary, zero dependencies, read-only; no MongoDB, no privilege management |
| usql | Go, MIT | psql-like: PostgreSQL, MySQL, MSSQL, Oracle, ClickHouse, Cassandra | ~10.2k stars, v0.21.6 (Sep 2026); no MongoDB |
| pgcli / mycli / iredis | Python, BSD-3 | PostgreSQL, MySQL/MariaDB, Redis (respectively) | Mature single-engine CLIs; not TUIs |
| mongosh | TypeScript/Node, Apache-2.0 | MongoDB | v2.14.2 (Nov 2026); REPL, not a TUI |
| vi-mongo | Go, Apache-2.0 | MongoDB | ~248 stars, v0.3.2 (Nov 2026); MongoDB only; hangs on large remote collections (#62, #86) |
| rainfrog | Rust, — | PostgreSQL | ~5.4k stars; PostgreSQL only |
| gobang | Rust, — | PostgreSQL, MySQL, SQLite | ~3.3k stars; abandoned since 2023 |
| mongotui | —, AGPL | MongoDB | 1 star, July 2026; the only Mongo-specific privilege UI found |

**Gap.** No TUI combines on-server operation, MongoDB, and SQL engines in one static binary. Privilege management is absent across the category. No surveyed TUI imports `~/.pgpass`. Keyboard conventions are inconsistent: broken Ctrl+Enter chords, hidden keybindings, arrow keys that differ by tool.

**Sources:** github.com/jorgerojas26/lazysql, github.com/tconbeer/harlequin, github.com/Maxteabag/sqlit, github.com/danvergara/dblab, github.com/xo/usql, github.com/dbcli/pgcli, github.com/dbcli/mycli, github.com/laixintao/iredis, github.com/mongodb-js/mongosh, github.com/kopecmaciej/vi-mongo, github.com/achristmascarl/rainfrog, github.com/TaKO8Ki/gobang, github.com/DevOpsPack/mongotui; sqlit HN discussion 46276002; PostgreSQL libpq-pgpass documentation.

## Credential file conventions

**`~/.pgpass`:** `host:port:database:user:password`; `*` wildcards; first matching line wins; `:` and `\` escaped with `\`; `#` comments; file must be `0600` or it is ignored; override with `PGPASSFILE` or the `passfile` connection parameter. Source: PostgreSQL libpq documentation.

**`~/.mongo.*`:** a user-specific convention, not a standard MongoDB file; `*` is any suffix (`~/.mongo.root`, `~/.mongo.myuser`), and a plain `~/.mongo` may also exist. On import, the tool scans and offers every matching file. Text file; comment lines start with `#`; one or more `mongodb://user:password@host1:port1,host2:port2/dbname?opts` lines; within the selected file, the last parsed line wins; if the file is malformed, the user fixes it. The tool reads it at connect time and never stores the password. Nearest standard artifacts — `~/.mongoshrc.js`, `~/.mongodb/mongosh/config`, `mongosh_repl_history` — are not connection profiles.

## Keymap — working decision

| Key | Action |
|---|---|
| `Enter` | Open/select; in an input or modal — execute/confirm and close |
| `Ctrl+X Ctrl+X` | Execute the current statement from anywhere |
| `Ctrl+X Ctrl+E` | Edit (cursor at the first editable position) |
| `Ctrl+X Ctrl+R` | Show the native command |
| `Ctrl+X Ctrl+W` | Toggle read-only / read-write (switching to read-write requires explicit confirmation) |
| `Ctrl+X i` | Toggle the global case-insensitive search mode; the status bar shows the mode (second key is plain `i`; `Ctrl+I` is `Tab`) |
| `Ctrl+Space` | Complete the next word from the current SQL/MQL dialect, schema objects, and context |
| `Up` / `Down` | Navigate query history while the query bar has focus; navigate suggestions while the completion popup is open |
| `Ctrl+F1` | Context help for the focused element or word under the cursor, anywhere in the interface |
| `F1` | Keymap help; layout-independent (`?` is an alias on Latin layouts) |
| `Ctrl+F` | Server-side regular-expression filter on the focused column or field; if focus is outside the data view, offer to move it first; result is labeled as filtered |
| `Ctrl+X s` | Compute the exact row or document count for the selected object (whole object, not the filtered view) |
| `/` | Filter the loaded rows or documents, less-style |
| `n` / `N` | Next / previous match in a filtered view |
| `m` | Fetch the next page of rows or documents |
| `PgDown` ×2 / `Down` ×3 at the end | Fetch the next page |
| `Ctrl+A` | Fetch all remaining rows or documents, capped at 10,000 per fetch; further `m` fetches 10,000-row batches |
| `Esc` | Cancel/back |
| `Tab` | Move focus between panes |
| `Ctrl+D` | Disconnect to the profile picker; confirm first if a query or count is running |
| `Ctrl+Left` / `Ctrl+Right` | Resize the tree width; persists per profile |
| `Ctrl+Up` / `Ctrl+Down` | Resize the query bar height (1–3 lines, step 1); persists per profile |

Letter shortcuts are layout-independent: the same physical key works in Latin and Cyrillic layouts. Avoid `Ctrl+S` and `Ctrl+Q` (terminal flow control). `Ctrl+X Ctrl+R` is reserved for the v2 privilege matrix and is not active in v1. At the profile picker, `n` opens a new connection form and `i` imports from source files.

## Read-only enforcement

- **Product guard, not a security boundary.** Read-only mode blocks write statements in the tool's input; a privileged user can bypass it outside the tool.
- SQL blocklist: `UPDATE`, `INSERT`, `DELETE`, `MERGE`, `CREATE`, `ALTER`, `DROP`, `TRUNCATE`, `GRANT`, `REVOKE`, `COMMENT`, and equivalent write/admin statements per dialect.
- MongoDB blocklist: `insert`, `update`, `delete`, `findAndModify`, `create*`, `drop*`, `collMod`, `createIndexes`, `dropIndexes`, `renameCollection`, `grant*`, `revoke*`, `createUser`, `updateUser`, `dropUser`, `createRole`, and cluster/admin commands.
- Second layer where supported: PostgreSQL `default_transaction_read_only=on`; MySQL/MariaDB `SET SESSION TRANSACTION READ ONLY`. MongoDB has no session-level read-only mode.
- Policy: every connection starts read-only; the switch to read-write is explicit and confirmed; elevated-rights connections and root launches add warnings.

## History and audit

- History: JSONL plus human-readable TXT under `~/.local/state/db-adm-tui/history`; rotation 10 MB × 5 files per format.
- In-session passwords remain available in memory; on disk, password literals are written as `***`. Credentials and password literals are never logged.
- Audit (v2, separate from history): privilege changes recorded as who, what, when, before/after state, and the native command executed.
