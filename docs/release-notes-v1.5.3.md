# db-adm-tui v1.5.3

_Release date: 2026-11-20_

_[English](release-notes-v1.5.3.md) | [Русский](release-notes-v1.5.3.ru.md)_

Patch release for the MongoDB user form and the statement editor.

## MongoDB user form

- The authentication database defaults to `admin`, which covers the common
  case.
- On the database field, `Space` cycles through `admin` and the databases
  loaded in the tree.
- A custom database can still be typed for unusual cases; the field remains
  editable.

## Statement editor

- `Tab` opens completion in the `Ctrl+X Ctrl+E` editor.
- A second `Tab` accepts the selected completion item.
- Query-bar behavior is unchanged: `Tab` still moves focus there.

No profile, state, or data-format changes.
