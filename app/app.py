from flask import Flask

import db
from routes_v1 import beta, v1
from routes_v2 import v2

app = Flask(__name__)
db.init()

app.register_blueprint(v2)
app.register_blueprint(v1)
app.register_blueprint(beta)
