# db-adm-tui

Terminal database administration for PostgreSQL, MongoDB, and MySQL/MariaDB,
built to run on the database host inside a closed environment.

The repository now implements the connection layer, the real catalog tree, and
read-only data browsing with search: profiles and import, connect-time password
resolution, real open/ping/close, lazy database/schema/object navigation, a
Security node with users and roles, native access markers, a paged Data tab with
a bounded loaded window, an inline `/` filter with match navigation, a
server-side `Ctrl+F` regexp filter, a session-wide search case mode, object
statistics with exact count, index/structure details, native privilege
inspection, an access matrix for role membership and object privileges, a query
bar with Results, completion, context help, and history, and a read-only default
with an explicit read-write switch.

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

### Screenshots

![Profile picker](docs/screenshots/ss.profiles.png)

![PostgreSQL profile](docs/screenshots/ss.postgres.png)

![MongoDB profile](docs/screenshots/ss.mongodb.png)

![MySQL profile](docs/screenshots/ss.mysql.png)


### Profile picker

The tool starts on the profile picker.

| Key | Action |
|---|---|
| `Enter` | Connect, submit a form, import selected rows, expand a tree node, or open Data/details |
| `Up` / `Down` | Move the picker, import list, tree cursor, Data/Results row cursor, or details scroll; in the query bar, recall history |
| `PgUp` / `PgDn` | Scroll the picker/import list, tree, details, or completion; page the Data/Results grid |
| `Left` / `Right` | Collapse / expand the selected tree node; move the Data/Results cell cursor |
| `Home` / `End` | Jump to the first/last tree or import row; first/last Data/Results column |
| `Ctrl+Home` / `Ctrl+End` | Jump to the first/last row of the loaded Data range |
| `Ctrl+Left` / `Ctrl+Right` | Resize the tree by 2 cells (16 cells to half the terminal width) |
| `Ctrl+Up` / `Ctrl+Down` | Resize the query bar between 1 and 3 lines |
| `m` | Append the next Data/Results page |
| `Ctrl+A` | Append a full-fetch batch (Data) or drain the result cursor up to 10,000 rows |
| `` ` `` | Data/Results: expand/collapse the selected JSON object column into dotted subcolumns |
| `/` or `.` | Tree: quick search over loaded/expanded nodes. Data/Results: inline literal filter over loaded rows; `n`/`N` walk matches |
| `Ctrl+F` | Data tab: server-side regexp on the focused column or field |
| `Ctrl+X` then `i` | Toggle the session search case between insensitive and sensitive |
| `n` | New connection at the picker; next tree-search or filtered Data view match |
| `N` | Previous tree-search or filtered Data view match (wraps) |
| `i` | Import from `~/.pgpass`, `~/.mypass`, and `~/.mongo.*` |
| `e` | Edit the selected profile at the picker; with a role selected in the tree, open the privilege matrix focused on that role |
| `+` | Tree roles/users: create, grant, or assign. Data tab: insert a row/document. Requires `r/w` |
| `-` | Tree roles/users: revoke, remove, or drop. Data tab: delete the selected row/document. Requires `r/w` |
| `Space` | Select an import row, or open/expand the selected tree node |
| `Tab` / `Shift+Tab` | Move focus: workspace zones; fields in a modal |
| `1` / `2` / `3` / `4`, `[` / `]` | Select a context tab by position; tabs differ by object |
| `Esc` | Cancel a query or fetch, clear a filter, release Results, cancel an attempt, or close the current modal |
| `Ctrl+D` | Disconnect back to the picker |
| `Ctrl+Q` | Quit (also works in the workspace) |
| `Ctrl+X` then `Ctrl+X` | Execute the query bar statement from any workspace zone |
| `Ctrl+X` then `s` | Compute the exact count for the selected table/collection |
| `Ctrl+X` then `Ctrl+E` | Open the current statement in the edit modal |
| `Ctrl+X` then `Ctrl+R` | Open the access matrix in `r/w` (membership and object privileges) |
| `Ctrl+X` then `Ctrl+W` | Switch between read-only and read-write; read-write asks for confirmation |
| `Ctrl+Space` | Open completion for the word at the cursor; `Esc` then `Tab` also opens it in the query bar or edit modal |
| `Ctrl+F1` | Open offline context help for the focused element or word; `Enter` or `Ctrl+F1` again shows examples |
| `?` / `F1` | Help stub |
| `Ctrl+C` / `Ctrl+с` | Force quit, even below the 80×24 floor |

Latin and Cyrillic physical key positions resolve to the same actions
(`n`/`т`, `i`/`ш`, `Ctrl+Q`/`Ctrl+й`, `m`/`ь`, and the chord finals).

## Documentation

- [`docs/usage-modes.md`](docs/usage-modes.md) — practical `r/o` and `r/w`
  workflows, query examples, role/privilege matrix walkthroughs, and
  troubleshooting. Русская версия:
  [`docs/usage-modes.ru.md`](docs/usage-modes.ru.md).
- [`docs/roadmap-and-releases.md`](docs/roadmap-and-releases.md) — next
  implementation steps and the release/tagging checklist. Русская версия:
  [`docs/roadmap-and-releases.ru.md`](docs/roadmap-and-releases.ru.md).
- [`live-test/README.md`](live-test/README.md) — disposable PostgreSQL,
  MariaDB, and MongoDB fixtures, including a ready profile bundle in
  `live-test/home/` and `install-home.sh`. The manual TUI checklist is
  [`live-test/TUI-TEST-PLAN.md`](live-test/TUI-TEST-PLAN.md) (Russian).

## Tree and Security

After connecting, the workspace shows the databases visible to the connected
user plus a `Security` node. PostgreSQL databases show the `public` schema
first and fold every other schema under a `schemas` group; both expand into
tables, views, and sequences. MySQL/MariaDB databases expand directly into
tables, views, and sequences; MongoDB databases expand into collections and
views. Children load lazily on expand; a failed load stays attached to the
parent and `Enter` retries it.

**Tree quick search.** With focus on the tree, `/` (or `.`) opens a one-line
search bar over the rows already loaded and expanded: database names, schemas,
tables, collections, views, and security principals. Matching is literal and
follows the session case mode; matching text is highlighted with the match
style, including on the selected row. Typing jumps the selection to the first
matching row as soon as a pattern matches. `Enter` keeps the search, closes the
bar, and jumps to the first match; `n`/`N` jump to the next/previous match
below/above the current row with wrapping, and `Esc` clears the search. Typing
`n` while the bar is open inserts `n`; `Tab`, `Shift+Tab`, `Ctrl+D`, and other
global keys still work. The search never queries the database and never expands
a node by itself. With focus in the main pane, `/` continues to filter loaded
Data/Results rows as before.

`Security` expands into `users` and `roles` groups. PostgreSQL shows cluster
roles with attributes, MySQL/MariaDB shows `user@host` accounts and roles, and
MongoDB shows database-scoped principals. Missing privileges render
`not permitted`.

Tables, collections, and views carry a native access marker: `rw` for row-level
write access, `ro` for read without write, and `—` for no read. Sequences carry
no marker. An unavailable native check omits the marker instead of inventing
one. Markers cover table/schema/global grants in v1; column-only grants are not
inspected. The tree title counts loaded object rows across expanded nodes.

PostgreSQL exposes only the current database on its one live connection.
Expanding another database reconnects the same profile to it; a failed switch
returns to the picker with the profile row error. MySQL/MariaDB and MongoDB see
every database on one connection.

Selecting a table or collection opens the context tabs and defaults to Details.
Details shows identity, estimated count and size, exact count, and structure.
Press `Enter` to switch to Data and load its first page. The Privileges tab
shows native grants, memberships, and effective checks for the selected
context or principal.

## Object details and indexes

Tables and collections expose `Details`, `Indexes`, `Privileges`, and `Data`
tabs (`1`–`4`; `[`/`]` cycle). Selecting one in the tree opens Details and loads
its structure; `Enter` switches to Data and loads the first page. Moving the
cursor to the next table keeps the active tab: Details reloads structure,
Indexes reloads indexes, and Data loads a fresh 100-row page. Paging state
(`m`, `Ctrl+A`) does not carry over to the new table. Details shows identity,
statistics, and a Structure table. Statistics shows the estimated count with
its source or `unavailable`, the on-disk size, and the Mongo index size when
available; `Ctrl+X s` computes the exact whole-object count and `Esc` cancels
it. PostgreSQL/MySQL structures list columns with type, nullability, and
default; MongoDB samples the first 100 documents and lists field paths with
presence counts and percentages. Table cells are separated with `│` and `┼`
rules.

The Indexes tab has three sections: Details (name, unique, method), Statistics
(size and engine-specific extras such as cardinality, sparse, or TTL), and
Structure (covered columns or fields). An object with no indexes says so; a
failed metadata read shows an error without hiding the rest of the pane.

The Privileges tab reports native grants, memberships, and effective checks for
the selected object, database/schema, Security connection, or user/role. It
keeps PostgreSQL `has_*` checks and `requires SET ROLE`, MySQL `SHOW GRANTS`
plus granted/active roles, and MongoDB roles and inherited privileges in their
native terms. A failed section shows `not permitted` without hiding the other
sections. There is no `GRANT` or `REVOKE` action in read-only mode.

## Access matrix (Story 9b)

Structured role and grant changes live in a full-pane matrix opened with
`Ctrl+X Ctrl+R`. It is available only in `r/w`; in `r/o` the tool reports that
read-write mode is required and points to `Ctrl+X Ctrl+W` without switching
automatically.

The matrix has two panels:

- `1 Principal × Role` — membership. Rows are users and roles, columns are
  native roles, and a cell is `[x]` or `[ ]`.
- `2 Principal/Role × Object` — privileges. Rows are roles, plus users for
  PostgreSQL and MySQL so direct grants stay visible. Columns are the current
  database/schema scope and the tables, views, or collections loaded in the
  tree. MongoDB shows role rows only.

Use `Tab` or `Shift+Tab` to switch panels and `1`/`2` to select one directly.
Each panel keeps its own cursor, scroll position, and filter. Arrows move the
cell cursor, `Home`/`End` jump to an edge, `PageUp`/`PageDown` move a page,
`Space` toggles the default access for the cell, `e` opens details, `/` filters
rows and columns, and `Esc` closes the matrix (or clears an active filter
prompt first).

For a privilege cell, `Space` toggles the engine default (`CONNECT` on
PostgreSQL databases, `USAGE` on schemas, `SELECT` on SQL tables, `find` on
MongoDB). `e` opens the full native privilege list as a multi-select; apply the
diff with `Enter`. Built-in MongoDB roles are read-only. Targets and privilege
names are validated by the engine adapter, and unsupported combinations are
rejected before execution.

The current connection must hold native admin rights; the target user's login
and password are never needed. A `not permitted` result is shown as the native
error with a `Ctrl+D` reconnect hint. Successful changes are audited in the
existing history and refresh the matrix.

Role creation and drop remain in the tree: `roles` group + `+` creates a role,
and role + `-` can drop one. The role menu's grant/assign entries open the
matrix instead of a modal form; role + `e` opens the privilege panel with that
role selected.

## User administration (Story 9c)

User changes are available only in `r/w`. In `r/o`, `+`/`-` on the users group
or a user report that read-write mode is required and point to
`Ctrl+X Ctrl+W`.

Create a user from `Security → users → +`:

- **PostgreSQL:** enter a login role name and password. The role is created with
  `LOGIN` and no extra grants; use the access matrix afterwards.
- **MySQL/MariaDB:** enter a user name, host (`%` is the default), and password.
  The account is created as `'user'@'host'` with no extra grants.
- **MongoDB:** enter a user name and password in the current database. The user
  is created with an empty role list; use the matrix to assign roles.

The password is masked in the form. It is written to the matching credential
file only after the native account is created:

- PostgreSQL: `~/.pgpass` or the active `PGPASSFILE`;
- MySQL/MariaDB: `~/.mypass` or `MYPASSFILE`;
- MongoDB: `~/.mongo.<USER>-in-<DATABASE>` with a full `mongodb://` URI.

For PostgreSQL/MySQL, the matching old line is commented with `#` and the new
line is inserted immediately below it. For MongoDB, the old matching URI line is
commented and the new URI is appended. Unrelated lines are preserved. The file is
rewritten atomically with `0600` and its directory is kept `0700`. The password
never enters a profile, history, log, argv, or error text.

After a successful create the tool offers `Save <label> as a profile?` with
`Enter` to save and `Esc` to skip. This only writes the optional profile; it
does not switch the active session to the new user. Dropping a user is available
from `Security → users → user → -`; dropping the currently connected user is
refused locally.

User creation/drop is audited in history with the native statement (password
redacted), mode, status, and affected count. Row/document insert and delete are
covered under Row and document changes.

## Data browsing

The Data tab opens for every table and collection. Moving the tree cursor only
selects an object; press `Enter` or focus the Data tab to load the first page.
It shows up to 100 rows or documents with a pinned row-number gutter, column
headers, and a status line such as `loaded 100 of ~12,480` or
`loaded 100 · total unavailable`.

- With Data focused, `m` appends the next batch: 100 rows normally, or the
  full-fetch batch after `Ctrl+A`. Each `Ctrl+A` also grows the retained window
  by the same amount, so repeated presses accumulate rows instead of sliding
  the start forward. The window starts at 10,000 rows and grows with each
  `Ctrl+A`; the 64 MB byte cap still applies. (The batch is currently 1,000
  rows for scroll debugging; the intended value is 10,000.)
- `Ctrl+Home` and `Ctrl+End` jump to the first and last row of the loaded range.
  If the front was evicted, `Ctrl+Home` fetches the whole missing prefix in one
  request and keeps the rows already in memory.
- `` ` `` expands only the selected Data column when its loaded cells contain
  JSON objects; this works in every engine, including a text column that holds
  JSON. `profile` with `{email, name}` becomes `profile.email` and
  `profile.name`, while other JSON columns stay collapsed. Subcolumns are the
  ordered union of keys across the loaded rows; missing or non-object values
  render empty, explicit nulls render `null`, and nested objects or arrays stay
  compact JSON. Pressing `` ` `` again collapses that column. A non-JSON column
  reports `selected column has no JSON object fields`.
- At the last loaded row, press `PgDown` twice or `Down` three times to fetch
  the next page. The first press explains the gesture.
- A forward fetch that would exceed the retained-window cap drops the oldest
  rows and advances the window start; scrolling above the start prepends the
  previous batch and drops rows from the tail. The status shows a range such as
  `rows 5,001–15,000 of ~15,000`; `m` after a full window slides forward until
  the next `Ctrl+A` grows the cap.
- The stable order comes from the primary key (PostgreSQL/MySQL) or `_id`
  (MongoDB). Without one, the pane warns
  `no stable order; rows may repeat or skip` and browsing still works.
- PostgreSQL and MySQL keep driver column order. MongoDB unions top-level
  fields across loaded pages, `_id` first, then first-seen order; nested
  documents and arrays render as compact JSON (MongoDB extended JSON for
  ObjectID/date/binary wrappers). Missing fields stay empty, explicit nulls
  render `null`, empty strings render blank, binary values render as
  `<binary N bytes>`, and long widths truncate with `…`.
- Arrows move the row/column cursor, `Home`/`End` jump to the first/last
  column, and `PgUp`/`PgDn` move a screen. Wide rows scroll horizontally while
  the row-number gutter stays pinned.
- `Esc` while a page is loading cancels it, keeps the previous rows, and shows
  `cancelled`. A failed page keeps the previous window and shows an inline
  error; `m` retries the next page. A failed first page shows
  `press Enter to load rows` and retries with `m`.
- Inline cell editing, row updates, sorting, and export are not present in the
  Data tab; `+`/`-` insert or delete one row/document in `r/w`. Queries run from
  the query bar.

## Row and document changes (Story 9a)

Single-row insert and delete are available only from the Data tab. Results is
read-only; row updates, inline cell editing, sorting, and export are not part of
this slice. Both actions require `r/w`: in `r/o`, `+`/`-` report that read-write
mode is required and point to `Ctrl+X Ctrl+W` without switching automatically.

**Insert (`+`).** The SQL form mirrors the loaded grid columns. Each field is
`val`, `NULL`, or `DEFAULT`, switched with `Ctrl+N` and `Ctrl+D`. Fields start
in `DEFAULT`; typing switches one back to `val`. `Tab`/`Shift+Tab` moves between
fields, `Enter` inserts, and `Esc` cancels. The statement uses quoted identifiers and bound parameters,
and the engine casts each text value to the native column type. `DEFAULT` omits
the column; an all-default row uses `DEFAULT VALUES` on PostgreSQL or
`() VALUES ()` on MySQL/MariaDB. MongoDB opens an extended-JSON document editor
instead and calls `insertOne`.

**Delete (`-`).** The confirmation names the profile, object, and key
(`id=...` or `_id=...`); `Enter remove` executes and `Esc` cancels. The row
identity is the stable Data order key: the native primary key for PostgreSQL and
MySQL/MariaDB, or `_id` for MongoDB. If the page has no stable key, or a key
value cannot be represented safely, delete is blocked locally and no engine call
is made.

After a successful mutation the Data tab reloads from offset 0, the status line
keeps the native statement, and the existing history records the mode, status,
and affected count. Native engine errors stay visible.

## Data search and case mode

Search is read-only and never changes paging rules. Both search surfaces follow
the session case mode, which starts **case-insensitive** and is shown in the top
status line as `case: insensitive` or `case: sensitive`.

**Buffer filter (`/`).** `/` opens a one-line filter bar over the rows already
loaded; it matches a literal substring and never queries the database. The view
narrows as the pattern is typed, every occurrence in the visible cells is
highlighted with the match style (fill plus underline, so it is not color-only),
and the bar shows `match X of N`. `Enter` closes the bar and keeps the filter;
`Esc` clears it and restores every loaded row. An empty pattern also clears the
filter. `n` and `N` move through matching cells with wrapping; while a filter is
active, `n` never opens a new connection. A pattern with no match shows
`No matches for "<pattern>". Esc clears the filter.` and `0 of N loaded`, and
stays editable. Leading and trailing spaces are part of the pattern. While the
bar is open, text editing wins: `m`, `PgDown`, and `Down` at the last row do not
fetch a page. `Tab`, `Shift+Tab`, `Ctrl+D`, `F1`, and `Ctrl+X` chords still act
globally. Filters reset on an object or tab change.

**Server filter (`Ctrl+F`).** With a cell focused, `Ctrl+F` names its column or
field (`regexp on email`) and `Enter` runs one parameterized, read-only regexp
query. PostgreSQL uses `~`/`~*`, MySQL/MariaDB lowercases both sides for the
insensitive default and uses `REGEXP BINARY` for the sensitive path, and MongoDB
uses `$regex` with `$options: "i"` when insensitive. The result is paged by `m`
and `Ctrl+A` exactly like an unfiltered window. The status line counts
`matching rows`; the banner reads
`Filtered by regexp on <field>; N matching rows loaded` and labels any catalog
total as `(table estimate)` or `total unavailable`. `n`/`N` wrap through the
loaded matching rows. SQL filters the base column even when a JSON subfield is
focused; MongoDB filters the dotted path. An empty pattern clears an active
filter and closes the bar. An invalid regexp or engine error keeps the previous
view, shows the error inline, and leaves the pattern editable. `Esc` restores the
cached unfiltered view without a query (or reloads the first page if no cache
exists). `Ctrl+F` does not apply to Results in v1.

Regexp filters run on the server and can scan the whole table: an unanchored
`$regex`, PostgreSQL `CAST(... AS text)`, and MySQL `LOWER(...)`/`BINARY` forms
generally cannot use an ordinary index. Expect longer scans on large tables and
keep `Esc` ready to cancel a fetch.

**Case mode (`Ctrl+X i`).** The toggle applies to both filters: an active buffer
filter recomputes its matches immediately, and an active server filter refetches
from offset 0. `Ctrl+X i` also works while a prompt is open. With no filter, only
the top-status indicator changes. The mode lasts for the session and is not
persisted.

**Esc order in the Data view.** Cancel a running fetch first, then clear the
active filter or close its prompt, then leave the tab or pane. A failed fetch
keeps the previous rows; a failed case toggle restores the previous filter and
case.

All data reads are read-only and parameterized. Values never become SQL text,
and identifiers are quoted per engine. Views and sequences have no Data tab.

## Query bar and Results

The bottom bar is bound to the current database and, for MongoDB, the selected
collection. It accepts one logical statement: SQL passes through unchanged;
MQL accepts only `find`, `aggregate`, `count`, and `distinct`, with an optional
`db.<collection>.` prefix, and rejects arbitrary JavaScript. Multi-statement
input, shell escapes, and write statements are blocked before an engine call
while read-only mode is on. The mode is shown as `r/o` or `r/w` next to the
database; `Ctrl+X Ctrl+W` switches, and enabling read-write requires
confirmation.

The tree width and query height are adjustable with `Ctrl+Left`/`Ctrl+Right`
and `Ctrl+Up`/`Ctrl+Down`; the tree clamps to 16 cells through half the
terminal, the query bar to 1–3 lines. Both persist per profile in
`$XDG_STATE_HOME/db-adm-tui/layouts.json` (default
`~/.local/state/db-adm-tui/layouts.json`). A smaller terminal clamps the
display without overwriting the saved size. The query bar wraps the same
logical statement; it never becomes a multi-statement editor.

`Enter` runs the statement when the completion popup is closed, and
`Ctrl+X Ctrl+X` runs it from the tree, details, Data, or query zone. While a
query runs, the status line counts elapsed time and `Esc` cancels. A successful
query opens a transient `Results` tab with the statement's rows; SQL keeps the
driver column order, while MongoDB unions top-level fields (`_id` first).
Errors keep the statement in Results and in the query bar, and `Esc` reselects
the object and its previous tab. Columns share the full pane width when the
loaded set fits; wide sets keep natural widths and scroll horizontally. Results use the same cursor, column scrolling,
row selection, `/` literal filter, `n`/`N` matches, `m` paging, and `Ctrl+A`
drain as the Data grid; `Ctrl+F` does not apply to Results in v1. The result
cursor is forward-only and materialized up to 10,000 rows or 64 MB; `Esc`
releases it, and other engine-backed actions report `results open — Esc to
release` until then.

`Ctrl+Space` opens completion for the word at the cursor. Suggestions combine
dialect keywords and functions, loaded tree/database/schema/object names,
columns or fields from the current object, and identifiers from earlier
statements. `Up`/`Down` and `PgUp`/`PgDn` move through the popup, `Enter`
accepts, `Esc` closes, and an empty result says `no suggestions`. Completion
works at the beginning, middle, or end of an identifier and accepting a
suggestion replaces the whole word. In MQL, a key immediately before `:` is
also used as the completion term, so `$o:` offers `$or` and keeps the colon.
The popup opens above the cursor column in the query bar and edit modal.
In the query bar and edit modal, `Esc` then `Tab` opens the same popup without
changing focus; `Tab` alone still moves focus, never completes. Accepting a
Mongo array operator such as `$or`, `$and`, `$in`, or `$nin` adds `: [  ]` with
the cursor inside unless the text after the word already has a colon/array;
accepting a function term such as `COUNT`, `COALESCE`, `ObjectId`, or `ISODate`
adds `()` with the cursor inside unless an opening parenthesis is already
present.

`Ctrl+X Ctrl+E` opens the current statement in a one-field edit modal with the
cursor at the start. `Enter` executes and closes it; `Esc` cancels and leaves the
query bar unchanged. Completion and context help work inside the modal.

`Ctrl+F1` opens offline help for the word under the cursor in the query bar or
edit modal, or for the focused tree/details/data/results element. Every
completion term has an entry; an unknown word says `no help available`. Inside
the help modal, `Enter` or a second `Ctrl+F1` replaces the short description
with example-driven help. For `find`, that includes `$and`, projection, and
chained sort/limit examples. `Up`/`Down` and `PgUp`/`PgDn` scroll long examples;
`Esc` closes the modal.

Executed statements are stored per profile at
`$XDG_STATE_HOME/db-adm-tui/history/commands.jsonl` and `commands.log`
(default `~/.local/state/db-adm-tui/history`), created `0600`, with `***` for
password literals. Each format rotates at 10 MB and keeps five files total.
`Up`/`Down` in the query bar recall them, consecutive duplicates are collapsed,
and a history write failure warns without blocking the query. Each entry
records the mode it ran under (`r/o` or `r/w`).

## Read-only mode

Every connection starts in read-only mode and resets there on reconnect or a
PostgreSQL database switch. The mode belongs to the live session and is never
written to a profile.

While read-only, the guard blocks write and admin statements before they reach
the engine and the status line names the reason with
`Ctrl+X Ctrl+W to switch.` The guard is a product safeguard, not a security
boundary: it does not change the account's native permissions, and a privileged
user can bypass it outside the tool. PostgreSQL sets
`default_transaction_read_only = on`; MySQL/MariaDB sets
`SET SESSION TRANSACTION READ ONLY`. MongoDB has no session-level read-only, so
its one-time connect notice says `Read-only is enforced by the tool; MongoDB has
no session-level read-only.` and the guard is the only layer.

`Ctrl+X Ctrl+W` asks for confirmation before enabling read-write with
`Writes will run on <profile>`, then changes the badge to `r/w` and says
`read-write mode on. Read-only guard is off.` Switching back to `r/o` is
immediate and needs no confirmation. Mode changes are refused while a query
result cursor is open; press `Esc` to release Results first.

Superuser/root and server-admin accounts carry a persistent `privileged` tag in
the top status line. A connection with elevated rights warns that read-only mode
is a product guard, not a security boundary; the read-write confirmation repeats
the warning. Running the tool as the OS root adds the same warning at connect.
PostgreSQL elevated detection covers `rolsuper`, `rolcreaterole`, and
`rolcreatedb`; MySQL/MariaDB covers root and broad server grants such as
`SUPER`, `SYSTEM_USER`, `SYSTEM_VARIABLES_ADMIN`, `ALL PRIVILEGES ON *.*`, and
`WITH GRANT OPTION`; MongoDB covers server-wide roles and the `anyAction`
privilege.

## Profiles

Profiles live at `$XDG_CONFIG_HOME/db-adm-tui/profiles.json` (default
`~/.config/db-adm-tui/profiles.json`). The directory is `0700` and the file is
`0600`. Schema v1 stores only `label`, `engine`, `host`, `port`, `database`,
`user`, an optional `allow_no_tls` flag, an optional source pointer,
`created_at` / `updated_at` / `last_used`, and the optional `last_object`
selection. **No password is ever written to a profile.**

On quit or disconnect, the selected table or collection is saved as
`last_object`; the next connect expands the tree back to it. Data-grid cursor,
scroll, and filters are intentionally not saved.

PostgreSQL connections require TLS by default (`sslmode=require`). A profile
that targets a server without TLS can set `"allow_no_tls": true` (the
hand-edited aliases `allowNoTLS` and `notls` are also accepted). The connector
always tries TLS first; only a connection-level failure falls back to a
plaintext retry. Auth, permission, and configuration errors are not retried.

Press `e` at the picker to edit the selected profile. Label, host, port,
database, and user are editable; the engine and credential source stay fixed.
For PostgreSQL the form also toggles `allow_no_tls`. Changing a connection field
changes the profile key, but the old entry is replaced instead of orphaned, and
`created_at` / `last_used` survive the edit.

If `profiles.json` is corrupt or unreadable, the picker shows
`Profile storage is unreadable. Showing no profiles.` and stays usable; `n` and
`i` still work. Import and save refuse to overwrite an unreadable store; fix or
remove the file by hand before saving again. The picker source column warns
`ignored`, `missing`, `stale`, or `error` when a source is unusable. One profile
is kept per selected `~/.mongo.*` source file, so saving a new URI that matches
that source refreshes that profile (including its stored non-secret Mongo host
list and options); a changed `~/.pgpass` or `~/.mypass` selector leaves the old
profile visible as `stale` and creates a new one on re-import.

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

### MySQL/MariaDB (`~/.mypass`)

The format is the same as `.pgpass`:

```text
host:port:database:user:password
```

- `MYPASSFILE` is honored; otherwise `~/.mypass` is used.
- Owner-only permissions are required; a file readable by group/other is
  ignored.
- `#` comments, `\` escaping, and `*` wildcards work as in `.pgpass`.
- `*` matches any host, port, database, or user; the first matching line wins.
- A matching line is attached to the profile as `source: ~/.mypass`; the
  password is read at connect time and never stored in the profile.
- A missing file is an empty state, not an error.

### MongoDB (`~/.mongo.*`)

Every `~/.mongo.*` file plus `~/.mongo` is listed, one row per file. Select the
files to import with `Space`, then press `Enter`. Within a file, `#` comment
lines are skipped and the last valid `mongodb://` URI wins. `mongodb+srv://` is
not supported in v1. A malformed line makes that file show an `error` tag naming
the file and it does not import, while other files remain selectable. The source
selector preserves non-secret options such as `authSource`, `replicaSet`, and
`tls`; password keys are stripped before the profile is written.

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

On submit, the tool looks for a matching `~/.pgpass`, `~/.mypass`, or
`~/.mongo.*` source. If one exists, its password is used; otherwise the form
reveals a masked one-time password field. If a matched source password is rejected, the form
offers a one-time password instead and the saved profile becomes prompt-based.
On success the picker offers
`Save <label> as a profile? Enter save · Esc skip`. Saving stores the
non-secret fields and any matched source pointer, never the password.

## Password behavior

- Passwords are resolved from the saved source **on every connect**; changes to
  `~/.pgpass`, `~/.mypass`, or the selected `~/.mongo.*` file are respected.
- A saved profile with no source opens a centered masked
  `Password for <profile>` modal. `Enter` connects and `Esc` returns to the
  picker. The value is never stored.
- Credential files are read at connect. User creation writes/rotates the matching
  entry, but the password never enters a profile, history, log, argv, or error
  text. This is the v2 credential-file exception to the v1 read-only-source
  rule.
- The resolved password is retained in memory for the life of the session so a
  PostgreSQL database switch can reconnect; it is cleared on disconnect or
  process exit and is never written to disk.
- Errors name the profile and source file but never the secret.
- Source-derived text, labels, filenames, and errors are sanitized of ANSI and
  control sequences before rendering.

## Security and release

Passwords are resolved from the source at connect time and are never written to
profiles, history, argv, or error text. Driver errors are redacted against the
resolved secret and against password literals such as `PASSWORD '...'`,
`IDENTIFIED BY '...'`, URI userinfo, and JSON `password`/`pwd` pairs before they
reach the screen or the history file. History files are `0600`; each format
rotates at 10 MB and keeps five files total.

Database content, query errors, pasted text, filenames, and profile fields are
stripped of ANSI/control sequences and bidi overrides before rendering. The
offline help and completion corpus lives in the binary. The release build is a
single static Linux binary with version, commit, and build date embedded; it
opens no listening port, runs no daemon, and makes no telemetry or update call.

```sh
./bin/db-adm-tui --version
make release
```

`make release` writes `dist/db-adm-tui_<version>_linux_amd64.tar.gz` and
`dist/db-adm-tui_<version>_linux_arm64.tar.gz` with stripped, statically linked
binaries.

## Live test fixtures

`live-test/` contains disposable PostgreSQL 18, MariaDB 11.4, and MongoDB 8.2
stacks with local bind mounts, seeded databases, ordinary/admin/root users, and
an `rs0` single-node MongoDB replica set. Run `live-test/up.sh` on a Docker host;
see `live-test/README.md` for ports, credentials, SSH tunnels, and reset/move
instructions.

## Checks

```sh
make test    # go test -race ./...
make lint    # go vet ./...
make verify  # reject listening/http/telemetry code in cmd and internal
make build   # CGO_ENABLED=0 static binary in bin/
make release # linux amd64/arm64 tarballs in dist/
```

Golden views live in `internal/tui/testdata`. Regenerate them after an
intentional layout change:

```sh
go test ./internal/tui -run TestGolden -update
```

The story specs live at
`_bmad-output/specs/spec-db-adm-tui/stories/4-data-browsing-and-paging.md` and
`_bmad-output/specs/spec-db-adm-tui/stories/4b-data-search-and-case-mode.md`.
