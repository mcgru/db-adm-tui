#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SRC="$ROOT/home"
DEST="${HOME:?HOME is not set}"
STAMP="$(date -u +%Y%m%dT%H%M%SZ)"

backup() {
  local relative=$1
  if [[ -e "$DEST/$relative" ]]; then
    cp -a "$DEST/$relative" "$DEST/$relative.live-test-backup-$STAMP"
    echo "backup: $relative -> $relative.live-test-backup-$STAMP"
  fi
}

shopt -s nullglob
backup ".pgpass"
backup ".config/db-adm-tui/profiles.json"
for source in "$SRC"/.mongo.*; do
  backup "$(basename "$source")"
done

mkdir -p "$DEST/.config/db-adm-tui"
cp -a "$SRC/." "$DEST/"

chmod 700 "$DEST/.config/db-adm-tui"
chmod 600 "$DEST/.pgpass" "$DEST"/.mongo.* \
  "$DEST/.config/db-adm-tui/profiles.json" \
  "$DEST/LIVE-TEST-CREDENTIALS.md"

echo "installed live-test profiles into $DEST"
echo "source: $SRC"
