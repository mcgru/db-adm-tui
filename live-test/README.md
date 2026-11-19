# Live database fixtures

Throwaway PostgreSQL 18, MariaDB 11.4, and MongoDB 8.2 instances for exercising
`db-adm-tui` against real servers. Everything is local to this directory and
uses bind mounts, not Docker named volumes.

## Live server

The maintained test host is:

```text
ssh jojo@uuu
~/live-test
```

`jojo` can run Docker and Docker Compose on that host. The directory is a
disposable test environment, not production.

Update the server copy from this repository:

```bash
rsync -a --delete \
  --exclude='.env' \
  --exclude='.data/' \
  live-test/ jojo@uuu:~/live-test/
```

Then start or inspect it:

```bash
ssh jojo@uuu 'cd ~/live-test && ./up.sh'
ssh jojo@uuu 'cd ~/live-test && ./status.sh'
```

For local UI testing, forward the three ports:

```bash
ssh -N \
  -L 5432:127.0.0.1:5432 \
  -L 3306:127.0.0.1:3306 \
  -L 57017:127.0.0.1:57017 \
  jojo@uuu
```

The default Mongo replica-set member is advertised as `localhost:57017`, so
the tunnel above matches the fixture's `.env.example`.

## TUI test plan

The step-by-step manual test plan, connection profiles, expected messages, and
cleanup steps are in [`TUI-TEST-PLAN.md`](TUI-TEST-PLAN.md) (Russian).

## Quick start

Requirements: Docker Engine 24+, Docker Compose v2, and `openssl` (or a
`base64` fallback).

```bash
cd live-test
./up.sh
```

On first run `up.sh` copies each `*.env.example` to `*.env`, creates the data
directories, generates the MongoDB replica-set keyfile, starts the containers,
waits for health checks, and reruns the idempotent fixture scripts. Edit the
`.env` files first if the default test passwords or ports are not suitable.

```bash
./status.sh       # containers and connection hints
./down.sh         # stop containers; data directories stay in place
./reset.sh        # stop, delete all data, start fresh (asks for confirmation)
./reset.sh --yes  # same without the prompt
```

Each service also has its own rerunnable fixture:

```bash
./postgres/apply.sh
./mariadb/apply.sh
./mongo/apply.sh
```

## Layout

```text
live-test/
  up.sh / down.sh / status.sh / reset.sh
  postgres/
    docker-compose.yml
    .env.example
    apply.sh
    init/01-init.sh
  mariadb/
    docker-compose.yml
    .env.example
    apply.sh
    init/01-init.sh
  mongo/
    docker-compose.yml
    .env.example
    apply.sh
    init/01-init.js
```

Data directories are created next to the compose files:

```text
postgres/.data/18/docker   # PostgreSQL 18 cluster (PGDATA)
mariadb/.data             # /var/lib/mysql
mongo/.data/db            # /data/db
mongo/.data/keyfile       # replica-set keyfile, 0400, UID 999
```

To move a test iteration, run `./down.sh`, move these directories (and keep the
compose/env files), then run `./up.sh`. Do not regenerate the MongoDB keyfile
for an existing data directory.

## Default access

All passwords are test-only and live in each `.env.example`.

### PostgreSQL (`127.0.0.1:5432`)

| Principal | Kind | Purpose |
| --- | --- | --- |
| `postgres` | superuser | root connection |
| `db_admin` | LOGIN, CREATEDB, CREATEROLE | matrix admin, object owner |
| `app_user` | LOGIN | ordinary user with direct grants |
| `analyst` | LOGIN | ordinary user |
| `app_ro` | NOLOGIN role | read-only role |
| `reporting` | NOLOGIN role | reporting role |

Databases: `appdb`, `analytics`, `sandbox`. `appdb` has `app.customers`,
`app.orders`, `app.audit_log`; `analytics.events` and `sandbox.notes` are also
seeded. Membership and direct/table/schema/database grants are pre-created;
`db_admin` holds `WITH ADMIN OPTION` on the managed roles.

Example:

```text
postgres://db_admin:db_admin_test_password@127.0.0.1:5432/appdb?sslmode=disable
```

### MariaDB (`127.0.0.1:3306`)

| Principal | Kind | Purpose |
| --- | --- | --- |
| `root` | root | server root |
| `mdb_admin` | admin | matrix admin with grant options |
| `app_user` | ordinary user | direct table grants + role memberships |
| `analyst` | ordinary user | reporting role |
| `app_ro`, `app_writer`, `reporting` | roles | read, write, reporting |

Databases: `appdb`, `analytics`, `sandbox` with the same seed names as
PostgreSQL. Use an account form such as `mdb_admin@%` in the tool.

Example:

```text
mysql://mdb_admin:mdb_admin_test_password@127.0.0.1:3306/appdb
```

### MongoDB (`127.0.0.1:57017`, replica set `rs0`)

| Principal | Auth database | Purpose |
| --- | --- | --- |
| `root` | `admin` | root |
| `mongo_admin` | `admin` | user/role/database admin (not root) |
| `app_user` | `appdb` | ordinary user |
| `analyst` | `analytics` | ordinary user |
| `app_writer` | role in `appdb` | collection write privileges |
| `analytics_read` | role in `analytics` | database read privileges |
| `sandbox_reader` | role in `sandbox` | database read privileges |

Collections: `appdb.customers`, `appdb.orders`, `appdb.audit_log`,
`analytics.events`, `sandbox.notes`.

Example:

```text
mongodb://mongo_admin:mongo_admin_test_password@127.0.0.1:57017/admin?authSource=admin&replicaSet=rs0
```

## SSH tunnel

For the default ports:

```bash
ssh -N \
  -L 5432:127.0.0.1:5432 \
  -L 3306:127.0.0.1:3306 \
  -L 57017:127.0.0.1:57017 \
  user@your-server
```

MongoDB advertises `MONGO_REPLICA_HOST` from `mongo/.env`. It defaults to
`localhost:57017`, which works with the tunnel above. For direct access, change
it to a hostname/IP that the client can resolve and restart Mongo; for an
existing replica set, update the member host with `rs.reconfig` instead of
changing only the env file.

## Notes

- The stack is intentionally insecure: known passwords, published ports, and
  a single-node MongoDB replica set. Do not expose it to the internet.
- PostgreSQL 18 mounts `/var/lib/postgresql` because its default PGDATA is
  version-specific (`/var/lib/postgresql/18/docker`).
- Fixture scripts are idempotent: users/roles are updated, databases and
  collections/tables are created if absent, and seed rows are inserted only
  when missing.
