#!/usr/bin/env bash
set -euo pipefail

: "${MARIADB_ROOT_PASSWORD:?MARIADB_ROOT_PASSWORD is required}"
: "${MARIADB_ADMIN_USER:=mdb_admin}"
: "${MARIADB_ADMIN_PASSWORD:?MARIADB_ADMIN_PASSWORD is required}"
: "${MARIADB_APP_USER:=app_user}"
: "${MARIADB_APP_PASSWORD:?MARIADB_APP_PASSWORD is required}"
: "${MARIADB_ANALYST_USER:=analyst}"
: "${MARIADB_ANALYST_PASSWORD:?MARIADB_ANALYST_PASSWORD is required}"

export MYSQL_PWD="$MARIADB_ROOT_PASSWORD"

mariadb --protocol=socket -uroot <<SQL
CREATE DATABASE IF NOT EXISTS appdb CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS analytics CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS sandbox CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS appdb.customers (
  id bigint PRIMARY KEY,
  name varchar(120) NOT NULL,
  status varchar(24) NOT NULL,
  balance decimal(12,2) NOT NULL DEFAULT 0,
  created_at timestamp NOT NULL DEFAULT current_timestamp
) ENGINE=InnoDB;
CREATE TABLE IF NOT EXISTS appdb.orders (
  id bigint PRIMARY KEY,
  customer_id bigint NOT NULL,
  total decimal(12,2) NOT NULL,
  status varchar(24) NOT NULL,
  created_at timestamp NOT NULL DEFAULT current_timestamp,
  CONSTRAINT orders_customer_fk FOREIGN KEY (customer_id) REFERENCES appdb.customers(id)
) ENGINE=InnoDB;
CREATE TABLE IF NOT EXISTS appdb.audit_log (
  id bigint PRIMARY KEY,
  actor varchar(120) NOT NULL,
  action varchar(120) NOT NULL,
  created_at timestamp NOT NULL DEFAULT current_timestamp
) ENGINE=InnoDB;
CREATE TABLE IF NOT EXISTS analytics.events (
  id bigint PRIMARY KEY,
  event_type varchar(64) NOT NULL,
  actor varchar(120) NOT NULL,
  payload json NOT NULL,
  created_at timestamp NOT NULL DEFAULT current_timestamp
) ENGINE=InnoDB;
CREATE TABLE IF NOT EXISTS sandbox.notes (
  id bigint PRIMARY KEY,
  body varchar(255) NOT NULL
) ENGINE=InnoDB;

INSERT IGNORE INTO appdb.customers (id, name, status, balance) VALUES
  (1, 'Alice', 'active', 120.50),
  (2, 'Bob', 'paused', 40.00),
  (3, 'Carol', 'active', 300.75);
INSERT IGNORE INTO appdb.orders (id, customer_id, total, status) VALUES
  (1, 1, 42.00, 'paid'),
  (2, 1, 18.50, 'open'),
  (3, 2, 7.25, 'paid');
INSERT IGNORE INTO appdb.audit_log (id, actor, action) VALUES
  (1, 'system', 'bootstrap'),
  (2, 'mdb_admin', 'seed');
INSERT IGNORE INTO analytics.events (id, event_type, actor, payload) VALUES
  (1, 'login', 'alice', '{"source":"web"}'),
  (2, 'purchase', 'bob', '{"order_id":3}'),
  (3, 'logout', 'alice', '{}');
INSERT IGNORE INTO sandbox.notes (id, body) VALUES
  (1, 'scratch database for live tests'),
  (2, 'safe to reset');

CREATE ROLE IF NOT EXISTS app_ro;
CREATE ROLE IF NOT EXISTS app_writer;
CREATE ROLE IF NOT EXISTS reporting;

CREATE USER IF NOT EXISTS '${MARIADB_ADMIN_USER}'@'%' IDENTIFIED BY '${MARIADB_ADMIN_PASSWORD}';
ALTER USER '${MARIADB_ADMIN_USER}'@'%' IDENTIFIED BY '${MARIADB_ADMIN_PASSWORD}';
CREATE USER IF NOT EXISTS '${MARIADB_APP_USER}'@'%' IDENTIFIED BY '${MARIADB_APP_PASSWORD}';
ALTER USER '${MARIADB_APP_USER}'@'%' IDENTIFIED BY '${MARIADB_APP_PASSWORD}';
CREATE USER IF NOT EXISTS '${MARIADB_ANALYST_USER}'@'%' IDENTIFIED BY '${MARIADB_ANALYST_PASSWORD}';
ALTER USER '${MARIADB_ANALYST_USER}'@'%' IDENTIFIED BY '${MARIADB_ANALYST_PASSWORD}';

GRANT ALL PRIVILEGES ON *.* TO '${MARIADB_ADMIN_USER}'@'%' WITH GRANT OPTION;
GRANT SELECT ON appdb.* TO app_ro;
GRANT SELECT ON analytics.* TO app_ro;
GRANT SELECT, INSERT, UPDATE, DELETE ON appdb.* TO app_writer;
GRANT SELECT ON appdb.* TO reporting;
GRANT SELECT ON analytics.* TO reporting;
GRANT SELECT ON sandbox.* TO reporting;

GRANT SELECT, INSERT, UPDATE ON appdb.customers TO '${MARIADB_APP_USER}'@'%';
GRANT SELECT ON appdb.orders TO '${MARIADB_APP_USER}'@'%';
GRANT SELECT ON analytics.events TO '${MARIADB_ANALYST_USER}'@'%';

GRANT app_ro TO '${MARIADB_APP_USER}'@'%';
GRANT app_writer TO '${MARIADB_APP_USER}'@'%';
GRANT reporting TO '${MARIADB_ANALYST_USER}'@'%';

GRANT app_ro TO '${MARIADB_ADMIN_USER}'@'%' WITH ADMIN OPTION;
GRANT app_writer TO '${MARIADB_ADMIN_USER}'@'%' WITH ADMIN OPTION;
GRANT reporting TO '${MARIADB_ADMIN_USER}'@'%' WITH ADMIN OPTION;

SET DEFAULT ROLE app_ro FOR '${MARIADB_APP_USER}'@'%';
SET DEFAULT ROLE reporting FOR '${MARIADB_ANALYST_USER}'@'%';
FLUSH PRIVILEGES;
SQL

echo "mariadb: users, roles, databases, grants, and seed data are ready"
