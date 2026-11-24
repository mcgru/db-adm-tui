# db-adm-tui v1.12.0

_Release date: 2026-11-24_

_[English](release-notes-v1.12.0.md) | [Русский](release-notes-v1.12.0.ru.md)_

This release fills conventional account defaults in the new-profile form.

## Engine defaults

- PostgreSQL: an empty user becomes `postgres`, and an empty database becomes
  `postgres`. The `postgres` database and role are the conventional bootstrap
  names; `initdb` allows the role name to be changed, but distributions use
  `postgres`.
- MySQL/MariaDB: an empty user becomes `root`, the initial administrative
  account created during installation.
- MongoDB: there is no default login user; the field stays required. An empty
  `auth.db` becomes `admin`.

## When defaults apply

- Cycling `engine` fills an empty or still-automatic user/database/auth field
  without overwriting a value the user typed.
- Pressing `Enter` with an empty defaultable field connects using the default
  without first writing it into the form.
- After a successful connection, the profile used for the save prompt contains
  the resolved defaults. A failed connection leaves the form unchanged.

The saved profile schema is unchanged.
