#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

for service in postgres mariadb mongo; do
  echo "== $service =="
  env_file=".env"
  [[ -f "$ROOT/$service/.env" ]] || env_file=".env.example"
  (cd "$ROOT/$service" && docker compose --env-file "$env_file" -f docker-compose.yml ps)
  echo
done

cat <<'EOF'
Endpoints (defaults):
  PostgreSQL  postgres://db_admin:...@127.0.0.1:5432/appdb?sslmode=disable
  MariaDB     mysql://mdb_admin:...@127.0.0.1:3306/appdb
  MongoDB     mongodb://mongo_admin:...@127.0.0.1:57017/admin?authSource=admin&replicaSet=rs0

Credentials and ports live in each service's .env.
EOF
