import http.server
import socketserver
import os

DIST_DIR = os.path.abspath(
    os.path.join(os.path.dirname(__file__), "../apps/felicia-public-site/dist")
)

class NoCacheHandler(http.server.SimpleHTTPRequestHandler):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, directory=DIST_DIR, **kwargs)

    def end_headers(self):
        self.send_header("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
        self.send_header("Pragma", "no-cache")
        self.send_header("Expires", "0")
        super().end_headers()

if __name__ == "__main__":
    socketserver.TCPServer.allow_reuse_address = True
    with socketserver.TCPServer(("0.0.0.0", 5173), NoCacheHandler) as httpd:
        print(f"Serving {DIST_DIR} on 0.0.0.0:5173 with no-cache headers")
        httpd.serve_forever()
