#!/usr/bin/env bash
# Start the Globomantics purchasing API.
#   ./run.sh            -> vulnerable build (Lab Objective 1)
#   ./run.sh secure     -> hardened build   (Lab Objective 2 answer key)
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT"

case "${1:-vulnerable}" in
  vulnerable) ENTRY="app.py" ;;
  secure)     ENTRY="solution/app.py" ;;
  *) echo "usage: $0 [vulnerable|secure]" >&2; exit 2 ;;
esac

[ -d .venv ] || python3 -m venv .venv
# shellcheck disable=SC1091
source .venv/bin/activate

# Offline friendly: only reach for the network if Flask is genuinely missing.
python -c 'import flask' 2>/dev/null || pip install -r requirements.txt

export GLOBO_DB="${GLOBO_DB:-$ROOT/globomantics.db}"
[ -f "$GLOBO_DB" ] || python seed.py

echo "starting $ENTRY on ${HOST:-127.0.0.1}:${PORT:-5000}  (db: $GLOBO_DB)"
exec python "$ENTRY"
