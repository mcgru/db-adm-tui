# db-adm-tui v1.9.0

_Release date: 2026-11-20_

_[English](release-notes-v1.9.0.md) | [Русский](release-notes-v1.9.0.ru.md)_

This release adds the MongoDB home/auth database model and faster tree actions
for users and roles.

## MongoDB databases

- Profiles now distinguish `home.db` (URI/default database) from `auth.db`
  (`authSource`).
- The profile editor shows `home.db` first and a muted `auth.db` below it.
- MongoDB connection URIs use `home.db` as the path and `auth.db` as
  `authSource`.
- Credential files are still keyed by the authentication database, so changing
  `home.db` does not move the source file.

## User actions

- `+`, `n`/`т`, `u`/`г`, and `Ctrl+N` create a user from `Security` or `users`.
- On a MongoDB user, `h` opens a `home.db` picker; `Space` cycles the loaded
  databases and `Enter` updates the matching saved profile.
- The create form uses `auth.db` as its label and keeps the comma-separated
  `hosts` field.

## Role assignment

- On a MongoDB database or collection, `u` or `Ctrl+X Ctrl+R` opens
  `Role × Principal` and ensures built-in `read@db` and `readWrite@db` rows are
  available for assignment.
- Views and column-level privileges remain non-editable.

No destructive source-file changes are made by `h`; only the profile and its
credential URI are updated.
