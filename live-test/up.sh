#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

ensure_env() {
  local service=$1
  if [[ ! -f "$ROOT/$service/.env" ]]; then
    cp "$ROOT/$service/.env.example" "$ROOT/$service/.env"
    echo "created $service/.env from .env.example"
  fi
}

compose() {
  local service=$1
  shift
  (cd "$ROOT/$service" && docker compose --env-file .env -f docker-compose.yml "$@")
}

prepare_mongo_keyfile() {
  local keyfile="$ROOT/mongo/.data/keyfile"
  mkdir -p "$ROOT/mongo/.data/db" "$ROOT/postgres/.data" "$ROOT/mariadb/.data"

  if [[ ! -s "$keyfile" ]]; then
    if command -v openssl >/dev/null 2>&1; then
      openssl rand -base64 756 > "$keyfile"
    else
      dd if=/dev/urandom bs=1 count=756 2>/dev/null | base64 > "$keyfile"
    fi
    echo "generated mongo/.data/keyfile"
  fi

  chmod 400 "$keyfile" 2>/dev/null || true
  if [[ "$(id -u)" -eq 0 ]]; then
    chown 999:999 "$keyfile"
    chmod 400 "$keyfile"
  elif sudo -n true 2>/dev/null; then
    sudo chown 999:999 "$keyfile"
    sudo chmod 400 "$keyfile"
  else
    # The official mongo image runs mongod as UID 999. Fix ownership without
    # requiring sudo on the host. Reuse the PostgreSQL image because the stack
    # needs it anyway.
    docker run --rm -u 0 -v "$keyfile:/keyfile" postgres:18 \
      sh -c 'chown 999:999 /keyfile && chmod 400 /keyfile'
  fi
}

for service in postgres mariadb mongo; do
  ensure_env "$service"
done

prepare_mongo_keyfile

compose postgres up -d --wait --wait-timeout 240
compose mariadb up -d --wait --wait-timeout 240
compose mongo up -d --wait --wait-timeout 240

"$ROOT/postgres/apply.sh"
"$ROOT/mariadb/apply.sh"
"$ROOT/mongo/apply.sh"

echo
"$ROOT/status.sh"
