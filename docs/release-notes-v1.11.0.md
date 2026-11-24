# db-adm-tui v1.11.0

_Release date: 2026-11-24_

_[English](release-notes-v1.11.0.md) | [Русский](release-notes-v1.11.0.ru.md)_

This release makes profile creation look and behave like profile editing, and
adds TLS control to creation and import.

## New connection form

- `n` now opens a field-based form matching the edit window: engine, label,
  host, port, database, user, MongoDB `auth.db`, and the PostgreSQL `no TLS`
  toggle.
- `Space` on `engine` cycles PostgreSQL, MySQL/MariaDB, and MongoDB. The port
  follows the engine default unless it was edited.
- `Tab`/`Shift+Tab` and `Up`/`Down` move between fields.
- A matching `.pgpass`, `.mypass`, or `.mongo.*` source is still found
  automatically; otherwise the form reveals the masked one-time password field.
- PostgreSQL profiles can enable `allow_no_tls` directly during creation.

## Import

- The import window adds `t` → `no TLS [ ]`/`[x]`, applied to selected
  PostgreSQL entries before import.
- Imported MySQL/MariaDB and MongoDB entries are unaffected.

## Navigation

- `Up`/`Down` field navigation is covered for the new-connection, profile-edit,
  and user forms.
- `Space` cycles the engine selector and toggles the TLS fallback.

The saved profile schema is unchanged.
