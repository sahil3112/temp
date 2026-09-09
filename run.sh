#!/usr/bin/env bash

set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT"

[ -d .venv ] || python3 -m venv .venv
# shellcheck disable=SC1091
source .venv/bin/activate

python -c 'import fastapi, uvicorn, pydantic, yaml' 2>/dev/null || pip install -r requirements.txt

HOST="${HOST:-127.0.0.1}"
PORT="${PORT:-8000}"

echo "Globomantics Data Processing API"
echo "  GET  http://${HOST}:${PORT}/health"
echo "  POST http://${HOST}:${PORT}/api/v1/pack     (Objective 1 — memory safety)"
echo "  POST http://${HOST}:${PORT}/api/v1/ingest   (Objective 2 — YAML ingest)"
echo "  POST http://${HOST}:${PORT}/api/v1/jobs     (Objective 3 — JSON jobs)"
echo "  docs http://${HOST}:${PORT}/docs"
echo
echo "Ctrl+C to stop. Restart ./run.sh after you save a patch."

exec uvicorn server:app --host "$HOST" --port "$PORT"
