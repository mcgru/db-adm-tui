# Using db-adm-tui in r/o and r/w

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
| Create/drop user | not implemented | not implemented |
| Row/document editing | not implemented | not implemented |

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
DESCRIBE app.customers;
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

### Membership matrix

Open the matrix with `Ctrl+X Ctrl+R`; `1` selects `Principal × Role`:

```text
principal      reporting   app_ro   app_role
app_user          [ ]        [x]      [ ]
app_operator      [x]        [ ]      [ ]
```

- `Space` grants or revokes the selected membership.
- `e` opens membership details; `Space`/`Enter` toggles the cell.
- `/` filters principal and role names.
- `Tab`/`Shift+Tab`, `1`/`2` switch matrix panels.
- `Esc` closes the matrix, or clears an active filter prompt first.

In this example, `Space` on `app_operator × reporting` writes a native
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

Successful queries and role changes are recorded in the existing history with
the mode (`r/o` or `r/w`), native statement, status, and affected count.
Passwords are redacted. After a matrix change the matrix, Security lists, the
current Privileges pane, and loaded access markers are refreshed.

### What `r/w` does not do yet

- Create/drop users and write credential files (Story 9c).
- Insert/update/delete table rows or MongoDB documents (Story 9a).
- Edit `WITH ADMIN OPTION`/`INHERIT` membership attributes.
- Edit MongoDB users or insert documents through the query bar.
- Accept multiple SQL statements in one execution.

`+`/`-` on Data or a non-role tree node still reports that row changes are
Story 9a.

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
data, see [`live-test/README.md`](../live-test/README.md).
