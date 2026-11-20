# db-adm-tui v1.7.0

_Release date: 2026-11-20_

_[English](release-notes-v1.7.0.md) | [Русский](release-notes-v1.7.0.ru.md)_

This release fixes MongoDB replica-set host preservation and adds a host-list
field to MongoDB user creation.

## MongoDB profiles

- Editing a MongoDB profile no longer collapses a replica-set URI to its first
  host when changing the database, user, or label.
- The full `host1:port1,host2:port2` list and non-secret URI options are read
  from the source selector and preserved.
- An intentional host change still writes the new host target.

## MongoDB user creation

- The user form now has a `hosts` field after the authentication database.
- It defaults to the current profile's full host list.
- The value is a comma-separated `host:port` list; a bare host uses port 27017.
- The generated `.mongo.<USER>-in-<DATABASE>` credential URI keeps the full
  host list and the profile's `authSource`/`replicaSet` options.

No profile schema or data-format changes.
