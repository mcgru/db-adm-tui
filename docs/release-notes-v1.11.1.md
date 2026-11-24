# db-adm-tui v1.11.1

_Release date: 2026-11-24_

_[English](release-notes-v1.11.1.md) | [Русский](release-notes-v1.11.1.ru.md)_

This patch removes a height jump in the profile editor and makes the TLS
checkbox explicit.

## Profile editor

- The PostgreSQL TLS hint line is now reserved as soon as the edit window
  opens.
- Moving focus onto the TLS toggle no longer changes the modal height or shifts
  the window vertically.
- The hint is visible immediately, so the TLS behavior does not appear only
  after reaching the field.

## TLS checkbox

- The PostgreSQL control is now labeled `TLS` and is checked by default.
- Uncheck it with `Space` to allow the plaintext fallback after a failed TLS
  attempt.
- The new-connection form and the import toggle use the same `TLS [x]`/`[ ]`
  convention. In import, `t` toggles it for selected PostgreSQL entries.

No profile schema or connection behavior changed.

