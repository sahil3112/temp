from flask import Blueprint, jsonify, request

import db

v1 = Blueprint("v1", __name__, url_prefix="/v1")
beta = Blueprint("beta", __name__, url_prefix="/beta")


@v1.get("/users")
@beta.get("/users")
def list_users():
    return jsonify(db.query("SELECT * FROM users"))


@v1.get("/users/<int:user_id>")
@beta.get("/users/<int:user_id>")
def get_user(user_id):
    user = db.query("SELECT * FROM users WHERE id = ?", (user_id,), one=True)
    if user is None:
        return {"error": "not found"}, 404
    return jsonify(user)


@v1.put("/users/<int:user_id>")
def update_user(user_id):
    email = (request.get_json(silent=True) or {}).get("email")
    if not email:
        return {"error": "email required"}, 400
    if not db.execute("UPDATE users SET email = ? WHERE id = ?", (email, user_id)):
        return {"error": "not found"}, 404
    return jsonify(db.query("SELECT * FROM users WHERE id = ?", (user_id,), one=True))


@v1.delete("/users/<int:user_id>")
def delete_user(user_id):
    if not db.execute("DELETE FROM users WHERE id = ?", (user_id,)):
        return {"error": "not found"}, 404
    return "", 204
