# db-adm-tui v1.8.1

_Release date: 2026-11-20_

_[English](release-notes-v1.8.1.md) | [Русский](release-notes-v1.8.1.ru.md)_

Patch release for the PostgreSQL privilege matrix.

- Reverted database columns to the **current tree selection** instead of every
  visible database.
- Selecting a database shows its `CONNECT`/`CREATE`/`TEMPORARY` cells; selecting
  a table shows the schema/table scope.
- The selected scope is highlighted as `scope <database> · <object>` in the
  matrix footer.
- Views and column-level privileges remain non-editable.

No profile schema or data-format changes.
