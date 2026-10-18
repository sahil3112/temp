import functools
import os
import time

import jwt
from flask import g, request

SECRET = os.environ.get("JWT_SECRET", "globomantics-lab-secret")  # ponytail: lab default, use a real secret store in prod


def issue_token(user):
    claims = {"sub": str(user["id"]), "role": user["role"], "exp": int(time.time()) + 3600}
    return jwt.encode(claims, SECRET, algorithm="HS256")


def login_required(f):
    @functools.wraps(f)
    def wrapper(*args, **kwargs):
        try:
            claims = jwt.decode(request.headers.get("Authorization", "").removeprefix("Bearer "),
                                SECRET, algorithms=["HS256"])
        except jwt.PyJWTError:
            return {"error": "unauthorized"}, 401
        g.uid, g.role = int(claims["sub"]), claims["role"]
        return f(*args, **kwargs)
    return wrapper
