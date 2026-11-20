# db-adm-tui v1.6.0

_Release date: 2026-11-20_

_[English](release-notes-v1.6.0.md) | [Русский](release-notes-v1.6.0.ru.md)_

This release adds safe profile editing and deletion, and makes matrix toggles
visible immediately.

## Profile editing

- Editing a connection field updates the credential source with the existing
  password instead of leaving a stale entry.
- PostgreSQL/MySQL/MariaDB: the old `.pgpass`/`.mypass` line is commented and
  the new tuple is inserted below it; unrelated lines are preserved.
- MongoDB: the URI is written to its canonical
  `.mongo.<USER>-in-<DATABASE>` file with `authSource` rebound to the selected
  database; the old source file is kept and its old URI is commented.
- Label-only or TLS-only edits do not rewrite the source file.

## Profile deletion

- `Del` or `-` at the picker opens a confirmation for the selected profile.
- Only the profile entry is removed. The `.pgpass`, `.mypass`, or `.mongo.*`
  source file is never modified or deleted by profile deletion.

## Matrix feedback

- Membership and privilege cells update optimistically when `Space` is pressed.
- The matrix keeps the loaded cells visible during the background refresh; a
  failed mutation reloads the previous native state.

No profile schema or data-format changes.
