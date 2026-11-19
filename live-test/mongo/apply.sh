#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"
if [[ ! -f .env ]]; then
  echo "mongo/.env is missing; run ../up.sh or copy .env.example to .env" >&2
  exit 1
fi

set -a
# shellcheck disable=SC1091
. ./.env
set +a

compose=(docker compose --env-file .env -f docker-compose.yml)
mongo=(mongosh --port "$MONGO_PORT" --quiet)

"${compose[@]}" exec -T mongo "${mongo[@]}" \
  -u "$MONGO_ROOT_USERNAME" -p "$MONGO_ROOT_PASSWORD" --authenticationDatabase admin \
  --eval "try {
    rs.status();
    print('replica set already initialized');
  } catch (e) {
    rs.initiate({
      _id: '${MONGO_REPLICA_SET}',
      members: [{ _id: 0, host: '${MONGO_REPLICA_HOST}' }]
    });
    print('replica set initialized');
  }"

for _ in $(seq 1 60); do
  primary=$(
    "${compose[@]}" exec -T mongo "${mongo[@]}" \
      -u "$MONGO_ROOT_USERNAME" -p "$MONGO_ROOT_PASSWORD" --authenticationDatabase admin \
      --eval 'db.hello().isWritablePrimary' 2>/dev/null | tr -d '\r' | tail -n1
  )
  [[ "$primary" == "true" ]] && break
  sleep 2
done
if [[ "${primary:-}" != "true" ]]; then
  echo "mongo: replica set did not elect a primary" >&2
  exit 1
fi

"${compose[@]}" exec -T mongo "${mongo[@]}" \
  -u "$MONGO_ROOT_USERNAME" -p "$MONGO_ROOT_PASSWORD" --authenticationDatabase admin \
  /scripts/01-init.js

echo "mongo: replica set, users, roles, and seed data are ready"
