# db-adm-tui v1.4.1

_Release date: 2026-11-20_

_[English](release-notes-v1.4.1.md) | [Русский](release-notes-v1.4.1.ru.md)_

Patch release for the `v1.4.0` cell editor.

- The field name, mode tag (`[val]`/`[DEF]`/`[NULL]`), and prefilled cell
  value are now compact on one line.
- A bar separates the mode from the value, so the start of the inserted value
  is obvious.
- The insertion point uses a solid block cursor that blinks while the editor is
  open and stays visible after every key press.

No profile, state, or data-format changes. The `v1.4.0` release archives remain
valid; use `v1.4.1` for the editor fix.
