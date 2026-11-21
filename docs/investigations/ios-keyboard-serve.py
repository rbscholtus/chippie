#!/usr/bin/env python3
"""Round 6 of docs/investigations/ios-keyboard.md: serves the keyboard-experiment
builds to the phone, no-store, with ios-keyboard-diag2.js injected into the HTML
pages, and collects the phone's log (POST /log) into <work>/phone.log. One port
per build; the experiment list is at http://<mac>:8081/menu.

Usage: python3 docs/investigations/ios-keyboard-serve.py <work dir>
The work dir (outside the repository) holds the other builds, if present:
  pr27-build/   git worktree at 276d074 (Chippie PR #27), ./scripts/build.sh run
  pre27-build/  git worktree at cad8d09 (focus on release), ./scripts/build.sh run
  prototype/    a copy of prototype/, built with its build.sh, plus
                web/chippie-gz.wasm copied from web/chippie.wasm.gz
Port 8081 serves this repository's web/dist (./scripts/build.sh)."""
import json
import os
import sys
import threading
import time
from http.server import SimpleHTTPRequestHandler, ThreadingHTTPServer

HERE = os.path.dirname(os.path.abspath(__file__))
REPO = os.path.abspath(os.path.join(HERE, "..", ".."))
WORK = os.path.abspath(sys.argv[1]) if len(sys.argv) > 1 else sys.exit(__doc__)
BUILDS = {
    8081: ("branch", os.path.join(REPO, "web/dist")),
    8082: ("pr27", os.path.join(WORK, "pr27-build/web/dist")),
    8083: ("pre27", os.path.join(WORK, "pre27-build/web/dist")),
    8084: ("prototype", os.path.join(WORK, "prototype/web")),
}
LOG = os.path.join(WORK, "phone.log")
lock = threading.Lock()

ON = "Tap to switch the TV on (the ROM list opens)."
FIELD2 = ON + " Tap the search field <b>once</b>; wait 3 s. Then tap the search field <b>again</b>; wait 3 s."
EXPERIMENTS = [
    ("1", 8084, "local.html?x=control", "Prototype (known good on 8 Nov)",
     "Tap the field under &ldquo;B &middot; Ebitengine textinput (IME)&rdquo; once; wait 3 s. If no keyboard: tap it again. Then tap field A."),
    ("2", 8081, "?x=control", "This branch, unchanged", FIELD2),
    ("3", 8082, "?x=control", "Chippie PR #27 (no re-arm)", FIELD2),
    ("4", 8083, "?x=control", "Before PR #27 (focus on release)", FIELD2 + " Then a <b>third</b> tap on the field."),
    ("5", 8081, "?x=clone&at=0,0", "Copy of Ebitengine's textarea at 0,0",
     ON + " Then tap the &ldquo;PLAY (309)&rdquo; heading (do <b>not</b> touch the search field)."),
    ("6", 8081, "?x=clone&at=40,87", "Copy at 40,87 (where Ebitengine puts it)",
     ON + " Then tap the &ldquo;PLAY (309)&rdquo; heading (do <b>not</b> touch the search field)."),
    ("7", 8081, "?x=nomove", "Real field, Ebitengine's move undone", ON + " Tap the search field once; wait 3 s."),
    ("8", 8081, "?x=away&at=200,400", "Real field, textarea moved to 200,400", ON + " Tap the search field once; wait 3 s."),
    ("9", 8081, "?x=norewrite", "Real field, value rewrite swallowed", ON + " Tap the search field once; wait 3 s."),
    ("1b", 8084, "index.html?x=control", "Prototype, page without doctype/viewport",
     "Only if 1 showed no keyboard: same steps as 1."),
    ("10", 8081, "?x=native", "Chrome: what clears keyboard-down's stuck state?",
     ON + " (a) Tap the search field: keyboard. (b) Chrome's keyboard-down key. (c) Tap the search field: keyboard?"
     " (d) Reload with Chrome's reload button, switch on, tap the search field: keyboard?"
     " (e) If not: tap the small orange &ldquo;tap&rdquo; box at the bottom right: keyboard?"
     " (f) With that keyboard still open, tap the search field: keyboard?"),
]


def menu(host):
    rows = "".join(
        f'<li><a href="http://{host}:{port}/{q}"><b>{n}.</b> {title}</a><br><small>{steps}</small></li>'
        for n, port, q, title, steps in EXPERIMENTS
    )
    return f"""<!doctype html><meta charset=utf-8>
<meta name=viewport content="width=device-width,initial-scale=1">
<title>Keyboard experiments</title>
<style>body{{font:16px/1.4 system-ui;margin:16px;background:#16130f;color:#efe3cc}}
a{{color:#ffb43c;font-size:18px}} li{{margin:0 0 14px}} small{{color:#a8977a}}</style>
<h3>Keyboard experiments</h3>
<p><small>Each link loads a fresh page. Do the steps, then come back here with Safari's Back button.
The bottom line turns <span style="color:#7f7">green</span> when the page sees a keyboard.</small></p>
<ol style="padding-left:0;list-style:none">{rows}</ol>"""


def handler_for(build, root):
    class H(SimpleHTTPRequestHandler):
        def __init__(self, *a, **kw):
            super().__init__(*a, directory=root, **kw)

        def end_headers(self):
            self.send_header("Cache-Control", "no-store")
            super().end_headers()

        def log_message(self, fmt, *args):
            pass  # quiet

        def do_GET(self):
            path = self.path.split("?", 1)[0]
            if path == "/menu":
                host = self.headers.get("Host", "").split(":")[0]
                data = menu(host).encode()
                self.send_response(200)
                self.send_header("Content-Type", "text/html; charset=utf-8")
                self.send_header("Content-Length", str(len(data)))
                self.end_headers()
                self.wfile.write(data)
                return
            if path in ("/", "/index.html", "/local.html"):
                name = "local.html" if path == "/local.html" else "index.html"
                with open(os.path.join(root, name), encoding="utf-8") as f:
                    page = f.read()
                with open(os.path.join(HERE, "ios-keyboard-diag2.js"), encoding="utf-8") as f:
                    diag = "<script>\n" + f.read() + "\n</script>\n"
                i = page.rfind("</body>")
                page = page[:i] + diag + page[i:] if i >= 0 else page + diag
                data = page.encode()
                self.send_response(200)
                self.send_header("Content-Type", "text/html; charset=utf-8")
                self.send_header("Content-Length", str(len(data)))
                self.end_headers()
                self.wfile.write(data)
                with lock, open(LOG, "a", encoding="utf-8") as f:
                    f.write(json.dumps({"srv": time.strftime("%H:%M:%S"), "build": build, "kind": "page-served", "path": self.path, "ip": self.client_address[0]}) + "\n")
                return
            super().do_GET()

        def do_POST(self):
            if self.path != "/log":
                self.send_error(404)
                return
            n = int(self.headers.get("Content-Length") or 0)
            body = self.rfile.read(n).decode("utf-8", "replace")
            stamp = time.strftime("%H:%M:%S")
            with lock, open(LOG, "a", encoding="utf-8") as f:
                for line in body.splitlines():
                    if line.strip():
                        try:
                            rec = json.loads(line)
                        except ValueError:
                            rec = {"raw": line}
                        rec = {"srv": stamp, "build": build, **rec}
                        f.write(json.dumps(rec) + "\n")
            self.send_response(204)
            self.end_headers()

    return H


def main():
    servers = []
    for port, (build, root) in BUILDS.items():
        if not os.path.isdir(root):
            print(f"skip {build}: {root} missing", file=sys.stderr)
            continue
        s = ThreadingHTTPServer(("0.0.0.0", port), handler_for(build, root))
        threading.Thread(target=s.serve_forever, daemon=True).start()
        servers.append(s)
        print(f"{build}: http://0.0.0.0:{port}/  ({root})", flush=True)
    threading.Event().wait()


if __name__ == "__main__":
    main()
