# db-adm-tui v1.5.0

_Release date: 2026-11-20_

_[English](release-notes-v1.5.0.md) | [Русский](release-notes-v1.5.0.ru.md)_

This release makes the access matrix readable when role names carry a database
suffix.

## Matrix orientation

- `1 Role × Principal` now puts roles on the left and users/principals in the
  columns for PostgreSQL, MySQL/MariaDB, and MongoDB.
- Long names such as `readWriteAnyDatabase@admin` stay in the left label
  column instead of being squeezed into a cell header.
- `2 Principal/Role × Object` keeps principals/roles on the left and objects in
  the columns: that orientation already gives the long names the wide side.
  The truncated-column hint below also applies there.

## Compact labels

- MongoDB users in `admin` are shown without `@admin`; users in another
  database keep `@database`.
- MySQL user columns hide the default `%` host; non-default hosts stay visible.
- If the focused column label does not fit its cell, the full name is shown
  after `Tab switch` in the matrix header.

## Privilege details

- `Ctrl+A` in the privilege details multi-select selects every listed
  privilege, or clears them all when everything is already selected.

## Discoverability

`Security`, `users`, `roles`, and individual principal Details/Privileges views
now show `Ctrl+X Ctrl+R opens the access matrix (r/w)`.

## Engine notes

- PostgreSQL: role and user names are already short and database-less; the
  transpose mainly gives long role names more room.
- MySQL/MariaDB: `user@host` columns hide `@%`, while roles remain on the left;
  `user@host` is kept for non-default hosts.
- MongoDB: `role@db` rows are on the left, `user`/`user@db` columns on top, and
  built-in roles remain visible.

No profile, state, or data-format changes.
