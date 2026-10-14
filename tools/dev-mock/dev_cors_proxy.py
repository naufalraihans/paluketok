"""Proxy dev untuk menambal CORS BE (bug dilaporkan, bukan diperbaiki).

Masalah: middleware.CORS di BE mengirim Access-Control-Allow-Methods tanpa
PUT, sehingga preflight browser memblokir PUT /v1/admin/settings/ai/{slot}.

Proxy ini meneruskan 8081 -> 8080 dan menambahkan PUT pada header CORS.
Hanya untuk pengembangan lokal; tidak dipakai di produksi.
"""
import http.server
import socketserver
import sys
import urllib.error
import urllib.request

TARGET = "http://127.0.0.1:8080"
PORT = 8081
ALLOW_METHODS = "GET, POST, PUT, PATCH, DELETE, OPTIONS"
ALLOW_HEADERS = "Authorization, Content-Type, Idempotency-Key"


class Proxy(http.server.BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def _cors(self) -> None:
        origin = self.headers.get("Origin")
        if origin:
            self.send_header("Access-Control-Allow-Origin", origin)
            self.send_header("Vary", "Origin")
            self.send_header("Access-Control-Allow-Methods", ALLOW_METHODS)
            self.send_header("Access-Control-Allow-Headers", ALLOW_HEADERS)
            self.send_header("Access-Control-Max-Age", "600")

    def _forward(self, method: str) -> None:
        length = int(self.headers.get("Content-Length") or 0)
        body = self.rfile.read(length) if length else None
        req = urllib.request.Request(TARGET + self.path, data=body, method=method)
        for h in ("Authorization", "Content-Type", "Idempotency-Key"):
            if self.headers.get(h):
                req.add_header(h, self.headers[h])
        try:
            with urllib.request.urlopen(req, timeout=120) as r:
                status, raw = r.status, r.read()
        except urllib.error.HTTPError as e:
            status, raw = e.code, e.read()
        except Exception as e:  # backend mati
            self.send_response(502)
            self._cors()
            self.send_header("Content-Length", "0")
            self.end_headers()
            print("proxy error:", type(e).__name__)
            return
        self.send_response(status)
        self._cors()
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(raw)))
        self.end_headers()
        self.wfile.write(raw)

    def do_OPTIONS(self) -> None:  # noqa: N802
        self.send_response(204)
        self._cors()
        self.send_header("Content-Length", "0")
        self.end_headers()

    def do_GET(self) -> None:  # noqa: N802
        self._forward("GET")

    def do_POST(self) -> None:  # noqa: N802
        self._forward("POST")

    def do_PATCH(self) -> None:  # noqa: N802
        self._forward("PATCH")

    def do_PUT(self) -> None:  # noqa: N802
        self._forward("PUT")

    def do_DELETE(self) -> None:  # noqa: N802
        self._forward("DELETE")

    def log_message(self, fmt: str, *args) -> None:
        sys.stderr.write("proxy %s\n" % (fmt % args))


if __name__ == "__main__":
    with socketserver.ThreadingTCPServer(("127.0.0.1", PORT), Proxy) as httpd:
        print(f"dev-cors-proxy listening :{PORT} -> {TARGET}")
        httpd.serve_forever()