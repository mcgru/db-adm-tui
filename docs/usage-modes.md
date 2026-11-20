# Using db-adm-tui in r/o and r/w

_[English](usage-modes.md) | [Русский](usage-modes.ru.md)_

Every connection starts in read-only mode (`r/o`). The mode is a property of the
live session: it is shown next to the database in the top status line, is never
stored in a profile, and resets to `r/o` on reconnect and on a PostgreSQL
database switch.

Read-only mode is a product guard, not a security boundary. It blocks statements
before they reach the engine and, where the engine supports it, also applies a
native session read-only layer. Native account permissions still decide what is
actually possible.

| | `r/o` | `r/w` |
| --- | --- | --- |
| Tree, details, indexes, statistics, exact count | yes | yes |
| Data browsing, paging, filters, JSON expansion | yes | yes |
| Privileges tab | yes | yes |
| Read SQL/MQL queries | yes | yes |
| Write/admin SQL | blocked | yes, with native permissions |
| MQL `$out` / `$merge` | blocked | yes, with native permissions |
| Role/privilege matrix | blocked | yes, with native admin rights |
| Create/drop role | blocked | yes |
| Create/drop user | blocked | yes, with native admin rights |
| Row/document insert/update/delete | blocked | yes, one cell or a selected row set |

At the profile picker, `e` edits the selected profile and `Del` or `-` deletes
it with confirmation. Deleting a profile never deletes or modifies its
`.pgpass`, `.mypass`, or `.mongo.*` source file. Editing a connection field
moves the matching source entry to the new tuple with the same password.

## Read-only mode (`r/o`)

Connect normally; no extra key is needed. The top line shows `r/o`.

### Inspect a database

1. Select a profile and press `Enter`.
2. Expand the database and schema with `Right`.
3. Use `/` to search the loaded tree, `n`/`N` to walk matches, and `Enter` to
   jump to the match.
4. Select a table or collection and open its tabs.

The Details tab shows identity, statistics, structure, and access markers.
`Indexes` and `Privileges` show native metadata. `Ctrl+X s` computes the exact
count for the selected table/collection.

### Browse and filter data

With a table or collection selected, open `Data`:

```text
Enter        open the object
Tab          move focus to the main pane
/            filter the loaded rows/documents
n / N        next / previous match
m            append the next page
Ctrl+A       append a full-fetch batch
Ctrl+Home    jump to the first loaded row
Ctrl+End     jump to the last loaded row
`            expand the selected JSON column
```

`Ctrl+F` adds a server-side regexp filter on the focused SQL column; `Esc`
clears the filter or releases Results.

### Insert, update, or delete rows/documents

Open the Data tab in `r/w`. `+` opens the insert form: SQL columns start in
`DEFAULT` and accept `val`, `NULL`, or `DEFAULT` (`Ctrl+N`/`Ctrl+D`), while
MongoDB uses an extended-JSON document editor. `e` edits the focused cell or
field; primary-key columns and MongoDB `_id` are read-only. `Space` toggles row
selection: `-` deletes the current or every selected row/document, and `e`
applies the focused column's new value to all selected rows. A batch uses one
native statement. The confirmation names the profile, object, and key(s), and
the loaded page must have a stable key; otherwise the action is blocked locally.
After success Data refreshes and the operation is written to history with the
affected count. Results is read-only.

### Run read-only SQL

Put the statement in the query bar and press `Enter`, or use `Ctrl+X Ctrl+X`
from any workspace zone:

```sql
SELECT id, name, status FROM app.customers ORDER BY id;

WITH active AS (
  SELECT id, name FROM app.customers WHERE status = 'active'
)
SELECT * FROM active ORDER BY name;

EXPLAIN (ANALYZE, BUFFERS)
SELECT * FROM app.customers WHERE id = 1;

SHOW search_path;
SELECT column_name, data_type
FROM information_schema.columns
WHERE table_schema = 'app' AND table_name = 'customers'
ORDER BY ordinal_position;
```

MySQL/MariaDB examples:

```sql
SELECT id, name, status FROM appdb.customers ORDER BY id;
SHOW GRANTS FOR CURRENT_USER();
EXPLAIN SELECT * FROM appdb.customers WHERE id = 1;
```

Only one logical statement per execution is accepted. A trailing semicolon is
fine; two statements are rejected. The guard allows read starts such as
`SELECT`, `WITH`, `EXPLAIN`, `SHOW`, `DESCRIBE`, `VALUES`, and `TABLE`.

Read-only SQL is rejected with a message such as:

```text
Blocked: write statements are blocked in read-only mode Ctrl+X Ctrl+W to switch.
```

Examples that must not reach the engine while `r/o` is active:

```sql
UPDATE app.customers SET status = 'active' WHERE id = 2;
DELETE FROM app.customers WHERE id = 2;
CREATE ROLE reporting_ro;
GRANT SELECT ON app.customers TO reporting_ro;
```

### Run read-only MQL

MQL is limited to `find`, `aggregate`, `count`, and `distinct`; an optional
`db.<collection>.` prefix is accepted. Select a collection first.

```js
db.customers.find({ status: "active" }).sort({ name: 1 }).limit(10)

db.orders.aggregate([
  { $match: { status: "paid" } },
  { $group: { _id: "$customer_id", total: { $sum: "$total" } } }
])

db.customers.count({ status: "active" })

db.customers.distinct("status")
```

Read-only mode blocks write stages in an aggregate:

```text
Blocked: write stages are blocked in read-only mode Ctrl+X Ctrl+W to switch.
```

Arbitrary JavaScript (`$where`, `$function`, `$accumulator`) is rejected in both
modes. Direct `insert`, `update`, and `delete` commands are not part of the MQL
surface in either mode.

### Read-only checklist

Use `r/o` when you want to:

- explore an unfamiliar database;
- inspect native grants and effective checks;
- run `SELECT`/`EXPLAIN`/`SHOW`/`find`/`aggregate` reports;
- page through data and verify row counts;
- copy statements from history without accidentally running a write.

In `r/o`, the following actions are blocked with the mode-switch hint:

- `+`, `-`, and role `e` on Security roles;
- `Ctrl+X Ctrl+R` access matrix;
- SQL write/admin statements;
- MQL `$out` and `$merge`.

## Read-write mode (`r/w`)

Read-write is explicit and confirmed. It applies only to the current live
session.

### Enable and disable

```text
Ctrl+X Ctrl+W     open the confirmation
Enter             enable r/w (modal says "Writes will run on <profile>")
Esc               cancel
```

The confirmation repeats the warning for elevated/root accounts. After `Enter`
the badge changes to `r/w` and the status line says:

```text
read-write mode on. Read-only guard is off.
```

Switch back at any time with the same chord. Returning to `r/o` is immediate;
no confirmation is required:

```text
read-only mode on. Write statements are blocked.
```

Mode changes are refused while a query result cursor is open. Press `Esc` to
release Results first.

### Run a write statement

`r/w` allows one SQL statement of any kind, subject to the connected account's
native permissions. The tool still rejects multiple statements, malformed
quoting, and arbitrary Mongo JavaScript.

PostgreSQL:

```sql
UPDATE app.customers SET status = 'active' WHERE id = 2;

INSERT INTO app.audit_log (id, actor, action)
VALUES (3, 'db_admin', 'manual-update');

CREATE TABLE app.scratch_notes (
  id bigint PRIMARY KEY,
  body text NOT NULL
);

DROP TABLE app.scratch_notes;
```

MySQL/MariaDB:

```sql
UPDATE appdb.customers SET status = 'active' WHERE id = 2;

INSERT INTO appdb.audit_log (id, actor, action)
VALUES (3, 'mdb_admin', 'manual-update');
```

MongoDB: the query bar still accepts only `find`, `aggregate`, `count`, and
`distinct`. The write surface is an aggregate with `$out` or `$merge`:

```js
db.orders.aggregate([
  { $match: { status: "paid" } },
  { $out: "paid_orders" }
])
```

Direct `insert`, `update`, and `delete` commands are not accepted. The matrix
is the Mongo write surface for roles and privileges.

### Role creation and drop

Roles are managed from the tree:

```text
Security
  roles
    +              create role
    Enter          submit the name
```

```text
Security
  roles
    reporting_ro
    -              open the role menu
    Down Down Enter
                         Drop role
    Enter              confirm
```

Creating a role does not attach privileges or memberships. Use the matrix for
that.

### User creation and drop

Users are managed from the tree:

```text
Security
  users
    +              create user
    Enter          submit the form
```

The form asks for:

- PostgreSQL: a login role name and password;
- MySQL/MariaDB: a user name, host (`%` by default), and password;
- MongoDB: a user name, password, and authentication database. The database
  defaults to `admin`; on that field `Space` cycles through `admin` and the
  databases loaded in the tree, and a custom target can still be typed. It is
  where the user is stored and authenticated, not a database access grant.

The password field is masked. On success the native user is created first and
the password is written to the matching source file:

- PostgreSQL: `~/.pgpass` or the active `PGPASSFILE`;
- MySQL/MariaDB: `~/.mypass` or `MYPASSFILE`;
- MongoDB: `~/.mongo.<USER>-in-<DATABASE>`.

An existing matching password-file line is commented with `#` and the new line
is inserted below it. The file is written atomically with `0600`; the directory
is kept `0700`. Unrelated lines are preserved. The password never appears in a
profile, history entry, log, argv, or error message.

After creation the tool offers:

```text
Save <user>@<database> as a profile? Enter save · Esc skip
```

Saving writes only the optional profile; it does not switch the active session
to the new user. The new user starts with no grants or Mongo roles; use the
matrices to assign them.

Drop a user from `Security → users → user → -` and confirm with `Enter`. The
currently connected user is refused locally. Native dependencies can still
block the drop; the native error is shown unchanged.

### Membership matrix

The `Security`, `users`, `roles`, and principal Details/Privileges views show
the `Ctrl+X Ctrl+R` hint. Open the matrix with `Ctrl+X Ctrl+R`; `1` selects
`Role × Principal`:

```text
role           app_user   app_operator
reporting        [ ]          [x]
app_ro           [x]          [ ]
```

Roles stay on the left, where long names and database suffixes fit; users form
the compact columns. MongoDB users in `admin` show without `@admin`, and MySQL
user columns hide the default `%` host. If a focused column is truncated, the
full name is shown after `Tab switch` in the header.

- `Space` grants or revokes the selected membership.
- `e` opens membership details; `Space`/`Enter` toggles the cell.
- `/` filters principal and role names.
- `Tab`/`Shift+Tab`, `1`/`2` switch matrix panels; the selected principal
  follows across panels and the target tries to keep the same screen position.
- `Esc` closes the matrix, or clears an active filter prompt first.

In this example, `Space` on `reporting × app_operator` writes a native
`GRANT reporting TO app_operator`; toggling it off writes the matching
`REVOKE`.

### Object privilege matrix

`2` selects `Principal/Role × Object`:

```text
principal/role     [db] appdb   [schema] app   customers   orders
reporting              CONNECT       USAGE       SELECT     SELECT
app_user               CONNECT       USAGE       SELECT+2   SELECT
app_writer             —             —           INSERT+2   —
```

- `Space` toggles the default native privilege for that cell.
- `e` opens the full native privilege multi-select.
- `Ctrl+A` selects every listed privilege, or clears them all when everything is
  already selected.
- `Enter` applies the diff between the current cell and the selected list.
- Built-in MongoDB roles are read-only.

Default privileges:

| Engine | Database cell | Schema cell | Table/view/collection cell |
| --- | --- | --- | --- |
| PostgreSQL | `CONNECT` | `USAGE` | `SELECT` |
| MySQL/MariaDB | `SELECT` | — | `SELECT` |
| MongoDB | `find` | — | `find` |

Examples:

```text
2                         switch to privileges
Down / Right              move to app_user × app.customers
Space                     grant or revoke SELECT
e                         open native list: SELECT, INSERT, UPDATE, ...
Space Space               select INSERT and UPDATE
Enter                     apply the diff
```

The matrix writes native statements, not a portable abstraction. PostgreSQL
and MySQL/MariaDB keep direct user grants visible; MongoDB has role rows only
in the privilege panel.

### Matrix scope and keys

The matrix is scoped to the current database/schema and the objects loaded in
the tree. Close it, select another schema or database, and reopen to change
scope.

```text
Tab / Shift+Tab   switch panels
1 / 2             select membership / privileges
Arrows            move the cell cursor
Home / End        jump to the matrix edge
PgUp / PgDn       move a page
Space             toggle the cell
e                 open details
/                 filter rows and columns
Esc               close, or clear the filter prompt
```

Each panel keeps its own cursor, scroll position, and filter while the other
panel is active.

### Native permissions and another user

The current connection is the actor. To edit another user's membership or
grants, the current account must have native admin rights (for example
PostgreSQL role admin option, MySQL `WITH ADMIN OPTION`, or MongoDB
`userAdminAnyDatabase`/`dbAdminAnyDatabase`). The target user's login and
password are never needed.

If the account is not allowed, the native error is shown, for example:

```text
not permitted · this account cannot administer roles · Ctrl+D to reconnect with an admin profile
```

The tool does not impersonate the target user and does not silently switch
accounts. Reconnect with an admin profile using `Ctrl+D`.

### Audit and refresh

Successful queries, role changes, user changes, and row changes are recorded in
the existing history with the mode (`r/o` or `r/w`), native statement, status,
and affected count. Passwords are redacted. After a matrix change the matrix,
Security lists, the current Privileges pane, and loaded access markers are
refreshed; after a row change the Data tab reloads from the first page.

### What `r/w` does not do yet

- Insert more than one row at a time or change several columns in one statement;
  use the query bar.
- Edit `WITH ADMIN OPTION`/`INHERIT` membership attributes.
- Change an existing user's password or edit MongoDB user roles outside the
  matrix.
- Insert MongoDB documents through the query bar.
- Accept multiple SQL statements in one execution.

## Mode troubleshooting

| Message | Meaning | Action |
| --- | --- | --- |
| `write statements are blocked in read-only mode` | SQL write/admin statement in `r/o` | `Ctrl+X Ctrl+W`, confirm, rerun |
| `read-write mode is required; Ctrl+X Ctrl+W to switch.` | role action or matrix in `r/o` | switch to `r/w` |
| `write stages are blocked in read-only mode` | MQL `$out`/`$merge` in `r/o` | switch to `r/w` |
| `results open — Esc to release` | mode switch or new engine action with a live result cursor | `Esc`, then retry |
| `not permitted ... Ctrl+D` | current account lacks native admin rights | reconnect with an admin profile |
| `arbitrary JavaScript is not allowed` | `$where`/`$function`/`$accumulator` | rewrite the MQL without JavaScript |
| `multiple statements are not allowed` | two SQL statements in one execution | execute them one at a time |

For a disposable environment with all three engines, users, grants, and seed
data, see [`live-test/README.md`](../live-test/README.md). The step-by-step
manual TUI checklist is
[`live-test/TUI-TEST-PLAN.md`](../live-test/TUI-TEST-PLAN.md) (Russian).
