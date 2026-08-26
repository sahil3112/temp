#!/usr/bin/env python3
import http.server


class Handler(http.server.SimpleHTTPRequestHandler):
    def do_POST(self):
        self.send_response(302)
        self.send_header("Location", "dashboard.html")
        self.end_headers()


if __name__ == "__main__":
    http.server.test(HandlerClass=Handler, port=8080)
