# db-adm-tui v1.4.2

_Release date: 2026-11-20_

_[English](release-notes-v1.4.2.md) | [Русский](release-notes-v1.4.2.ru.md)_

Patch release for MongoDB user creation.

- The user form now has an editable **database** field for MongoDB.
- It defaults to the current database and can be changed to `admin` or any
  other database before creating the user.
- The credential file and optional profile use the selected database
  (`.mongo.<USER>-in-<DATABASE>`, `<USER>@<DATABASE>`).
- The form reminds that roles are assigned after creation through
  `Ctrl+X Ctrl+R` → `1 Principal × Role`.

PostgreSQL and MySQL/MariaDB forms are unchanged.
