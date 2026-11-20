# db-adm-tui v1.10.0

_Release date: 2026-11-20_

_[English](release-notes-v1.10.0.md) | [Русский](release-notes-v1.10.0.ru.md)_

This release adds tree-first matrix shortcuts, faster tree search insertion,
and more direct mode switching.

## Matrix

- User columns no longer carry a `u:` prefix. Roles keep `R:` for built-in
  roles and `r:` for custom roles, so `r:readWrite@accel` is a custom role and
  `R:read@config` is built-in.
- From a database, schema, table, or collection in the tree:
  - `e` or `u` opens `1 Role × Principal`;
  - `r` opens `2 Principal/Role × Object`.
- `Ctrl+X Ctrl+R` remains the global matrix shortcut.
- On MongoDB databases and collections, the membership matrix still ensures
  `read@db` and `readWrite@db` rows exist.
- On the privileges panel, leading `[db]` and `[schema]` columns stay frozen
  while table and collection columns scroll, when at least two scrolling
  object columns fit beside them.
- `Ctrl+Up`/`Ctrl+Down` jump to the start/end of the selected column;
  `Ctrl+Left`/`Ctrl+Right` jump to the start/end of the selected row.

## Tree

- `w` with tree focus opens the same read-write confirmation as
  `Ctrl+X Ctrl+W`; `Enter`, `e`, or `w` accepts.
- Quick search supports fast double `/` to insert the selected node name.
- `Ins` inserts the full selected value, including `schema.table` when
  applicable.
- Insertion works at the current search cursor and does not expand nodes or
  query the server.

## User form

- Labels now end with `:`: `name:`, `password:`, `auth.db:`, `hosts:`, and
  `host:` for MySQL.
- `Up`/`Down` move between fields in addition to `Tab`/`Shift+Tab`.
- Profile editing also accepts `Up`/`Down` for field navigation.

The access matrix remains scoped to the current database/schema and loaded
objects. Views and column-level privileges remain non-editable.
