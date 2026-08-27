#!/usr/bin/env python3

import os
import sqlite3
import traceback
import uuid
from datetime import datetime, timezone
from pathlib import Path

from flask import Flask, abort, g, jsonify, request
from werkzeug.exceptions import HTTPException

DB_PATH = os.environ.get("GLOBO_DB") or str(Path(__file__).resolve().parent / "globomantics.db")
HOST = os.environ.get("HOST", "127.0.0.1")
PORT = int(os.environ.get("PORT", "5000"))

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
    app.run(host=HOST, port=PORT, debug=False)
