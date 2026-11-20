# db-adm-tui v1.6.1

_Release date: 2026-11-20_

_[English](release-notes-v1.6.1.md) | [Русский](release-notes-v1.6.1.ru.md)_

Patch release for matrix panel switching.

- `Tab`/`Shift+Tab` and `1`/`2` now keep the selected principal when switching
  between `Role × Principal` and `Principal/Role × Object`.
- The membership column user/role becomes the matching privileges row; switching
  back returns to that principal's column.
- The target panel tries to keep the principal at the same screen position.
- If a MongoDB user has no privileges row (users are not editable there), the
  selected role row is used instead.

No profile, state, or data-format changes.
