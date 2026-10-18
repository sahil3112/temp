import os
import sqlite3

from werkzeug.security import generate_password_hash

PATH = os.path.join(os.path.dirname(os.path.abspath(__file__)), "users.db")

SEED = [  # id, username, email, password, role, ssn, api_key (all fake)
    (1, "admin", "admin@globomantics.com", "Adm1n#2024", "admin", "900-00-0001", "gk_live_admin_7f3a91c2"),
    (2, "alice", "alice@globomantics.com", "Alice#2024", "user", "900-00-0002", "gk_live_alice_4be07d15"),
    (3, "bob", "bob@globomantics.com", "Bob#2024", "user", "900-00-0003", "gk_live_bob_c8d21e60"),
]


def init():
    # ponytail: rebuilt on every start so each lab run begins from the same data
    if os.path.exists(PATH):
        os.remove(PATH)
    c = sqlite3.connect(PATH)
    c.execute("""CREATE TABLE users (id INTEGER PRIMARY KEY, username TEXT UNIQUE, email TEXT,
                 password_hash TEXT, role TEXT, ssn TEXT, api_key TEXT)""")
    c.executemany("INSERT INTO users VALUES (?,?,?,?,?,?,?)",
                  [(i, u, e, generate_password_hash(p, method="pbkdf2:sha256"), r, s, k)
                   for i, u, e, p, r, s, k in SEED])
    c.commit()
    c.close()


def _run(sql, args):
    c = sqlite3.connect(PATH)
    c.row_factory = sqlite3.Row
    try:
        with c:
            cur = c.execute(sql, args)
            return [dict(r) for r in cur.fetchall()], cur.rowcount
    finally:
        c.close()


def query(sql, args=(), one=False):
    rows = _run(sql, args)[0]
    return (rows[0] if rows else None) if one else rows


def execute(sql, args=()):
    return _run(sql, args)[1]  # rows affected
