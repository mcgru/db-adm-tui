# db-adm-tui v1.8.0

_Release date: 2026-11-20_

_[English](release-notes-v1.8.0.md) | [Русский](release-notes-v1.8.0.ru.md)_

This release streamlines matrix details and widens PostgreSQL database access
editing.

## Matrix details

- `e` and `Enter` both open the details modal.
- `e` or `Esc` closes it without applying.
- `Enter` still applies privilege changes or toggles a membership.
- Footer and action hints show the new keys.

## PostgreSQL privileges

- The database columns now include every database visible in the tree, not only
  the current database. `CONNECT`/`CREATE`/`TEMPORARY` can be edited for any of
  them from one session.
- Schema and table columns remain scoped to the current database/schema.
- Views and column-level privileges are no longer editable; only database,
  schema, and table targets are offered (collection targets for MongoDB).

No profile schema or data-format changes.
