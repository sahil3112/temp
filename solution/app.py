import re

from flask import Flask, request

import db
from routes_v2 import v2

SUNSET = "Thu, 31 Dec 2026 23:59:59 GMT"
DEPRECATED_SINCE = "@1767225600"  # 2026-01-01 (RFC 9745)
LEGACY_PATH = re.compile(r"^/(v1|beta)(/|$)")

app = Flask(__name__)
db.init()

app.register_blueprint(v2)
# v1 and beta are no longer registered, so the router answers 404 for them


@app.after_request
def sunset_legacy(resp):
    if LEGACY_PATH.match(request.path):
        resp.headers["Sunset"] = SUNSET
        resp.headers["Deprecation"] = DEPRECATED_SINCE
        resp.headers["Link"] = '</v2/users>; rel="successor-version"'
    return resp


@app.errorhandler(404)
def not_found(_):
    return {"error": "not found"}, 404
