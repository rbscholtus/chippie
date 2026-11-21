// Keyboard diagnostics v2 (round 6) for docs/investigations/ios-keyboard.md.
// Injected by ios-keyboard-serve.py at serve time; never part of the app.
// Sends its log to the server (POST /log), shows one status line, adds no
// buttons.
//
// Experiments (?x=..., one per fresh page load):
//   control              nothing changed
//   clone&at=X,Y[&tap=N] on the N-th canvas touchend (default 2: the 1st switches
//                        the TV on), focus a copy of Ebitengine's textarea placed
//                        at X,Y before the focus, from a body touchend listener
//                        (the same place and phase as Ebitengine's own handler).
//                        &move=1: place it at 0,0, then move it to X,Y right after
//                        the focus, as Ebitengine does.
//   nomove               real flow; Ebitengine's move after its deferred focus is
//                        undone in the same task (left/top back to 0)
//   away&at=X,Y          real flow; the textarea is moved to X,Y instead
//   norewrite            real flow; for 1 s after the deferred focus, writes to the
//                        textarea's value and setSelectionRange are swallowed
//   native               a small visible <input> at the bottom right, focused only
//                        by the finger (experiment 10, §11.5)
(() => {
  "use strict";
  const P = new URLSearchParams(location.search);
  const X = P.get("x") || "control";
  const AT = (P.get("at") || "0,0").split(",").map(Number);
  const TAP = Number(P.get("tap") || 2);
  const MOVE = P.get("move") === "1";
  const sid = Math.random().toString(36).slice(2, 7);
  const t0 = performance.now();
  const EBI = "ebitengine-textinput";

  // ---- logging -------------------------------------------------------------
  const queue = [];
  const recent = [];
  const ts = () => +((performance.now() - t0) / 1000).toFixed(3);
  const name = (e) => !e ? "-" : e === window ? "window" : e === document ? "document" : (e.id || e.tagName || String(e));
  const flush = () => {
    if (!queue.length) return;
    const body = queue.splice(0).join("\n");
    try {
      if (!navigator.sendBeacon("/log", body)) throw new Error("beacon refused");
    } catch (_) {
      fetch("/log", { method: "POST", body, keepalive: true }).catch(() => {});
    }
  };
  const log = (kind, data = {}) => {
    queue.push(JSON.stringify({ sid, x: X, port: location.port, t: ts(), kind, ...data }));
    recent.push(kind);
    if (recent.length > 3) recent.shift();
  };
  setInterval(flush, 400);
  addEventListener("pagehide", flush);
  document.addEventListener("visibilitychange", () => { log("visibility", { state: document.visibilityState }); flush(); });
  addEventListener("error", (e) => log("js-error", { msg: String(e.message), src: e.filename + ":" + e.lineno }));

  const vv = window.visualViewport;
  const vvInfo = () => vv ? { vvH: Math.round(vv.height), vvTop: Math.round(vv.offsetTop), vvPageTop: Math.round(vv.pageTop), vvScale: +vv.scale.toFixed(3) } : {};
  const metas = () => [...document.querySelectorAll('meta[name="viewport"]')].map((m) => m.content);

  log("session", {
    ua: navigator.userAgent, url: location.href, compat: document.compatMode,
    screen: [screen.width, screen.height], dpr: devicePixelRatio,
    inner: [innerWidth, innerHeight], outer: [outerWidth, outerHeight], ...vvInfo(),
    maxTouch: navigator.maxTouchPoints, coarse: matchMedia("(pointer: coarse)").matches,
    params: Object.fromEntries(P),
  });
  setTimeout(() => log("viewport-metas", { metas: metas() }), 5000);

  // ---- keyboard detection: baseline per orientation --------------------------
  const base = {};
  let kbd = false;
  const checkKbd = (why) => {
    if (!vv) return;
    // The device's orientation, not the window's shape: Chrome on iOS shrinks
    // the window itself when the keyboard opens (430x742 -> 430x401).
    const so = screen.orientation && screen.orientation.type;
    const o = so ? so.split("-")[0] : (Math.abs(window.orientation || 0) === 90 ? "landscape" : "portrait");
    base[o] = Math.max(base[o] || 0, vv.height);
    const now = vv.height < 0.85 * base[o];
    if (now !== kbd) {
      kbd = now;
      log(now ? "KEYBOARD-SHOWN" : "KEYBOARD-HIDDEN", { why, base: Math.round(base[o]), inner: [innerWidth, innerHeight], ...vvInfo(), active: name(document.activeElement) });
      flush();
    }
  };
  if (vv) {
    vv.addEventListener("resize", () => { log("vv-resize", { ...vvInfo(), inner: [innerWidth, innerHeight] }); checkKbd("vv-resize"); });
    vv.addEventListener("scroll", () => log("vv-scroll", vvInfo()));
  }
  addEventListener("resize", () => { log("win-resize", { inner: [innerWidth, innerHeight], ...vvInfo() }); checkKbd("win-resize"); });

  // ---- events, focus and calls -----------------------------------------------
  let canvasEnds = 0;
  let touchSeq = 0;
  const pt = (e) => { const t = (e.changedTouches && e.changedTouches[0]) || (e.touches && e.touches[0]); return t ? [Math.round(t.clientX), Math.round(t.clientY)] : null; };
  for (const type of ["touchstart", "touchend", "touchcancel", "mousedown", "mouseup", "click"]) {
    addEventListener(type, (e) => {
      if (type === "touchstart") touchSeq++;
      log(type, { seq: touchSeq, target: name(e.target), at: pt(e) || (e.clientX !== undefined ? [e.clientX, e.clientY] : null), active: name(document.activeElement) });
    }, { capture: true, passive: true });
    addEventListener(type, (e) => {
      log(type + ":after", { prevented: e.defaultPrevented, active: name(document.activeElement) });
    }, { passive: true }); // bubble on window: after every page handler
  }
  for (const type of ["focusin", "focusout"]) {
    addEventListener(type, (e) => log(type, { target: name(e.target), related: name(e.relatedTarget) }), true);
  }
  const stack = () => (new Error().stack || "").split("\n").slice(2, 5).map((s) => s.trim().slice(0, 90));
  for (const m of ["focus", "blur"]) {
    const orig = HTMLElement.prototype[m];
    HTMLElement.prototype[m] = function (...a) {
      log("call-" + m, { on: name(this), from: stack() });
      return orig.apply(this, a);
    };
  }

  // ---- Ebitengine's textarea: value, selection, style, flags -----------------
  const valueDesc = Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, "value");
  const ssr = HTMLTextAreaElement.prototype.setSelectionRange;
  let swallowUntil = 0;
  const hookTextarea = (ta) => {
    if (ta._diagHooked) return;
    ta._diagHooked = true;
    Object.defineProperty(ta, "value", {
      configurable: true,
      get() { return valueDesc.get.call(this); },
      set(v) {
        const swallowed = performance.now() < swallowUntil;
        log("ebi-value-set", { v: String(v).slice(0, 20), swallowed, active: name(document.activeElement) });
        if (!swallowed) valueDesc.set.call(this, v);
      },
    });
    ta.setSelectionRange = function (...a) {
      const swallowed = performance.now() < swallowUntil;
      log("ebi-setSelectionRange", { a, swallowed });
      if (!swallowed) return ssr.apply(this, a);
    };
    new MutationObserver(() => {
      const s = ta.style;
      log("ebi-style", { left: s.left, top: s.top, w: s.width, h: s.height, fs: s.fontSize });
    }).observe(ta, { attributes: true, attributeFilter: ["style"] });
    log("ebi-textarea-found", { attrs: [...ta.attributes].map((a) => a.name + "=" + a.value.slice(0, 60)) });
  };
  new MutationObserver(() => {
    const ta = document.getElementById(EBI);
    if (ta) hookTextarea(ta);
  }).observe(document.documentElement, { childList: true, subtree: true });

  let lastFlags = "";
  let lastActive = "";
  const poll = () => {
    const w = window;
    const flags = `x=${w._ebitengine_textinput_x} y=${w._ebitengine_textinput_y} ready=${w._ebitengine_textinput_ready}`;
    if (flags !== lastFlags) { lastFlags = flags; log("ebi-flags", { flags }); }
    const a = name(document.activeElement);
    if (a !== lastActive) { lastActive = a; log("active", { a }); }
    checkKbd("poll");
    requestAnimationFrame(poll);
  };
  requestAnimationFrame(poll);

  // After any focus of a textarea we care about, report its box a frame later.
  addEventListener("focusin", (e) => {
    const el = e.target;
    if (!(el.id === EBI || el.id === "diag-clone")) return;
    requestAnimationFrame(() => {
      const r = el.getBoundingClientRect();
      const cs = getComputedStyle(el);
      log("box", { id: el.id, rect: [r.left, r.top, r.width, r.height].map(Math.round), fs: cs.fontSize, pos: cs.position, active: name(document.activeElement), ...vvInfo() });
    });
    setTimeout(() => log("after-1s", { id: el.id, kbd, active: name(document.activeElement), ...vvInfo(), inner: [innerWidth, innerHeight] }), 1000);
  }, true);

  // ---- experiments -------------------------------------------------------------
  let ebiFocusedInThisEvent = false;
  addEventListener("touchstart", () => { ebiFocusedInThisEvent = false; }, { capture: true, passive: true });
  addEventListener("focusin", (e) => { if (e.target.id === EBI) ebiFocusedInThisEvent = true; }, true);

  if (X === "nomove" || X === "away" || X === "norewrite") {
    // Bubble on window: runs after Ebitengine's body handler, in the same task.
    addEventListener("touchend", () => {
      if (!ebiFocusedInThisEvent) return;
      ebiFocusedInThisEvent = false;
      const ta = document.getElementById(EBI);
      if (X === "nomove") { ta.style.left = "0px"; ta.style.top = "0px"; }
      if (X === "away") { ta.style.left = AT[0] + "px"; ta.style.top = AT[1] + "px"; }
      if (X === "norewrite") swallowUntil = performance.now() + 1000;
      log("experiment-applied", { x: X, left: ta.style.left, top: ta.style.top });
      flush();
    }, { passive: true });
  }

  addEventListener("DOMContentLoaded", () => {
    // Status line (no pointer events).
    const line = document.createElement("div");
    line.style.cssText = "position:fixed;left:0;right:0;bottom:0;z-index:99999;pointer-events:none;" +
      "font:12px/1.3 ui-monospace,monospace;padding:2px 6px;background:rgba(0,0,0,.75);color:#7f7;white-space:nowrap;overflow:hidden";
    document.body.appendChild(line);
    setInterval(() => {
      line.style.color = kbd ? "#000" : "#7f7";
      line.style.background = kbd ? "#7f7" : "rgba(0,0,0,.75)";
      line.textContent = `${X}${X === "clone" || X === "away" ? "@" + AT : ""} · kbd=${kbd ? "YES" : "no"} · ${name(document.activeElement)} · taps ${touchSeq} · ${recent.join(" ")}`;
    }, 200);

    if (X === "native") {
      // A plain visible input, outside the TV, that the finger taps itself:
      // the browser focuses it natively (no page code calls focus()).
      const box = document.createElement("input");
      box.id = "diag-native";
      box.placeholder = "tap";
      box.style.cssText = "position:fixed;right:6px;bottom:26px;width:64px;height:32px;font-size:16px;z-index:99999;" +
        "background:#333;color:#fff;border:2px solid #ffb43c;border-radius:4px;padding:0 4px";
      document.body.appendChild(box);
    }

    if (X === "clone") {
      // Registered before Go starts, so it runs before Ebitengine's body handler.
      document.body.addEventListener("touchend", (e) => {
        if (e.target.tagName !== "CANVAS") return;
        canvasEnds++;
        if (canvasEnds !== TAP) { log("clone-wait", { canvasEnds, tap: TAP }); return; }
        const src = document.getElementById(EBI);
        let c;
        if (src) {
          c = src.cloneNode(false);
          c.removeAttribute("style");
          c.style.cssText = src.style.cssText;
        } else {
          c = document.createElement("textarea");
          c.style.cssText = "position:absolute;left:0;top:0;opacity:0;resize:none;pointer-events:none;overflow:hidden;width:1px;height:1px";
        }
        c.id = "diag-clone";
        c.style.left = (MOVE ? 0 : AT[0]) + "px";
        c.style.top = (MOVE ? 0 : AT[1]) + "px";
        document.body.appendChild(c);
        log("clone-made", { fromEbi: !!src, style: c.style.cssText, ebiStyle: src ? src.style.cssText : null });
        c.value = "";
        c.focus();
        if (MOVE) { c.style.left = AT[0] + "px"; c.style.top = AT[1] + "px"; }
        log("clone-focused", { active: name(document.activeElement), left: c.style.left, top: c.style.top });
        flush();
      });
    }
  });
})();
