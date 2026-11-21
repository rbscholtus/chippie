# Investigation: the iPhone keyboard doesn't open for the search field

Status: **resolved for the app**, 2026-11-22 (round 6, §11; checked on an
iPhone and an Android tablet, §11.6). The failures of rounds 1–5 were
browser state on the phone, not Chippie or Ebitengine; what that state was
is still unknown (§11.4). Written for a fresh look, possibly by another
person or model, so it repeats context that is elsewhere in the repository.

Facts are marked as **observed** (seen on the phone or in a log) or
**inferred** (our reading of them). Confidence is stated where it matters.

## 0. Outcome (read this first)

- **Nothing in Chippie's or Ebitengine's code path stops the keyboard.** In
  round 6 the phone ran the same flow as in the failing round 5: focus in
  the tap's `touchend`, the textarea moved to 40,87, its value rewritten a
  frame later. The keyboard opened on the **first tap** in Safari and Chrome,
  in every build that focuses the field on the press (this branch, Chippie
  PR #27, the prototype).
- **Ebitengine didn't change.** It has been v2.10.6 since the first commit,
  with the same `go.sum` hash as the prototype.
- **The failures were tied to the browser's state** for the old address
  (`10.17.2.116:8081`). The same Safari, never restarted, on the same iOS,
  worked at once on the new address (`192.168.100.148`, another location).
  Earlier, "a Safari restart" had also helped (§4.1).
- **Rounds 4–5's hypotheses are refuted:** the move to the caret, the
  position under the finger, the value rewrite and Ebitengine's textarea
  itself all show the keyboard (§11.2).
- **Chrome on iOS has a real trap:** its keyboard-down key hides the
  keyboard but keeps the focus on the textarea. After that, Chrome ignores
  every `focus()` from code, even during a tap, until the page reloads or a
  finger taps a real text box (observed, §11.5). The branch's re-arm in
  `press()` couldn't get out of it (it only focuses from code). Its keyboard
  check never saw Chrome's keyboard either (§7.3), so every tap on the field
  closed and reopened the keyboard.
- **The fix:** the re-arm is gone. While the keyboard stays hidden though
  the textarea has the focus, an invisible real `<input>`, the keyboard box,
  sits over the field. The next tap focuses it natively, and it hands the
  focus to the textarea (DESIGN §13). The keyboard check uses a
  per-orientation baseline (`keyboardWatch`).
- **Chippie PR #27's focus-on-press is right:** the build before it opens
  the keyboard only on the second tap.

## 1. The problem

On an iPhone 16 (iOS, Safari and Chrome; Chrome on iOS uses WebKit too),
tapping Chippie's library search field gives it the focus (amber border,
blinking caret), but **the on-screen keyboard never appears**. On desktop
browsers the field works. Earlier in the same week the keyboard did appear
on the iPhone sometimes (often only after a second tap, or after restarting
Safari); since the changes on branch `fix-ios-keyboard` it has not appeared
at all in the real search field, in either browser.

## 2. How the search field works

- Chippie draws everything with Ebitengine v2.10.6 (Go → WebAssembly); the
  search field is drawn in Go (`app/textfield.go`).
- On touch screens the field uses Ebitengine's `exp/textinput`
  (`Composer`). In the browser, Ebitengine types into a **hidden textarea**
  (`id="ebitengine-textinput"`, plus a second `ebitengine-textinput-dummy`)
  created in `exp/textinput/textinput_js.go` with:
  `position:absolute; left:0; top:0; opacity:0; resize:none;
  pointer-events:none; overflow:hidden; width:1px; height:1px` and the
  attributes `autocapitalize=off spellcheck=false translate=no wrap=off`.
- iOS shows the keyboard only for a focus made inside a user gesture, so
  Ebitengine **defers** the focus (its issue #2898): `Start` (called from
  `Update`) sets `window._ebitengine_textinput_x/_y`; a JS listener on
  `<body>` for `touchend`/`mouseup`/`keyup` (installed with `eval`) then
  does `textarea.value = ''; textarea.focus();` and moves the textarea to
  `left/top = _x/_y px`, and sets `_ebitengine_textinput_ready`. On the next
  `Update`, `Start` sees `ready` and opens the session, calling
  `setSurroundingTextToTextarea`, which on touch screens always does
  `value = ""; value = <text>` and, if needed, `setSelectionRange`.
- Ebitengine's canvas listeners call `preventDefault()` on every touch
  event, and its `touchstart` listener calls `canvas.focus()` unless a
  textarea has the focus.
- Ebitengine runs a hook before every `Update`
  (`dismissVirtualKeyboardIfNeeded`): when a textarea has the DOM focus but
  no session is open and no start is pending, after two ticks it calls
  `canvas.focus()`.
- Chippie (branch `fix-ios-keyboard`, uncommitted) focuses the field when a
  finger **presses** it (`textField.press`, from `updatePicker`), so `Start`
  runs before that tap's `touchend` and the deferred focus happens in the
  same tap.

## 3. Test setup

- Build: `./scripts/build.sh` on branch `fix-ios-keyboard` (uncommitted
  changes, listed in §8), served from the Mac with `Cache-Control: no-store`
  (a small Python server on port 8081) so the phone never runs a stale
  build.
- Phone: iPhone 16, Safari and Chrome, on the same network as the Mac.
- A **diagnostic overlay** was injected into the served `web/dist/index.html`
  only (never in the repository's `web/page.html`); it is kept as
  [ios-keyboard-diag.html](ios-keyboard-diag.html) (the final version, with
  all test buttons of the last round). It is active with `?diag` and shows:
  - top line: `active=` (document.activeElement), `pending=`/`ready=`
    (Ebitengine's deferred-start flags), `vv=` visualViewport height /
    innerHeight / outerHeight / screen.height, `coarse=` (`(pointer:
    coarse)`), `ebi@left,top` (the textarea's style position);
  - a log (seconds since load) of `touchstart`/`touchend` (with target),
    `focusin`/`focusout`, and in the last round every **`focus()`/`blur()`
    call** made by code (the prototypes are wrapped) and every
    visualViewport `resize` ("viewport height").
- Keyboard detection used in the logs: in **Safari**, the keyboard is open
  when the visualViewport height drops below innerHeight (e.g. `426/775`);
  in **Chrome on iOS**, both shrink together (e.g. `401/401`), so that
  comparison does not detect it there (see §7.3).
- Emulation: Playwright (1.64) WebKit with the `iPhone 16` device preset
  and touches built with `document.createTouch`/`createTouchList` (WebKit's
  `Touch` constructor is not available), and Chromium with CDP touch events.
  Neither shows an on-screen keyboard; they only show focus behaviour.

## 4. Results, in order

### 4.1 Before the diagnostics (user reports)

1. With the build from Chippie PR #27 (focus on the tap's **release**):
   Safari needed two taps on the field, or a Safari restart; Chrome showed
   the keyboard once, then never again after the keyboard's "keyboard down"
   button. *Correction (round 6): PR #27 already focuses on the **press**
   (`8cb6641`); the build that focused on the release is the one before it
   (`cad8d09`).*
2. With branch `fix-ios-keyboard` (focus on the **press**): no keyboard in
   either browser.

### 4.2 Real search-field tap, Safari (first `?diag` screenshot)

Observed, fresh page load, ROM list open, one tap on the search field:

```
active=ebitengine-textinput pending=false ready=false
vv=775/775 coarse=true
1.64 focusin CANVAS
35.33 touchstart on CANVAS      (switching on: the list opens)
35.41 touchend on CANVAS
47.34 touchstart on CANVAS      (the search field)
47.45 touchend on CANVAS
47.45 focusout CANVAS
47.45 focusin ebitengine-textinput
```

Inferred: the deferred focus happens inside the tap's `touchend`, as iOS
requires, and the textarea **keeps** the focus; the keyboard does not open
(`775/775`). High confidence.

### 4.3 Real search-field taps, Chrome (`Chippie 2.png`, since deleted)

Observed, several taps on the field:

```
active=ebitengine-textinput pending=false ready=false vv=775/775 coarse=true
69.67 focusout CANVAS
69.67 focusin ebitengine-textinput
69.79 touchstart on CANVAS
69.80 focusout ebitengine-textinput
69.80 focusin CANVAS
69.91 touchend on CANVAS
69.91 focusout CANVAS
69.91 focusin ebitengine-textinput
71.07 touchstart on CANVAS
71.08 focusout ebitengine-textinput
71.08 focusin CANVAS
71.19 touchend on CANVAS
71.19 focusout CANVAS
71.19 focusin ebitengine-textinput
```

Inferred: the focus moves to the canvas just after `touchstart` (Chippie's
`textField.press` "re-arm", §8, because it thinks the keyboard is hidden)
and back to the textarea in `touchend`, every tap. No keyboard. Same
outcome as Safari.

### 4.4 Round 1: four buttons, touch **not** cancelled (`Chippie 3, 8–11.png`, deleted)

Each button focused an element in its own `touchend` handler, without
`preventDefault`:

| Button | Focuses | Keyboard (Safari / Chrome) | Log |
|---|---|---|---|
| 1 | a visible `<input>` | no / no* | `focusin diag-visible`, `focusout diag-visible` ~10 ms later |
| 2 | the visible input, **with** `preventDefault` on touchstart+touchend | **yes / yes** | focus kept |
| 3 | an invisible textarea styled like Ebitengine's | no / once | `focusin`, `focusout` 20–170 ms later |
| 4 | Ebitengine's textarea | no / no | `focusin`, `focusout` ~10 ms later |

\* Chrome logged `test 1 +1s: vv=401/401` and `test 3 +1s: vv=401/401` once:
the keyboard may have started opening before the focus was lost.

Inferred: without `preventDefault`, iOS's own handling of the tap on the
button takes the focus away right after the handler; 1, 3 and 4 were
therefore **invalid**. Only 2 counts: a focus in a `touchend` whose touch
is cancelled opens the keyboard.

### 4.5 Round 2: every button cancels the touch (`Chippie2/5/6/7/8.png`)

| Button | Focuses | Safari | Chrome |
|---|---|---|---|
| 2 | visible input | **yes** | yes |
| 5 | invisible 1 px textarea styled like Ebitengine's (`diag-hidden`) | **yes** (`test 5 +1s: vv=426/775 active=diag-hidden`) | yes |
| 6 | Ebitengine's real textarea | no | no (brief flicker) |
| 7 | invisible textarea, 120×30, touchable (`diag-sized`) | **yes** | yes |
| 8 | Ebitengine's real textarea, enlarged to 120×30 and touchable | no | no (brief flicker) |

Observed log for 6 (8 is the same):

```
390.42 focusin ebitengine-textinput
390.42 test 6: focused ebitengine-textinput
390.43 focusout ebitengine-textinput
390.43 focusin CANVAS
391.45 test 6 +1s: vv=775/775/932 active=CANVAS
```

Inferred: **invisibility is not the cause** (5 and 7 work). 6 and 8 are
**invalid**: Ebitengine's hook moved the focus to the canvas within ~10 ms
because no session was open (§2). The Chrome flicker is the keyboard
starting and being dismissed.

### 4.6 Round 3: exact clones of Ebitengine's textarea

`cloneNode(false)` of `ebitengine-textinput` (same attributes, inline style,
position) with id `diag-clone`, so Ebitengine's code leaves it alone.

| Button | How | Safari | Chrome |
|---|---|---|---|
| 9 | clone focused in the button's own `touchend` (touch cancelled) | **yes** | yes |
| 10 | clone focused from a **`<body>` `touchend` listener** during the bubble, as Ebitengine does | **yes** | yes |

The clone's attributes were logged as `id autocapitalize spellcheck translate
wrap style`, position `0px,0px` (the user had not tapped the search field in
this round, so Ebitengine had not moved its textarea).

Inferred: the element's attributes/style and focusing from a body listener
are **not the cause**.

Chrome, observed: after closing the keyboard with **Chrome's own dismiss
key** (not the test's `x`, which calls `blur()`), no button opened the
keyboard again. Inferred (medium confidence): Chrome's dismiss hides the
keyboard but leaves the focus on the element, and a `focus()` on the
already-focused element does nothing.

### 4.7 Round 4: the tap on the canvas (`Chippie11-12.png`)

The button only arms; the next tap on the **canvas** (on the list's
"PLAY (309)" heading, which does nothing in the app) focuses the clone in a
body `touchend` listener, exactly where Ebitengine focuses its textarea.
Ebitengine's canvas handlers run as usual (preventDefault, and
`canvas.focus()` on touchstart).

| Button | Variation | Safari | Chrome |
|---|---|---|---|
| 11 | as is | **yes** | yes |
| 12 | `canvas.focus()` made a no-op for that touch | **yes** | yes |

Observed (12):

```
311.90 touchstart on CANVAS
311.90 test 12: touchstart on canvas, active BODY
311.90 test 12: canvas.focus() skipped
311.99 touchend on CANVAS
311.99 clone at 0px,0px attrs id autocapitalize spellcheck translate wrap style
311.99 focusin diag-clone
311.99 test 12 (canvas tap): focused diag-clone
313.00 test 12 +1s: vv=426/775 active=diag-clone
```

Inferred: **a tap on the canvas, with Ebitengine's handlers, does allow a
keyboard**, and the `canvas.focus()` at touchstart doesn't prevent it.

### 4.8 Round 5: Ebitengine's post-focus steps (`Chippie.png`, latest)

Ebitengine does two things to its textarea that the clone hadn't: it
**moves** it right after focusing it (to the caret, `left/top`), and a frame
later **rewrites** its value and selection. The canvas-tap test was repeated
with those added:

| Button | After focusing the clone on the canvas tap | Result |
|---|---|---|
| 13 | moved to Ebitengine's recorded position, **40px,87px** | **no keyboard** (Safari) |
| 14 | 20 ms later `value=""; value=""; setSelectionRange(0,0)` | no keyboard (Safari)† |
| 15 | both | no keyboard (Safari)† |

† Run **after** a failed real search-field tap and after 13; see below.

*Correction (round 6): `clone()` copies the inline style, so after the real
search-field tap every clone (13, 14, 15 and later) started at 40,87; 14 was
not "rewrite only". Round 6 shows that 40,87 works too (§11.2), so these
failures were the same browser state as the real flow's.*

The same session's real search-field tap, with focus-call logging:

```
active=ebitengine-textinput pending=false ready=false
vv=775/775/932/932 coarse=true ebi@40px,87px
0.00 viewport height 775
18.13 call focus() on CANVAS
18.13 focusin CANVAS
29.95 touchstart on CANVAS          (switching on)
29.95 call focus() on CANVAS
30.18 touchend on CANVAS
55.31 touchstart on CANVAS          (the search field)
55.31 call focus() on CANVAS
55.43 touchend on CANVAS
55.43 call focus() on ebitengine-textinput
55.43 focusout CANVAS
55.43 focusin ebitengine-textinput
55.43 ebi box at 40px,87px (rect 40,87 in 430x775)
```

Observed: **no `viewport height` line** after the tap, so the keyboard did
not even start to open. Nothing else calls `focus()`/`blur()`. The user then
"played around more but got no keyboard anymore, ever" (with any button).

At the start of round 5, the first tap meant to switch the TV on opened a
keyboard and not the list: it had landed on the overlay's visible test
input, which sat over the top of the TV. That was the test input, not the
app; the overlay was then changed to put the input at the right edge.

## 5. What is established

High confidence:

- The deferred focus fires inside the tap's `touchend` and the textarea keeps
  the focus; Chippie's and Ebitengine's timing do what iOS documents.
- iOS shows the keyboard for a `focus()` in a cancelled (`preventDefault`)
  `touchend`, on the canvas or on a button, for a visible input, an
  invisible 1×1 textarea, an invisible full-size textarea, and an exact
  clone of Ebitengine's textarea at `0,0`, from the target's handler or from
  a `<body>` listener, with or without Ebitengine's `canvas.focus()` at
  touchstart.
- In the real flow no `focus()`/`blur()` calls happen besides Ebitengine's
  `canvas.focus()` (touchstart) and `textarea.focus()` (touchend).
- The keyboard never starts to open in the real flow (no viewport resize).
- Chrome on iOS behaves like Safari, except its dismiss key leaves the focus
  on the element.

## 6. Open questions and hypotheses

1. **What differs between the real flow and the clone tests 11/12?** Known
   differences: (a) the real textarea is moved to `40px,87px` right after
   `focus()`; (b) a frame later its value is rewritten (forced on touch
   screens) and the selection set; (c) the real textarea has Ebitengine's
   listeners (`compositionstart/end`, `focusout`, `keydown`, `change`);
   (d) in the real flow Chippie's `Update` runs `Start`/`deferStart` during
   the tap (the clone tests' tap on "PLAY" does nothing in the app);
   (e) the tap's position (the field vs the heading).
2. **Is the failure sticky?** Round 5 suggests that once a programmatic focus
   failed to show the keyboard, later ones (13, 14, 15, and "anything")
   failed too until the page was reloaded. Rounds 2–4 all worked, and in
   them the search field had not been tapped first. The very first `?diag`
   screenshot (§4.2) was a fresh load and failed on the first tap, so the
   search-field flow fails even without an earlier failure. Hypothesis
   (medium confidence): something in the real flow trips an iOS rule after
   which iOS stops honouring programmatic focus on this page.
   *Round 6: the failure was sticky, but not because of the real flow, which
   works on a fresh address. The state belonged to the browser and the old
   address (§11.4).*
3. **Test 13 is ambiguous**: it ran after a failed real tap, so it may have
   failed because of (2) rather than because of the move. It should be
   repeated on a freshly loaded page **without** tapping the search field
   first (enter the position by hand, e.g. `40px,87px`, which is where
   Ebitengine puts it on an iPhone 16 in portrait, also measured in WebKit
   emulation).
4. Ebitengine's `_x/_y` are "client position in native pixels"
   (`caretBoundsInClientNativePixels`); measured as `40px,87px` CSS on the
   phone (dpr 3), so on screen. Not suspected any more.

Suggested next experiments (each on a **freshly loaded page**):

- 13 again without a prior search-field tap (position typed in); then 14
  alone; then 15.
- The real flow with Ebitengine's textarea **not moved** (patch
  `style.left/top` setters to no-ops for that element, or wrap the body
  handler) — if the keyboard appears, the move is the cause.
- The real flow with `setSurroundingTextToTextarea`'s value writes disabled
  (wrap the `value` setter on that element).
- Safari's Web Inspector (Develop menu on the Mac, phone connected) to watch
  the real flow live and try the above from the console without new builds.

## 7. Related findings

### 7.1 Upstream Ebitengine

Merged on `main` 2026-10-28, **not** in v2.10.6 nor in v2.11.0-alpha
(2026-09-05):

- #3974 "exp/textinput: report browser focus loss as user-ended input":
  the browser `focusout` now calls `endByUser`, so `Composer.OnEndByUser`
  fires when the user closes the keyboard (Chippie works around this, §8).
- #3972 "exp/textinput: release abandoned deferred browser focus": clears an
  unconsumed `ready` flag so focus returns to the canvas.
- #3745 (2026-09-22): the deferred start uses the caret top instead of the
  bottom (one line height).
- #4068 (2026-11-17): composition state of input events.

None of them mentions the keyboard not appearing. Their PR descriptions say
mobile Safari / physical devices were not tested.

### 7.2 Desktop-only behaviour

The deferred focus happens on the next `mouseup`/`keyup` too; desktop
browsers don't need it and work.

### 7.3 Chippie's keyboard-visibility check

`platform.KeyboardShown` (branch `fix-ios-keyboard`) compares
`visualViewport.height` with `innerHeight × 0.85`. Correct in Safari
(`426/775`), **wrong in Chrome on iOS** (`401/401` with the keyboard open).
A per-orientation baseline (the largest visualViewport height seen) would
work in both. *Done in round 6 (§11.3): `platform.VisibleHeight` and
`keyboardWatch`. The orientation must come from `screen.orientation`: with
Chrome's keyboard open the window is 430×401, wider than tall.*

## 8. State of the code

Branch `fix-ios-keyboard` (from `main` at `276d074`, Chippie PR #27 merged),
**uncommitted**. As of round 6 it passes `golangci-lint` (native and
wasm), `go test ./...` and the Playwright suite (37 passed, 20 skipped).
Changes:

- `app/textfield.go`:
  - `blur()` ends the session at once, so the keyboard closes when a game
    starts.
  - `keyboardClosed()` unfocuses the field when the textarea loses the DOM
    focus after having had it (Safari's done button). This stops the
    keyboard coming back by itself.
  - `watchKeyboard()` (round 6) tracks the keyboard with `keyboardWatch`
    and shows the keyboard box while it stays hidden though the textarea
    has the focus.
  - Browser state comes through replaceable function fields; unit tests in
    `app/textfield_test.go`.
  - *(Rounds 1–5 had a `press()` re-arm, session end + `canvas.focus()`,
    instead; removed in round 6, §11.5.)*
- `platform/textinput_js.go` / `_other.go`:
  - `TextInputFocused`: checks Ebitengine's textarea ids (internals of
    v2.10.6).
  - `VisibleHeight` (round 6, replaces `KeyboardShown`, §7.3).
  - `ShowKeyboardBox`/`HideKeyboardBox` (round 6, replace `FocusCanvas`).
- `app/game.go`: `Update` blurs the field whenever the picker isn't
  showing.
- `app/game_flow.go`:
  - `updatePicker` focuses the field on a press on it.
  - `relayout` gives the field its CSS scale.
  - `switchOnAt`: a tap on "< ROM list"/"Settings >" while the TV is off
    opens that panel.
- `app/ui.go`: the status taps are drawn while the TV is off; "Tap to switch
  on" on touch screens.
- `platform/filepicker_js.go`: "Load a ROM from this device" hidden on
  `(pointer: coarse)` (iOS opens the file chooser only inside the tap; the
  app acts a frame later).
- `e2e/tests/touch.spec.ts`: four phone tests. The keyboard closes when a
  game starts; a closed keyboard doesn't come back on the next tap; the
  keyboard box; a Settings tap while off. The first two fail on the old
  code.
- `docs/DESIGN.md` §11, §13, §14 and `README.md` describe the above.

## 9. Options considered

*Round 6: the hidden-textarea approach works (§0). A narrow form of option 1
is used only while the keyboard is stuck hidden (Chrome's keyboard-down):
the keyboard box, §11.5.*

1. **Real HTML input on touch screens** (recommended in the session): place
   a visible DOM `<input>` exactly over the drawn search field on touch
   screens; the finger taps a real input, which iOS always honours (button 2
   worked every time in both browsers). Text goes to Go on every `input`
   event; the element is removed when the drawer closes. Removes the need
   for the textarea-id check, the re-arm and keyboard detection. Cost: one
   DOM element over the canvas, against the "everything drawn in Go" rule;
   needs a DESIGN note, positioning on relayout, and styling to match.
2. **An on-screen keyboard drawn in Go** in the drawer, like the remote.
   Most work; no autocorrect, other scripts or dictation.
3. **Find the cause and report it to Ebitengine** (Web Inspector, §6),
   possibly after trying Ebitengine `main` with #3972/#3974. Fixes it for
   everyone if it works; no guarantee or timeline.

## 10. Files

- [ios-keyboard-diag.html](ios-keyboard-diag.html): the final diagnostic
  script (paste before `</body>` of `web/dist/index.html` after a build;
  activate with `?diag`). Its buttons: 13/14/15 arm a canvas-tap test (then
  tap the "PLAY" heading), `x` blurs. Earlier rounds' buttons are described
  in §4 and can be restored from that description.
- Screenshots in the repository root (untracked): `Chippie.png` (round 5 and
  the real flow), `Chippie11-12.png` (round 4), `Chippie2/5/6/7/8.png`
  (round 2). The first `?diag` screenshot and the Chrome screenshots of
  round 1 were overwritten or deleted; their logs are copied above.
- Round 6: [ios-keyboard-serve.py](ios-keyboard-serve.py) (the server, with
  its usage in the docstring) and
  [ios-keyboard-diag2.js](ios-keyboard-diag2.js) (the diagnostics it
  injects; the experiments are described at its top).

## 11. Round 6 (2026-11-22): the same flow works

### 11.1 Method

Every earlier round was read off screenshots, with test buttons over the
TV. Round 6 changed the method:

- **A server** (`ios-keyboard-serve.py`) serves four builds side by side,
  each on its own port with `no-store`:
  - 8081: the branch;
  - 8082: Chippie PR #27 (`276d074`);
  - 8083: the build before it (`cad8d09`, focus on release);
  - 8084: the prototype.
- **The phone sends its log to the server** (`POST /log`, JSON lines), so
  nothing is copied by hand.
- **The diagnostics** (`ios-keyboard-diag2.js`) show only a status line and
  add no buttons. They log touches, focus changes, every `focus()`/`blur()`
  call with its caller, Ebitengine's `_x/_y/ready` flags, and the
  textarea's `value` writes and style changes. The keyboard counts as open
  when the visible height drops below 85 % of the tallest seen in the
  device orientation.
- **Each experiment is a fresh page load**, picked from a menu
  (`/menu`) by URL.
- **A Playwright smoke test** checked that each experiment armed and logged
  before the phone was used.

Phone: iPhone 16 Plus (430×932 CSS pixels, dpr 3), Safari 27.2 and Chrome on
iOS 158. Safari had not been restarted since rounds 1–5, and iOS hadn't
been updated in between. The only change: the Mac was at another location,
so the page came from `192.168.100.148` instead of `10.17.2.116`.

### 11.2 Safari: every variant opens the keyboard on the first tap

| # | Build, experiment | Result (observed, log) |
|---|---|---|
| 1 | prototype, field B | first tap |
| 2 | branch | first tap; the second tap keeps it open; after Done, a tap opens it again |
| 3 | PR #27 | as 2 |
| 4 | before PR #27 (focus on release) | second tap only (by design) |
| 5 | copy of the textarea at 0,0, tap on "PLAY" | opens |
| 6 | copy at 40,87, tap on "PLAY" | opens |
| 7 | real flow, Ebitengine's move undone | opens |
| 8 | real flow, textarea moved to 200,400 | opens |
| 9 | real flow, value rewrite swallowed | opens |
| 1b | prototype page without doctype or viewport meta | opens |

The log of experiment 2 shows the same sequence as round 5's failing tap
(§4.8): `touchend`, `focus()` on `ebitengine-textinput` from Ebitengine's
handler, style `left:40px; top:87px`, two value writes. The visible height
then dropped from 775 to 426, about 170 ms after the touchend.

### 11.3 Chrome on iOS: the keyboard check, and the re-arm

- First tap: opens in every build that focuses on the press (experiments 1,
  2, 3, 5–9); before PR #27, on the second tap.
- **Second tap on the field while the keyboard is open:**
  - **branch:** the keyboard closed and reopened (observed). The log shows
    the re-arm's `canvas.focus()` from Go at the press, the window growing
    to 720–730, then the touchend's focus and the window shrinking to 401
    again.
  - **PR #27:** it stays open.
- **Chrome's keyboard-down key** (PR #27): the window grows back to 742, and
  **the textarea keeps the DOM focus** (no `focusout`). Later taps on the
  field do nothing: no `focus()` call, no keyboard (observed). So the
  re-arm is needed. Its keyboard check (§7.3) was wrong because Chrome
  shrinks the window itself: 430×742 to 430×401.
- **The keyboard check:** `platform.VisibleHeight` (the visual viewport's
  height and `screen.orientation`), and `keyboardWatch` in
  `app/textfield.go`: shown while below 85 % of the tallest height seen in
  that orientation. The field samples it every frame while focused (DESIGN
  §13). Unit tests use the measured Safari and Chrome heights. On the phone
  it worked: a second tap with the keyboard open left it alone.
- **But the re-arm didn't help after keyboard-down** (observed with the
  fixed check): at the next tap on the field it moved the focus to the
  canvas and the touchend focused the textarea again, yet no keyboard
  appeared, tap after tap. Tapping the ROM list first (the focus fully on
  the canvas), then the field: still none. See §11.5.

### 11.4 Open: what was the browser state on the old address?

Rounds 1–5 failed on `10.17.2.116:8081`, including on a freshly reloaded
page. Round 6 worked on `192.168.100.148` in the same Safari, never
restarted. "A Safari restart" had helped in round 0 too. So the state
survived reloads and belonged to that address or its tab. Candidates
(unverified):

- the WebContent process kept for that site (a new site gets a new one;
  a restart ends them all);
- the site's stored data (Chippie's settings and ROMs in IndexedDB and
  localStorage);
- per-site Safari settings.

If it happens again, before anything else, try these in order and note
which one clears it:

1. a new tab with the same address;
2. a private tab;
3. Settings → Apps → Safari → Advanced → Website Data: delete that site's
   data;
4. a restart of Safari.

Run the same experiments (`ios-keyboard-serve.py`) on that address to see
whether the flow is unchanged.

### 11.5 Chrome on iOS after keyboard-down: only a reload or a real tap

Experiment 10: `?x=native` adds a small visible `<input>` at the bottom
right that the finger taps itself (no page code focuses it). Observed, in
Chrome 158 on iOS 27.2:

1. Tap the field: the keyboard opens. Keyboard-down: it hides, and the
   textarea keeps the DOM focus.
2. Tap the field: no keyboard (the same, however often).
3. **A reload** clears it: switch on, tap the field, and the keyboard opens.
4. **A finger tapping the visible input** opens the keyboard. After that,
   with the keyboard open, a tap on the field closes it (Ebitengine's
   `canvas.focus()` at touchstart) and the touchend's focus opens it again.
   So the native focus cleared the state.
5. Each later keyboard-down makes it stuck again.

So the fix is the **keyboard box** (DESIGN §13). While the keyboard stays
hidden for 30 frames though the textarea has the focus, an invisible real
`<input>` (`#chippie-keyboard-box`) covers the drawn field. The next tap
lands on it; its `focus` handler focuses Ebitengine's textarea, and the
field hides the box once the keyboard shows. Headless Chromium never shows
a keyboard, so the browser test "a hidden keyboard puts a real text box
over the search field" exercises it. The box may also get past the Safari
state of §11.4, if that was the same trap. It covers the case where the
keyboard never opened, not only Chrome's keyboard-down.

### 11.6 The keyboard box on the iPhone and on an Android tablet

Experiment 2 with the keyboard box build, observed and confirmed in the log:

- **iPhone, Chrome 158:** after each keyboard-down, the next tap on the
  field landed on `chippie-keyboard-box`. The focus passed to Ebitengine's
  textarea and the keyboard opened; typing went on from the text already
  there.
- **iPhone, Safari 27.2:** works; the box is never needed (Done gives the
  focus up).
- **Samsung Galaxy Tab, Android 10:**
  - **Chrome 157:** upright the visible height goes 1150 → 776 with the
    keyboard; sideways 704 → 322. Hiding the keyboard with Android's back
    gesture keeps the focus on the textarea, as Chrome's keyboard-down does
    on iOS; the box brought the keyboard back.
  - **Samsung Internet 31:** the same; it reports a desktop user agent
    (`X11; Linux x86_64`) but `(pointer: coarse)`, so the field uses the
    input method. Its sideways height went 664 → 330 with the keyboard.

The user reported every step working in both tablet browsers and both
orientations: first tap, second tap, hide and tap again, typing, and the
keyboard closing when a game starts. The log has Samsung Internet sideways
only.
