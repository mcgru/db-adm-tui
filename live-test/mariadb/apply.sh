#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"
if [[ ! -f .env ]]; then
  echo "mariadb/.env is missing; run ../up.sh or copy .env.example to .env" >&2
  exit 1
fi

docker compose --env-file .env -f docker-compose.yml exec -T mariadb \
  bash /docker-entrypoint-initdb.d/01-init.sh
