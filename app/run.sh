#!/usr/bin/env bash
# Starts the Globomantics user API on :5000; the reloader restarts it when you save a file.
cd "$(dirname "$0")" && exec python3 -m flask --app app run --host 0.0.0.0 --port "${PORT:-5000}" --reload
