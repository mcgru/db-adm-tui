#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

for service in postgres mariadb mongo; do
  env_file=".env"
  [[ -f "$ROOT/$service/.env" ]] || env_file=".env.example"
  (cd "$ROOT/$service" && docker compose --env-file "$env_file" -f docker-compose.yml down --remove-orphans)
done
