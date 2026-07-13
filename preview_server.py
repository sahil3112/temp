#!/usr/bin/env python3
# ponytail: preview-only convenience so direct-clicking Sign In here doesn't 501.
# The real, graded login flow runs through SET's Credential Harvester clone
# (port 80, see poc/), not this static server — this just avoids a raw error
# page when eyeballing the standalone portal.
import http.server


class Handler(http.server.SimpleHTTPRequestHandler):
    def do_POST(self):
        self.send_response(302)
        self.send_header("Location", "dashboard.html")
        self.end_headers()


if __name__ == "__main__":
    http.server.test(HandlerClass=Handler, port=8080)
