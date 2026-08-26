#!/usr/bin/env python3
"""Payload runner and leak verifier for the Globomantics purchasing API.

Standard library only -- nothing to install, works offline.

    python attack.py                 # against http://127.0.0.1:5000
    python attack.py --url http://127.0.0.1:8080
    python attack.py -v              # dump every response body

Exit status is the verification: 1 while any response still discloses
internals, 0 once every one of them is clean. Run it before you fix the API
and again afterwards.
"""
import argparse
import json
import re
import sys
import urllib.error
import urllib.parse
import urllib.request

# What "leaking" means, concretely. Each pattern is something a client has no
# business being told about the server that produced the response.
INDICATORS = [
    ("python traceback", re.compile(r"Traceback \(most recent call last\)")),
    ("source file path", re.compile(r'File\s+\\?"(?:/|[A-Za-z]:\\\\)[^"\\]+\.py')),
    ("dependency internals", re.compile(r"site-packages|dist-packages|/werkzeug/|/flask/")),
    ("sql statement", re.compile(r"\bSELECT\b[\s\S]{0,240}?\bFROM\b", re.IGNORECASE)),
    ("database driver error", re.compile(r"sqlite3|OperationalError|IntegrityError")),
    ("database file path", re.compile(r"[^\s\"']*\.db\b")),
    ("internal column name", re.compile(r"supplier_cost_cents")),
]

MALFORMED_JSON = b'{"sku": "GLM-1003", "quantity": 2, "customer_email": "buyer@globomantics.test",}'
VALID_ORDER = json.dumps(
    {"sku": "GLM-1003", "quantity": 1, "customer_email": "buyer@globomantics.test"}
).encode()

# (label, method, path, body). Steps 2 and 3 are the two attacks from Lab
# Objective 1; the rest are there to prove the fix did not break the API.
CASES = [
    ("Baseline: product catalogue", "GET", "/api/products", None),
    ("Attack 1: malformed JSON order", "POST", "/api/orders", MALFORMED_JSON),
    ("Attack 2: broken quoting in search", "GET", "/api/products?search=%27%20OR", None),
    ("Baseline: valid order", "POST", "/api/orders", VALID_ORDER),
    ("Probe: unknown order reference", "GET", "/api/orders/GLM-ORD-NOSUCH", None),
]


def send(base, method, path, body, timeout=10):
    req = urllib.request.Request(
        urllib.parse.urljoin(base, path),
        data=body,
        method=method,
        headers={"Content-Type": "application/json", "Accept": "application/json"},
    )
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            return resp.status, resp.read().decode("utf-8", "replace")
    except urllib.error.HTTPError as exc:
        # 4xx and 5xx are the interesting ones -- their bodies are the target.
        return exc.code, exc.read().decode("utf-8", "replace")


def scan(text):
    return [name for name, pattern in INDICATORS if pattern.search(text)]


def evidence(text, found):
    """First line of the body that trips any indicator, trimmed for the table."""
    for line in text.splitlines():
        if any(p.search(line) for n, p in INDICATORS if n in found):
            line = line.strip().strip(",").strip('"')
            return line[:110] + ("..." if len(line) > 110 else "")
    return ""


def reference_id(text):
    try:
        return json.loads(text).get("reference_id")
    except (ValueError, AttributeError):
        return None


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--url", default="http://127.0.0.1:5000", help="base URL of the API")
    ap.add_argument("-v", "--verbose", action="store_true", help="print full response bodies")
    args = ap.parse_args()
    base = args.url.rstrip("/") + "/"

    try:
        send(base, "GET", "/health", None, timeout=5)
    except urllib.error.URLError as exc:
        sys.exit(f"cannot reach {base} ({exc.reason}). Start the API first: ./run.sh vulnerable")

    print(f"target: {base}\n")
    print(f"{'#':<3}{'STEP':<40}{'STATUS':<8}{'VERDICT':<9}DISCLOSED")
    print("-" * 100)

    leaking = 0
    ids = []
    for i, (label, method, path, body) in enumerate(CASES, 1):
        status, text = send(base, method, path, body)
        found = scan(text)
        if found:
            leaking += 1
        rid = reference_id(text)
        if rid:
            ids.append((label, rid))
        verdict = "LEAK" if found else "clean"
        print(f"{i:<3}{label:<40}{status:<8}{verdict:<9}{', '.join(found) or '-'}")
        if found:
            print(f"{'':<60}{evidence(text, found)}")
        if args.verbose:
            print(f"\n--- response body ---\n{text}\n")

    print("-" * 100)
    if ids:
        print("\ncorrelation IDs returned (grep these in logs/api.log):")
        for label, rid in ids:
            print(f"  {rid}   {label}")

    if leaking:
        print(f"\nFAIL: {leaking} of {len(CASES)} responses disclosed internal detail.")
        return 1
    print(f"\nPASS: all {len(CASES)} responses are free of internal detail.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
