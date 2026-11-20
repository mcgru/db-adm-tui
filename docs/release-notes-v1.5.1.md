# db-adm-tui v1.5.1

_Release date: 2026-11-20_

_[English](release-notes-v1.5.1.md) | [Русский](release-notes-v1.5.1.ru.md)_

Patch release for the MongoDB user form.

- Field order is now **name → password → database**, matching the visual order.
- The database field defaults to the current database and is still editable to
  `admin` or another database.
- The form now explains that this is the **authentication database** where the
  MongoDB user is stored, not a database access grant. Roles are assigned
  separately through `Ctrl+X Ctrl+R` → `1 Role × Principal`.

No profile, state, or data-format changes.
