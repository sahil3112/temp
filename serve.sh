#!/usr/bin/env bash
# Serves the Globomantics portal locally on :8080 for SET's Site Cloner to clone.
cd "$(dirname "$0")"
python3 preview_server.py
