#!/usr/bin/env bash
set -euo pipefail

: "${POSTGRES_USER:=postgres}"
: "${PG_ADMIN_USER:=db_admin}"
: "${PG_ADMIN_PASSWORD:?PG_ADMIN_PASSWORD is required}"
: "${PG_APP_USER:=app_user}"
: "${PG_APP_PASSWORD:?PG_APP_PASSWORD is required}"
: "${PG_ANALYST_USER:=analyst}"
: "${PG_ANALYST_PASSWORD:?PG_ANALYST_PASSWORD is required}"

psql_admin=(psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname postgres)
psql_vars=(
  -v admin_user="$PG_ADMIN_USER"
  -v admin_pw="$PG_ADMIN_PASSWORD"
  -v app_user="$PG_APP_USER"
  -v app_pw="$PG_APP_PASSWORD"
  -v analyst_user="$PG_ANALYST_USER"
  -v analyst_pw="$PG_ANALYST_PASSWORD"
)

"${psql_admin[@]}" "${psql_vars[@]}" <<'SQL'
SELECT format('CREATE ROLE %I LOGIN CREATEDB CREATEROLE PASSWORD %L', :'admin_user', :'admin_pw')
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = :'admin_user') \gexec
ALTER ROLE :"admin_user" WITH LOGIN CREATEDB CREATEROLE PASSWORD :'admin_pw';

SELECT format('CREATE ROLE %I LOGIN PASSWORD %L', :'app_user', :'app_pw')
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = :'app_user') \gexec
ALTER ROLE :"app_user" WITH LOGIN PASSWORD :'app_pw';

SELECT format('CREATE ROLE %I LOGIN PASSWORD %L', :'analyst_user', :'analyst_pw')
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = :'analyst_user') \gexec
ALTER ROLE :"analyst_user" WITH LOGIN PASSWORD :'analyst_pw';

SELECT format('CREATE ROLE %I NOLOGIN', 'app_ro')
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_ro') \gexec
SELECT format('CREATE ROLE %I NOLOGIN', 'reporting')
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'reporting') \gexec

SELECT format('CREATE DATABASE %I OWNER %I', db, :'admin_user')
FROM (VALUES ('appdb'), ('analytics'), ('sandbox')) AS v(db)
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = v.db) \gexec

ALTER DATABASE appdb OWNER TO :"admin_user";
ALTER DATABASE analytics OWNER TO :"admin_user";
ALTER DATABASE sandbox OWNER TO :"admin_user";
SQL

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname appdb "${psql_vars[@]}" <<'SQL'
SET ROLE :"admin_user";
CREATE SCHEMA IF NOT EXISTS app AUTHORIZATION :"admin_user";

CREATE TABLE IF NOT EXISTS app.customers (
  id bigint PRIMARY KEY,
  name text NOT NULL,
  status text NOT NULL,
  balance numeric(12,2) NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS app.orders (
  id bigint PRIMARY KEY,
  customer_id bigint NOT NULL REFERENCES app.customers(id),
  total numeric(12,2) NOT NULL,
  status text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS app.audit_log (
  id bigint PRIMARY KEY,
  actor text NOT NULL,
  action text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

INSERT INTO app.customers (id, name, status, balance) VALUES
  (1, 'Alice', 'active', 120.50),
  (2, 'Bob', 'paused', 40.00),
  (3, 'Carol', 'active', 300.75)
ON CONFLICT (id) DO NOTHING;
INSERT INTO app.orders (id, customer_id, total, status) VALUES
  (1, 1, 42.00, 'paid'),
  (2, 1, 18.50, 'open'),
  (3, 2, 7.25, 'paid')
ON CONFLICT (id) DO NOTHING;
INSERT INTO app.audit_log (id, actor, action) VALUES
  (1, 'system', 'bootstrap'),
  (2, 'pg_admin', 'seed')
ON CONFLICT (id) DO NOTHING;
RESET ROLE;

REVOKE ALL ON DATABASE appdb FROM PUBLIC;
GRANT CONNECT ON DATABASE appdb TO :"admin_user", app_ro, reporting, :"app_user", :"analyst_user";
GRANT USAGE ON SCHEMA app TO app_ro, reporting, :"app_user", :"analyst_user";
GRANT SELECT ON app.customers, app.orders, app.audit_log TO app_ro;
GRANT SELECT ON ALL TABLES IN SCHEMA app TO reporting;
GRANT SELECT, INSERT, UPDATE ON app.customers TO :"app_user";
GRANT SELECT ON app.orders TO :"app_user";

GRANT reporting TO :"app_user";
GRANT app_ro TO :"app_user";
GRANT reporting TO :"admin_user" WITH ADMIN OPTION;
GRANT app_ro TO :"admin_user" WITH ADMIN OPTION;
GRANT :"app_user" TO :"admin_user" WITH ADMIN OPTION;
SQL

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname analytics "${psql_vars[@]}" <<'SQL'
SET ROLE :"admin_user";
CREATE SCHEMA IF NOT EXISTS analytics AUTHORIZATION :"admin_user";
CREATE TABLE IF NOT EXISTS analytics.events (
  id bigint PRIMARY KEY,
  event_type text NOT NULL,
  actor text NOT NULL,
  payload jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO analytics.events (id, event_type, actor, payload) VALUES
  (1, 'login', 'alice', '{"source":"web"}'::jsonb),
  (2, 'purchase', 'bob', '{"order_id":3}'::jsonb),
  (3, 'logout', 'alice', '{}'::jsonb)
ON CONFLICT (id) DO NOTHING;
RESET ROLE;

REVOKE ALL ON DATABASE analytics FROM PUBLIC;
GRANT CONNECT ON DATABASE analytics TO :"admin_user", app_ro, reporting, :"app_user", :"analyst_user";
GRANT USAGE ON SCHEMA analytics TO app_ro, reporting, :"app_user", :"analyst_user";
GRANT SELECT ON analytics.events TO app_ro, reporting, :"analyst_user";
GRANT SELECT ON analytics.events TO :"app_user";
GRANT reporting TO :"analyst_user";
SQL

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname sandbox "${psql_vars[@]}" <<'SQL'
SET ROLE :"admin_user";
CREATE SCHEMA IF NOT EXISTS sandbox AUTHORIZATION :"admin_user";
CREATE TABLE IF NOT EXISTS sandbox.notes (
  id bigint PRIMARY KEY,
  body text NOT NULL
);
INSERT INTO sandbox.notes (id, body) VALUES
  (1, 'scratch database for live tests'),
  (2, 'safe to reset')
ON CONFLICT (id) DO NOTHING;
RESET ROLE;

REVOKE ALL ON DATABASE sandbox FROM PUBLIC;
GRANT CONNECT ON DATABASE sandbox TO :"admin_user", app_ro, reporting, :"app_user", :"analyst_user";
GRANT USAGE ON SCHEMA sandbox TO reporting;
GRANT SELECT ON sandbox.notes TO reporting;
SQL

echo "postgres: users, databases, grants, and seed data are ready"
