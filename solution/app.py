#!/usr/bin/env python3
"""Globomantics Purchasing API -- internal build 4.3.0 (hardened).

Answer key for Lab Objective 2. Same routes and same behaviour as ../app.py
for every request that succeeds; the difference is entirely in what leaves
the process when a request fails.

What changed, against the three TODO banners in ../app.py:

  TODO-1  APP_ENV, defaulting to "production" so a missing or misspelled
          variable fails safe, plus a logger that keeps the full detail
          server side in logs/api.log.
  TODO-2  One handler for expected client errors (real status code, short
          message about the caller's mistake) and one for everything else
          (correlation ID out, traceback to the log).
  TODO-3  The search term is bound as a parameter and the endpoint no longer
          builds its own error response.

    ./run.sh secure
"""
import logging
import os
import sqlite3
import uuid
from datetime import datetime, timezone
from pathlib import Path

from flask import Flask, abort, g, jsonify, request
from werkzeug.exceptions import HTTPException

REPO_ROOT = Path(__file__).resolve().parents[1]

# Fail safe: anything other than an explicit "development" is production.
APP_ENV = os.environ.get("APP_ENV", "production").strip().lower()
DB_PATH = os.environ.get("GLOBO_DB") or str(REPO_ROOT / "globomantics.db")
LOG_DIR = Path(os.environ.get("GLOBO_LOG_DIR") or (REPO_ROOT / "logs"))
HOST = os.environ.get("HOST", "127.0.0.1")
PORT = int(os.environ.get("PORT", "5000"))

LOG_DIR.mkdir(parents=True, exist_ok=True)
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(levelname)-8s %(name)s %(message)s",
    handlers=[
        logging.FileHandler(LOG_DIR / "api.log", encoding="utf-8"),
        logging.StreamHandler(),
    ],
)
log = logging.getLogger("globomantics.api")

app = Flask(__name__)


def db():
    if "db" not in g:
        g.db = sqlite3.connect(DB_PATH)
        g.db.row_factory = sqlite3.Row
    return g.db


@app.teardown_appcontext
def close_db(_exc):
    conn = g.pop("db", None)
    if conn is not None:
        conn.close()


def public_product(row):
    return {
        "sku": row["sku"],
        "name": row["name"],
        "price_cents": row["price_cents"],
        "stock_qty": row["stock_qty"],
    }


# ---------------------------------------------------------------------------
# Global exception filter
# ---------------------------------------------------------------------------
# Expected client errors keep their real status code, because a 404 that
# answers 500 is its own kind of broken. What they lose is any wording that
# describes the server: no exception types, no paths, no SQL.
CLIENT_MESSAGES = {
    400: "Malformed request body.",
    404: "Resource not found.",
    405: "Method not allowed for this resource.",
    409: "The request conflicts with the current state of the resource.",
    415: "Unsupported media type.",
    422: "Request validation failed.",
    429: "Too many requests.",
}

GENERIC_MESSAGE = "An unexpected error occurred."


def _correlation_id():
    return uuid.uuid4().hex[:12].upper()


@app.errorhandler(HTTPException)
def handle_client_error(exc):
    # A 5xx dressed up as an HTTPException is still an incident: send it down
    # the unexpected-error path so it gets an ID and a log entry.
    if exc.code is None or exc.code >= 500:
        return handle_unexpected_error(exc)
    return (
        jsonify(
            {
                "error": exc.name,
                "message": CLIENT_MESSAGES.get(exc.code, "The request could not be processed."),
            }
        ),
        exc.code,
    )


@app.errorhandler(Exception)
def handle_unexpected_error(exc):
    correlation_id = _correlation_id()

    # Everything the vulnerable build put in the response body goes here
    # instead. exc_info=True writes the full traceback to logs/api.log.
    log.error(
        "unhandled exception correlation_id=%s method=%s path=%s remote=%s",
        correlation_id,
        request.method,
        request.path,
        request.remote_addr,
        exc_info=exc,
    )

    body = {
        "error": "internal_server_error",
        "message": f"{GENERIC_MESSAGE} Reference ID: {correlation_id}",
        "reference_id": correlation_id,
    }

    # Environment parity: the same code path runs in every environment. A
    # developer opts into detail explicitly; nobody inherits it by accident.
    if APP_ENV == "development":
        body["debug"] = {"exception": type(exc).__name__, "detail": str(exc)}

    return jsonify(body), 500


@app.get("/health")
def health():
    return jsonify(
        status="ok",
        service="globomantics-purchasing-api",
        version="4.3.0",
        env=APP_ENV,
    )


@app.get("/api/products")
def list_products():
    term = request.args.get("search")

    if term is None:
        rows = db().execute(
            "SELECT sku, name, price_cents, stock_qty FROM products ORDER BY sku"
        ).fetchall()
        return jsonify([public_product(r) for r in rows])

    # Bound parameter: the term is data, never statement text. A quote in the
    # search box is now a character to match on, not a syntax error to leak.
    pattern = f"%{term}%"
    rows = db().execute(
        "SELECT sku, name, price_cents, stock_qty FROM products"
        " WHERE name LIKE ? OR sku LIKE ? ORDER BY sku",
        (pattern, pattern),
    ).fetchall()
    return jsonify([public_product(r) for r in rows])


@app.post("/api/orders")
def create_order():
    payload = request.get_json(force=True)

    if not isinstance(payload, dict):
        abort(400, description="Request body must be a JSON object.")

    missing = [f for f in ("sku", "quantity", "customer_email") if f not in payload]
    if missing:
        abort(400, description=f"Missing required field(s): {', '.join(missing)}.")

    try:
        quantity = int(payload["quantity"])
    except (TypeError, ValueError):
        abort(400, description="Field 'quantity' must be an integer.")
    if quantity < 1:
        abort(400, description="Field 'quantity' must be at least 1.")

    conn = db()
    product = conn.execute(
        "SELECT sku, price_cents, stock_qty FROM products WHERE sku = ?",
        (payload["sku"],),
    ).fetchone()
    if product is None:
        abort(404, description="Unknown SKU.")
    if quantity > product["stock_qty"]:
        abort(409, description="Insufficient stock for the requested quantity.")

    reference = "GLM-ORD-" + uuid.uuid4().hex[:6].upper()
    total_cents = product["price_cents"] * quantity
    created_at = datetime.now(timezone.utc).isoformat(timespec="seconds")
    with conn:
        conn.execute(
            "INSERT INTO orders (reference, sku, quantity, total_cents, customer_email, created_at)"
            " VALUES (?, ?, ?, ?, ?, ?)",
            (reference, product["sku"], quantity, total_cents, str(payload["customer_email"]), created_at),
        )
        conn.execute(
            "UPDATE products SET stock_qty = stock_qty - ? WHERE sku = ?",
            (quantity, product["sku"]),
        )

    log.info("order created reference=%s sku=%s qty=%s", reference, product["sku"], quantity)
    return (
        jsonify(
            {
                "reference": reference,
                "sku": product["sku"],
                "quantity": quantity,
                "total_cents": total_cents,
                "status": "confirmed",
                "created_at": created_at,
            }
        ),
        201,
    )


@app.get("/api/orders/<reference>")
def get_order(reference):
    row = db().execute(
        "SELECT reference, sku, quantity, total_cents, customer_email, created_at"
        " FROM orders WHERE reference = ?",
        (reference,),
    ).fetchone()
    if row is None:
        abort(404, description="Order not found.")
    return jsonify(dict(row))


if __name__ == "__main__":
    log.info("starting globomantics-purchasing-api env=%s db=%s", APP_ENV, DB_PATH)
    app.run(host=HOST, port=PORT, debug=False)
