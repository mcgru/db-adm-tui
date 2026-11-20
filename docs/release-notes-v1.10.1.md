# db-adm-tui v1.10.1

_Release date: 2026-11-20_

_[English](release-notes-v1.10.1.md) | [Русский](release-notes-v1.10.1.ru.md)_

This release extends the tree-first matrix shortcuts to the security groups.

## Matrix shortcuts

- With `Security`, `users`, or `roles` selected in the tree:
  - `e` opens `1 Role × Principal`;
  - `r` opens `2 Principal/Role × Object`.
- Database, schema, table, and collection shortcuts are unchanged.
- `Ctrl+X Ctrl+R` remains the global shortcut.

The matrix keeps its current scope, frozen `[db]`/`[schema]` columns, and
read-write requirement.
