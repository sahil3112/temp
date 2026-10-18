from flask import Blueprint, g, jsonify, request
from werkzeug.security import check_password_hash

import auth
import db

v2 = Blueprint("v2", __name__, url_prefix="/v2")

PUBLIC = "id, username, email, role"


def _allowed(user_id):
    return g.role == "admin" or g.uid == user_id


@v2.post("/auth/login")
def login():
    body = request.get_json(silent=True) or {}
    user = db.query("SELECT * FROM users WHERE username = ?", (body.get("username", ""),), one=True)
    if user is None or not check_password_hash(user["password_hash"], body.get("password", "")):
        return {"error": "invalid credentials"}, 401
    return {"token": auth.issue_token(user)}


@v2.get("/users")
@auth.login_required
def list_users():
    if g.role != "admin":
        return {"error": "forbidden"}, 403
    return jsonify(db.query(f"SELECT {PUBLIC} FROM users"))


@v2.get("/users/<int:user_id>")
@auth.login_required
def get_user(user_id):
    if not _allowed(user_id):
        return {"error": "forbidden"}, 403
    user = db.query(f"SELECT {PUBLIC} FROM users WHERE id = ?", (user_id,), one=True)
    if user is None:
        return {"error": "not found"}, 404
    return jsonify(user)


@v2.put("/users/<int:user_id>")
@auth.login_required
def update_user(user_id):
    if not _allowed(user_id):
        return {"error": "forbidden"}, 403
    email = (request.get_json(silent=True) or {}).get("email")
    if not email:
        return {"error": "email required"}, 400
    if not db.execute("UPDATE users SET email = ? WHERE id = ?", (email, user_id)):
        return {"error": "not found"}, 404
    return jsonify(db.query(f"SELECT {PUBLIC} FROM users WHERE id = ?", (user_id,), one=True))


@v2.delete("/users/<int:user_id>")
@auth.login_required
def delete_user(user_id):
    if g.role != "admin":
        return {"error": "forbidden"}, 403
    if not db.execute("DELETE FROM users WHERE id = ?", (user_id,)):
        return {"error": "not found"}, 404
    return "", 204
