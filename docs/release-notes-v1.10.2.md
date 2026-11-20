# db-adm-tui v1.10.2

_Release date: 2026-11-20_

_[English](release-notes-v1.10.2.md) | [Русский](release-notes-v1.10.2.ru.md)_

This release focuses the matrix on the principal selected in the tree.

## Principal focus

- With a specific user selected in the tree, `e` opens `1 Role × Principal`
  with the cursor on that user's column.
- With a specific user selected, `r` opens `2 Principal/Role × Object` with the
  cursor on that user's row.
- A role opened from the tree keeps its privilege row focus.
- Group shortcuts on `Security`, `users`, and `roles` remain unchanged.

The cursor is scrolled into view, including the frozen `[db]`/`[schema]`
columns on the privileges panel.
