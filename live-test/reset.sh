#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [[ "${1:-}" != "--yes" ]]; then
  read -r -p "Delete postgres/.data, mariadb/.data, and mongo/.data? [y/N] " answer
  [[ "$answer" == "y" || "$answer" == "Y" ]] || exit 1
fi

remove_data() {
  local service=$1
  local path="$ROOT/$service/.data"
  [[ -d "$path" ]] || return 0

  # Database files are owned by the container UIDs; fall back to a root
  # container when the host user cannot delete them. The postgres image is used
  # as the helper because it declares no volumes below /data.
  if rm -rf "$path" 2>/dev/null; then
    return 0
  fi
  docker run --rm -u 0 -w / -v "$path:/data" postgres:18 \
    sh -c 'find /data -mindepth 1 -delete'
  rm -rf "$path"
}

"$ROOT/down.sh"
remove_data postgres
remove_data mariadb
remove_data mongo
"$ROOT/up.sh"
