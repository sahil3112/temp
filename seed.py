#!/usr/bin/env python3
"""Create (or reset) the Globomantics purchasing database.

Safe to re-run at any point in the lab -- it drops both tables and rebuilds
them, which is how you get stock levels back after hammering the API.

    python seed.py
"""
import os
import sqlite3
from pathlib import Path

DB_PATH = os.environ.get("GLOBO_DB") or str(Path(__file__).resolve().parent / "globomantics.db")

SCHEMA = """
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS products;

CREATE TABLE products (
    id                  INTEGER PRIMARY KEY,
    sku                 TEXT    NOT NULL UNIQUE,
    name                TEXT    NOT NULL,
    price_cents         INTEGER NOT NULL,
    stock_qty           INTEGER NOT NULL,
    supplier_cost_cents INTEGER NOT NULL
);

CREATE TABLE orders (
    id             INTEGER PRIMARY KEY,
    reference      TEXT    NOT NULL UNIQUE,
    sku            TEXT    NOT NULL,
    quantity       INTEGER NOT NULL,
    total_cents    INTEGER NOT NULL,
    customer_email TEXT    NOT NULL,
    created_at     TEXT    NOT NULL
);
"""

# supplier_cost_cents is commercially sensitive. The API never returns it --
# which is exactly why seeing it inside a leaked SQL statement matters.
PRODUCTS = [
    ("GLM-1001", "Globomantics SmartHub Pro",      24999,  12, 14200),
    ("GLM-1002", "Globomantics Mesh Router X6",    18950,  40, 10100),
    ("GLM-1003", "Globomantics Thermal Sensor Kit", 7499, 220,  2600),
    ("GLM-1004", "Globomantics Edge Gateway 500",  43900,   5, 27500),
    ("GLM-1005", "Globomantics Solar Relay Node",  12750,  88,  6300),
]


def main():
    conn = sqlite3.connect(DB_PATH)
    with conn:
        conn.executescript(SCHEMA)
        conn.executemany(
            "INSERT INTO products (sku, name, price_cents, stock_qty, supplier_cost_cents)"
            " VALUES (?, ?, ?, ?, ?)",
            PRODUCTS,
        )
    conn.close()
    print(f"seeded {len(PRODUCTS)} products into {DB_PATH}")


if __name__ == "__main__":
    main()
