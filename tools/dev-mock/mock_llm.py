"""Mock LLM OpenAI-compatible (dev/test local, TANPA key & biaya).

Tujuan: menjalankan flow Fase 10 end-to-end (register -> project -> run done ->
regenerate) tanpa memanggil LLM sungguhan. Tahap dibaca dari prompt yang
disusun internal/worker/prompt.go ("Buat artefak tahap \"<stage>\"."), lalu
artefak dibalas dari mock_art.py yang memenuhi internal/worker/validate.go.

Jalankan: python mock_llm.py  (default :9098)
"""
import json
import re
import sys
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

sys.path.insert(0, __file__.rsplit("\\", 1)[0])
from mock_art import build  # noqa: E402

STAGE_RE = re.compile(r'tahap "([a-z]+)"')


class Handler(BaseHTTPRequestHandler):
    def log_message(self, *a):  # senyap: jangan banjiri stdout saat screenshot
        pass

    def do_POST(self):
        n = int(self.headers.get("Content-Length", "0"))
        req = json.loads(self.rfile.read(n) or b"{}")
        text = "\n".join(str(m.get("content", "")) for m in req.get("messages", []))
        m = STAGE_RE.search(text)
        stage = m.group(1) if m else "prd"
        body = {
            "id": "mock-1",
            "model": req.get("model", "mock"),
            "choices": [{"index": 0, "message": {"role": "assistant", "content": build(stage)}, "finish_reason": "stop"}],
            "usage": {"prompt_tokens": 100, "completion_tokens": 200, "total_tokens": 300},
        }
        raw = json.dumps(body).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(raw)))
        self.end_headers()
        self.wfile.write(raw)


if __name__ == "__main__":
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 9098
    print(f"mock-llm listening :{port}", flush=True)
    ThreadingHTTPServer(("127.0.0.1", port), Handler).serve_forever()