#!/usr/bin/env bash

set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT"

case "${1:-vulnerable}" in
  vulnerable) ENTRY="app.py" ;;
  secure)     ENTRY="solution/app.py" ;;
  *) echo "usage: $0 [vulnerable|secure]" >&2; exit 2 ;;
esac

export GLOBO_DB="${GLOBO_DB:-$ROOT/globomantics.db}"
[ -f "$GLOBO_DB" ] || python seed.py

echo "starting $ENTRY on ${HOST:-127.0.0.1}:${PORT:-5000}  (db: $GLOBO_DB)"
exec python "$ENTRY"
