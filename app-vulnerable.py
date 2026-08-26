#!/usr/bin/env python3
"""Globomantics Purchasing API -- internal build 4.2.1.

    ####################################################################
    #  DELIBERATELY VULNERABLE. Lab target only. Never deploy this.    #
    ####################################################################

Two information-disclosure defects are planted in this file's error handling.
Both are the kind that survive code review because the code around them looks
fine: validation is present, the ORM-less SQL "works", and debug mode is off.

Objective 2 of the lab is to fix them. Look for the TODO-1 / TODO-2 / TODO-3
banners below.

    ./run.sh vulnerable
"""
import os
import sqlite3
import traceback
import uuid
from datetime import datetime, timezone
from pathlib import Path

from flask import Flask, abort, g, jsonify, request
from werkzeug.exceptions import HTTPException

# ---------------------------------------------------------------------------
# TODO-1 (Lab Objective 2, Step 1) -- configuration and logging
#
# There is no environment switch here and no server-side logging. Every
# deployment of this service therefore behaves like a developer laptop, and
# when something goes wrong the only place the detail goes is the HTTP
# response. Add an APP_ENV setting that defaults to "production", and a
# logger that writes full detail to logs/api.log.
# ---------------------------------------------------------------------------
DB_PATH = os.environ.get("GLOBO_DB") or str(Path(__file__).resolve().parent / "globomantics.db")
HOST = os.environ.get("HOST", "127.0.0.1")
PORT = int(os.environ.get("PORT", "5000"))

app = Flask(__name__)


def db():
    """Per-request SQLite connection."""
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
    """Shape a product row for the client. supplier_cost_cents is withheld."""
    return {
        "sku": row["sku"],
        "name": row["name"],
        "price_cents": row["price_cents"],
        "stock_qty": row["stock_qty"],
    }


# ---------------------------------------------------------------------------
# TODO-2 (Lab Objective 2, Step 2) -- the global error handler
#
# This single handler catches every exception the application raises and
# serialises it straight back to the caller: exception type, message, and the
# complete Python traceback with absolute source paths.
#
# Replace it with two handlers:
#   * one for werkzeug HTTPException -- keep the real status code, return a
#     short message that describes the CLIENT's mistake and nothing else;
#   * one for Exception -- mint a correlation ID, log the full traceback
#     server side, and return only the generic message plus that ID.
# ---------------------------------------------------------------------------
@app.errorhandler(Exception)
def debug_error_handler(exc):
    status = exc.code if isinstance(exc, HTTPException) else 500
    return (
        jsonify(
            {
                "error": type(exc).__name__,
                "message": str(exc),
                "traceback": traceback.format_exc().splitlines(),
                "path": request.path,
                "method": request.method,
            }
        ),
        status,
    )


@app.get("/health")
def health():
    return jsonify(
        status="ok",
        service="globomantics-purchasing-api",
        version="4.2.1",
        env=os.environ.get("APP_ENV", "development"),
    )


@app.get("/api/products")
def list_products():
    term = request.args.get("search")

    if term is None:
        rows = db().execute(
            "SELECT sku, name, price_cents, stock_qty FROM products ORDER BY sku"
        ).fetchall()
        return jsonify([public_product(r) for r in rows])

    # -----------------------------------------------------------------------
    # TODO-3 (Lab Objective 2, Step 3) -- the product search query
    #
    # The search term is interpolated straight into the SQL string, so any
    # value that breaks the quoting raises sqlite3.OperationalError. The
    # except block below then hands the caller the failing statement and the
    # database path, which together disclose the schema -- including the
    # supplier_cost_cents column the API is careful never to return.
    #
    # Bind the search term as a parameter, and delete the except block so
    # database failures reach the global handler like everything else.
    # -----------------------------------------------------------------------
    sql = (
        "SELECT sku, name, price_cents, stock_qty, supplier_cost_cents "
        f"FROM products WHERE name LIKE '%{term}%' OR sku LIKE '%{term}%' "
        "ORDER BY sku"
    )
    try:
        rows = db().execute(sql).fetchall()
    except sqlite3.Error as exc:
        # Left in from the 4.1 release to help the on-call team debug faster.
        return (
            jsonify(
                {
                    "error": "database_error",
                    "exception": type(exc).__name__,
                    "detail": str(exc),
                    "query": sql,
                    "database": DB_PATH,
                }
            ),
            500,
        )
    return jsonify([public_product(r) for r in rows])


@app.post("/api/orders")
def create_order():
    # Malformed JSON raises werkzeug BadRequest right here, before a single
    # line of the validation below gets a chance to run.
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
    # debug=False on purpose. The leaks below are not Werkzeug's debugger --
    # they are this application's own error handling.
    app.run(host=HOST, port=PORT, debug=False)
