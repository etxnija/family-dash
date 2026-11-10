---
type: component
tags: [eink, prime, epaper, render, adapter, hexagonal, font, panel, port, pngpanel, epd, rotation, build-constraints, cable, refresh-loop, clock, partial, eink-loop, flags, sleep]
summary: eink/ e-paper adapter — Panel port (Full/Partial/Sleep/Close) with three adapters (pngpanel PNG writer, linux/arm Waveshare 7.5" V2 epd driver, non-linux/arm epd stub), pure rotate.go (-cable left/right mapping, Rotate/RotateRect), driver call order and rotate_ms/refresh_ms logs, Partial primed with the last shown frame (primeBeforePartial: InitPart, DisplayPart previous, DisplayPart new); eink.Loop refresh loop behind -eink-loop (fetch every 30m/new day with 5m retry, full/partial/none by pixel diff around ClockRect, Sleep then single Close on SIGINT/SIGTERM, -eink-fetch-every/-eink-full-every/-eink-sleep; priming fixed the grainy partial after ~60 s Sleep in a Nov 2026 spike, not yet verified in the real binary on the panel); Stockholm view-model, 1-bit portrait 480x800 renderer (Go Mono, single column, clock/weather band, two-line wrapped entries, content-height sections trimmed by priority, ClockRect top/bottom on multiples of 8), plus verified driver facts and Nov 2026 spike timings.
links: [docs/arch/adr/0002-dual-display-adapters-over-shared-interface.md, docs/arch/adr/0003-eink-panel-port-and-refresh-policy.md, docs/arch/adr/0004-portrait-frame-rotated-by-panel-adapter.md, docs/raspberry-pi-setup.md, dashboard-page.md, composition-root.md, eink-view.md]
updated: 2026-11-10 (Partial priming documented with the Nov 2026 spike findings and timings)
generated: { by: worker, at: 2026-11-10T00:00:00Z }
verified: [{ by: "human", at: 2026-11-10 }, { by: "human", at: 2026-11-10 }, { by: "human", at: 2026-11-10 }, { by: "human", at: 2026-11-10 }, { by: "human", at: 2026-11-10 }, { by: "human", at: 2026-11-10 }]
---

# E-paper Adapter (`eink/`)

## Context & Rationale
`eink/` turns the same domain data as `web/` into a black/white
frame (`*image.Gray`) for the Waveshare 7.5" panel, drawn in portrait
480x800. The renderer never rotates; the panel adapter turns the frame to
the panel's native landscape orientation
([ADR-0004](../../docs/arch/adr/0004-portrait-frame-rotated-by-panel-adapter.md)). Per
[ADR-0002](../../docs/arch/adr/0002-dual-display-adapters-over-shared-interface.md)
it is deliberately independent of `web/`: it has its own view-model and
renderer and imports nothing from `web`. Frames are shown through the
`Panel` port (`eink/panel.go`, per
[ADR-0003](../../docs/arch/adr/0003-eink-panel-port-and-refresh-policy.md)).
There are two real adapters: `eink/pngpanel` writes the frame to a PNG
(`-eink-png`), and `eink/epd` drives the Waveshare panel on linux/arm
(`-eink-panel`). Alone, both flags are one-shot; with `-eink-loop` the
same panel is driven by `eink.Loop` (`eink/loop.go`, "Refresh loop"
below), which uses `Partial` for the clock (see
[composition-root.md](composition-root.md) and "Panel port and adapters"
below).

## Key Rules & Mechanics
- `NewView` (`eink/view.go`, details in [eink-view.md](eink-view.md))
  converts every time to `Europe/Stockholm` and builds ready-to-draw
  Swedish strings; `Render(View)` only lays them out.
- `Render` draws anti-aliased text and rules on white, then `threshold`s
  every pixel at 128 (<128 → 0, else 255), so the output is strictly 1-bit.
- One content column (X 16–464, `contentRect`). Header: date + week
  label on line 1, owners left-aligned on line 2. Below it a clock/weather
  band: `ClockRect()` on the left, a 1 px vertical rule at `weatherRuleX`,
  then the temperature and up to 5 weather lines on the `weatherRect`
  sub-image. The SKOLLUNCH rule (`lunchTop`, `sectionGap` below
  `bandBottom`) closes the band, and the vertical rule runs down to meet
  it; then
  SKOLLUNCH, IDAG, LÄXOR OCH PROV and KOMMANDE DAGAR stack in that order.
- Entries wrap to at most `maxEntryLines` (2) lines via `wrapEntry`:
  line 1 starts with the prefix, line 2 with a hanging indent of one
  space per prefix rune (Go Mono is monospace). Owners are shown as
  initials in all three entry sections (IDAG, LÄXOR OCH PROV, KOMMANDE
  DAGAR); only the header owners line spells the names out. IDAG rows
  read `<time|heldag>  <Initial>  <summary>[, <location>]` and KOMMANDE
  DAGAR rows `<weekday>  <Initial>  <summary>[, <location>], <time|heldag>`;
  `ownerInitialPrefix` gives the initial and two spaces, or nothing when
  there is no owner. LÄXOR OCH PROV rows start with the initial
  (`HomeworkGroup.Initial`). Only IDAG and KOMMANDE DAGAR have a
  variable-width first label, so only they pad it: the time/weekday
  label is right-padded with spaces (`padRight`, `maxLabelWidth`) to the
  widest label among *all* the section's rows, before trimming, so
  summaries line up in one column whichever rows end up visible; the
  padding is part of the prefix, so the hanging indent follows it. In
  LÄXOR OCH PROV the one-letter initial is the first label, so it is
  naturally aligned. The body breaks greedily at spaces; the suffix
  (homework `" (Prov)"`/day label, upcoming `", "+time`) is unbreakable
  and always ends the last line. Body text past two lines, and a single
  word wider than a whole line, are cut with `"…"` (`fitText`).
- Sections flow from `lunchTop` (`sectionsTop`), each taking its natural
  height: `sectionHeight(lines)` = `headingOffset + lines*rowHeight`, plus
  `sectionGap` before the next section's rule. An empty section takes 1
  line, for its empty text. Positions are computed per render by
  `layoutSections` (pure, in `render.go`) from `sectionEntries`
  (`lunchEntries`, `eventEntries`, `homeworkEntries`, `upcomingEntries`),
  so they move with content. The sections fit when the last baseline
  plus `bodyDescent` is ≤ `contentBottom` (20 body lines in total today).
- When the total exceeds the area down to `contentBottom`,
  `layoutSections` lowers one line of budget at a time in `trimOrder`
  (KOMMANDE DAGAR → LÄXOR OCH PROV → SKOLLUNCH → IDAG, least important
  first), taking each section down to its floor before the next is
  touched. Each budget is applied through `budgetLines`: whole entries
  while they fit in `budget-1` lines, then `"+N till"`, where N is the
  hidden entries and the line itself counts against the budget. An entry
  is never shown in part.
- The floors are `minSectionLines` 2 and `minTodayLines` 3 (IDAG), or the
  section's natural line count if it is smaller. Floors bound the budget,
  not the visible lines. IDAG's "at least 3 visible lines" holds with the
  current constants (the line room leaves IDAG ≥ 14 lines of budget), and
  `TestLayoutSections` asserts it, so it fails if the constants change.
- `ClockRect` and the weather band are fixed, and `lunchTop` is a
  constant, so the sections only ever move below it. A partial refresh of
  the clock keeps working however the sections move.
- Other over-wide text (header, weather) is cut with `"…"` (`fitText`).
  Every region is drawn onto a clipped `SubImage` (header, content
  column, clock, weather), so nothing spills outside it; the weather can
  never reach the clock rect.
- `ClockRect()` (X 16–248, Y 88–176) is the only place the clock is drawn.
  Its Y edges are multiples of 8: in portrait the panel's native X axis
  is the portrait Y axis, one byte holds 8 native pixels and `DisplayPart`
  requires byte-aligned native X edges. Its portrait X edges are free.
- Fonts are Go Mono and Go Mono Bold from `golang.org/x/image/font/gofont`
  (embedded, parsed once at init; `renderMu` serializes `Render` since
  faces aren't concurrency-safe). x/image is pinned at v0.36.0 (see the
  x/image gotcha below for the version thresholds).
- All layout numbers (sizes, baselines, column edges, limits, clock rect)
  live in `eink/layout.go`; tweak the layout there, not in `render.go`.

## Gotchas & Failure Modes
- Owners are identified by initial only in the entry sections, so two
  family members whose names start with the same letter would be
  indistinguishable there. Today's owners (Kaj, Lillian) do not clash;
  this is not handled yet.
- The panel is landscape-native (800x480). The driver adapter alone
  rotates the portrait frame by 90 degrees and maps partial-refresh rects
  (e.g. `ClockRect()`) to native coordinates (ADR-0004); the portrait
  renderer never rotates, so the PNG is the true mounted view.
- A budget is not a line count, because an entry is never cut to fill the
  gap. With two-line entries a budget of 2 shows only `"+N till"`, and
  lowering the budget by one line can drop two visible lines (budget 5
  shows two entries + `"+N till"` = 5 lines, budget 4 one entry +
  `"+N till"` = 3). The trim loop checks the actual lines, so it may
  stop with a line of room to spare.
- x/image has two version thresholds, both read from the module cache:
  - v0.37.0–v0.45.0 declare `go 1.25.0`. Any of them makes `go mod tidy`
    rewrite our `go 1.25` directive, which is why we pin v0.36.0.
  - v0.46.0 and later declare `go 1.26.0` and need Go 1.26.
  - Our directive is now `go 1.25.0` anyway: `github.com/ditrytus/epd7in5v2`
    v0.1.0 declares `go 1.25.0` in its own `go.mod`, so `go mod tidy`
    rewrote ours when the panel driver was added. This is the same Go
    1.25 language version, not a new minimum, so it is accepted. The
    v0.36.0 pin still matters for the second threshold: v0.46.0+ would
    raise it to `go 1.26.0`. x/image is still at v0.36.0.
- Glyph coverage: Go Mono has åäö, `…`, `·`, `°`, `–`, `—`
  (`TestFaces_HaveGlyphs`); any new symbol must be added to that test, or it
  renders as a missing-glyph box.
- Mono fonts are wide: at 72 px "HH:MM" is about 216 px of the
  232 px clock rect. Changing the clock font or size means resizing `ClockRect` (keeping
  Y edges on multiples of 8) and rechecking the weather width beside it.

## Panel port and adapters
- The port is `eink.Panel` (`eink/panel.go`): `Full(frame image.Image)
  error`, `Partial(frame image.Image, rect image.Rectangle) error`,
  `Sleep() error`, `Close() error`. Frames are portrait `ScreenWidth` x
  `ScreenHeight` as drawn by `Render`; the adapter does any rotation.
  `Partial` takes the whole frame plus a portrait `rect` (e.g.
  `ClockRect()`); the adapter crops and rotates it, and the panel keeps
  showing everything outside `rect` (ADR-0003).
- Callers call `Sleep` after every update and `Close` when done; the
  one-shot helper `showOnce` always calls both, even after a failure.
- There are three adapters:
  - `eink/pngpanel`: `New(path)` writes each `Full` frame to `path` as a
    PNG. It rejects a frame that is not 480x800 before creating the file
    (the error names the bounds) and removes the partial file if
    `png.Encode` fails. `Partial` writes the whole frame exactly like
    `Full` (rect ignored, shared `write` helper), so the file always
    shows the latest frame. `Sleep` and `Close` are no-ops. It imports `eink`
    only for the size constants.
  - `eink/epd` on linux/arm drives the Waveshare 7.5" V2 through
    `github.com/ditrytus/epd7in5v2` v0.1.0.
  - `eink/epd` elsewhere is a stub with the same exported API, whose
    `New`, `Full`, `Partial`, `Sleep` and `Close` all return "e-paper panel is only
    supported on linux/arm".
- Build-constraint split in `eink/epd`:
  - `rotate.go` has no build constraint. It holds the pure half (`Cable`,
    `ParseCable`, `Rotate`, `RotateRect`) and is tested on every
    platform.
  - `device.go` has no build constraint either. It holds `devicePanel`,
    the call order and error wrapping behind a small `device` interface,
    so `device_test.go` tests it on the host with a fake.
  - `epd.go` (`//go:build linux && arm`) is thin. It is the only file
    importing the driver and only passes `epd7in5v2.NewEPD()` and
    `epd7in5v2.ScreenBounds` to `openDevice`.
  - `epd_other.go` (`//go:build !(linux && arm)`) is the stub. The host
    build never links the driver or periph.
- `-cable left|right` (default `left`, parsed by `ParseCable`, exactly
  `"left"`/`"right"`) names the side of the hung portrait frame where the
  ribbon cable exits, and picks the rotation direction. `Rotate(frame,
  cable)` rejects anything but 480x800, reads relative to `Bounds().Min`,
  and returns an 800x480 `*image.Gray` thresholded at 128. For a portrait
  pixel (px, py), the native pixel is:
  - `CableLeft`: `X = py`, `Y = 479 - px`
  - `CableRight`: `X = 799 - py`, `Y = px`

  The portrait `ClockRect()` Y edges 88/176 therefore become native X
  edges 88/176 (left) or 624/712 (right), both byte-aligned
  (`TestRotate_ClockRect`).
- `RotateRect(frame, rect, cable)` validates like `Rotate`, then rejects
  an empty `rect`, one not inside 480x800 (relative to the frame's own
  origin), and one whose native X edges are not multiples of 8. The
  native rect is X `[y0, y1)`, Y `[480-x1, 480-x0)` (left) or X
  `[800-y1, 800-y0)`, Y `[x0, x1)` (right). It returns a `(0,0)`-origin
  `*image.Gray` of the native rect's size, the shape `DisplayPart`
  wants, plus the native rect. Both functions fill pixels through the
  shared `rotateInto`, so a `RotateRect` pixel always equals `Rotate`'s
  at the same point offset by the native rect's Min (`TestRotateRect`). Which name matches which physical cable side
  has not been checked on the panel yet. If the image comes out upside
  down, use the other value.
- Driver call order (`devicePanel`):
  - `New(cable)` calls `GPIOInit()` once. If it fails, `New` calls
    `Close()` and joins both errors.
  - `Full` runs `Rotate`, checks the result against `ScreenBounds`, then
    calls `Init()` before every `DisplayImage`. Every `Full` is a full
    refresh (Init 0.17 s + 3.4–3.9 s). On success it remembers `frame`
    as the shown frame.
  - `Partial` fails with `epd: partial refresh before any frame was
    shown`, without any device call, until a `Full` or `Partial` has
    succeeded. Otherwise it runs `RotateRect` on the remembered frame and
    on the new one (same `rect`, so the same native rect), checks the
    native rect lies inside `ScreenBounds`, then calls `InitPart`,
    `DisplayPart(previous, nativeRect)`, `DisplayPart(small, nativeRect)`
    and on success remembers the new frame. `Full`'s `Init` returns the
    controller from partial mode.
  - The remembered frame (`shown`) is an independent thresholded copy
    made by `copyFrame`, set only after `DisplayImage` or the last
    `DisplayPart` succeeds; any failure leaves it unchanged.
  - `primeBeforePartial` (a const in `device.go`, true, copied into
    `devicePanel.prime` by `openDevice`, no flag) is the one switch. Set
    to false, `Partial` makes a single `DisplayPart` and needs no shown
    frame; `shown` is still updated.
  - Why prime: in the Nov 2026 spike a partial after about 60 s of deep
    sleep, without priming, left the changed clock digit grainy and
    half-written, while writing the shown image first was clean (see
    "Hardware and driver facts"). Working theory, not confirmed: during a
    long deep sleep the controller loses its copy of what is on screen,
    so the partial update diffs against bad data, and the first write
    restores it. The cost is one extra partial write (about 0.5 s) per
    minute.
  - `Sleep()` follows every update and `Close()` comes last. Both are
    called by the caller (`showOnce` or `eink.Loop`), not by `Full` or
    `Partial`.
  - Errors are wrapped with the step name (`epd: gpio init`, `epd:
    rotate`, `epd: init`, `epd: display`, `epd: rotate rect`, `epd: init
    part`, `epd: display part (previous image)`, `epd: display part`,
    `epd: sleep`, `epd: close`).
- `Full` logs `rotated e-paper frame` with `rotate_ms` and `cable`, then
  `full refresh` with `refresh_ms` (Init + DisplayImage). `Partial`
  logs `rotated e-paper rect` with `rotate_ms` (both rects when
  priming), `cable` and `native_rect`, then `partial refresh` with
  `primed` and `partial_ms` (InitPart + both DisplayPart writes when
  primed, so about 1.2 s).
- The sensors only build for the host. Check `epd.go` changes and build
  the Pi binary with
  `GOOS=linux GOARCH=arm GOARM=6 CGO_ENABLED=0 go build ./... && GOOS=linux GOARCH=arm GOARM=6 go vet ./...`
  and `GOOS=linux GOARCH=arm GOARM=6 CGO_ENABLED=0 go build -o family-dashboard-pi ./cmd/family-dashboard`.
- Driver gotcha (v0.1.0): `closeGPIO` (`gpio.go:81-91` in the driver
  module) returns early only when all of `pwr`, `dc` and `rst` are nil,
  and otherwise calls `Out(gpio.Low)` on all three. `GPIOInit`
  (`gpio.go:52-58`) sets them in the order `rst`, `dc`, `pwr` and stops at
  the first failed lookup (`initPinOut`, `gpio.go:105-118`) without
  closing anything. So if `rst` is found but `dc` or `pwr` is not, the
  `Close()` in `New` calls `Out` on a nil interface and panics. Not seen
  on the Pi Zero W.

## Refresh loop (`eink/loop.go`)
- `NewLoop(panel, views, clock, settings).Run(ctx)`. `ViewSource` is
  `func(ctx, now) (View, error)`; `Clock` is `Now`/`Sleep(ctx, d)`
  (`SystemClock` is the real one, tests use a fake that advances time).
  `DefaultLoopSettings()`: `FetchEvery` 30m, `FullEvery` 1h,
  `SleepEveryUpdate` true.
- Start: load Stockholm, fetch, `Full` (reason `"start"`). A failing first
  fetch closes the panel and returns the error; a failing first `Full` is
  only logged and retried.
- Each tick sleeps until the next minute plus `tickOffset` (2 s), so the
  shown minute is the new one. A fetch is due when `FetchEvery` has passed
  or the Stockholm date changed since the last successful fetch, but not
  within `FetchRetryDelay` (5m) of a failed one; a failed fetch keeps the
  previous view.
- Each tick renders the view with `clockText(now)` and compares it with
  the shown frame (`diffPixels`, split outside/inside `ClockRect()`).
  Decision order: `full`/`"retry"` after a failed refresh, `full`/
  `"interval"` after `FullEvery`, `full`/`"content changed"` if a pixel
  outside the clock differs, `partial`/`"clock"` if one inside differs,
  else `none`. Every decision is logged as `e-paper update` with
  `refresh`, `reason`, `clock`, `changed_pixels`, `since_full`.
- The shown frame only changes on a successful refresh, so a failed
  `Partial` is redone as a `Full`. `Sleep` follows every panel call when
  `SleepEveryUpdate` (errors logged, never fatal).
- Why this order: a full refresh takes about 4 s (Init + 3.4–3.9 s) and
  flashes, a primed partial about 1.2 s (InitPart + two writes of about
  0.5 s), so the loop does the cheapest refresh that leaves the
  panel right. Partial refreshes ghost, so `FullEvery` forces a full one
  regularly. `Partial` only redraws `ClockRect()`, so any change outside
  it (new events, weather, a new day) needs a full refresh. After a
  failed refresh the panel's content is unknown, so only a full refresh
  makes it right again. If no pixel changed, nothing is sent.
- Cancellation: `Sleep` then `Close`, returns nil. `Close` is a single
  defer in `Run`, so it runs exactly once on every exit path. With
  `SleepEveryUpdate` the panel is therefore slept twice at shutdown (after
  the last update and in `shutdown`); this is known and not yet decided.
- `cmd` runs the loop under `signal.NotifyContext` (SIGINT, SIGTERM), so a
  signal cancels `ctx` and the process exits 0; a failing first fetch
  exits 1 (see [composition-root.md](composition-root.md)).
- Flags (all need `-eink-loop`, which needs exactly one of `-eink-png` /
  `-eink-panel`):
  - `-eink-loop`: run the loop instead of drawing one frame. With
    `-eink-png` the PNG is rewritten on every full or partial refresh.
  - `-eink-fetch-every` (default 30m, at least 1m): `FetchEvery`.
  - `-eink-full-every` (default 1h, at least 1m): `FullEvery`.
  - `-eink-sleep` (default true): `SleepEveryUpdate`.
- Partial after `Sleep`: verified in the Nov 2026 spike (see "Hardware
  and driver facts"). Without priming a partial after about 60 s of deep
  sleep was grainy; priming with the shown image fixed it. **Not yet
  verified on the panel:** the fix inside the real binary (`-eink-loop
  -eink-panel` with priming); a human will check it and mark it here.
  `-eink-sleep=false` stays as the fallback, since staying awake was
  also clean at 60 s.

## Hardware and driver facts for the panel adapter
Verified on a Pi Zero W with the Waveshare 7.5" V2 and
`github.com/ditrytus/epd7in5v2` v0.1.0 (see
[raspberry-pi-setup.md](../../docs/raspberry-pi-setup.md)):
- The panel is 800x480 and 1-bit only (no greyscale).
- `DisplayImage` is a full refresh after `Init` (about 4 s in total,
  timings below). About 1.5 s after `InitFast` (library figure, not
  measured).
- `InitPart` followed by `DisplayPart(img, rect)` is a partial refresh
  (about 0.7 s in total, timings below).
- `DisplayPart` takes an image the size of `rect` with its origin at (0,0),
  not the full frame. Passing the full frame shows the frame's top-left
  corner inside `rect`.
- `rect` X edges (native) must be multiples of 8. In portrait this
  applies to the clock rectangle's portrait Y edges (top/bottom), which
  become the native X edges after rotation.
- Anything wider than `rect` is clipped.
- Partial refresh ghosts, so a periodic full refresh is required.
- Call `Sleep()` after use.
- Cross-compile with `GOOS=linux GOARCH=arm GOARM=6 CGO_ENABLED=0`.

Verified on the same Pi Zero W with a spike in Nov 2026, drawing the
clock rect (88 px wide, 232 px tall on the panel):
- Timings: `Init` 0.17 s, full refresh (`DisplayImage`) 3.4–3.9 s,
  `InitPart` 0.17 s, partial update (`DisplayPart`) about 0.5 s, `Sleep`
  0.04 s.
- Partials 2 s apart were clean in every variant tried: sleep and
  re-init each time, re-init without sleep, init once without sleep,
  writing twice, and white-then-digits.
- A partial after about 60 s of deep sleep, without priming, gave a
  grainy, half-written digit in the clock rect.
- At 60 s, priming (writing the shown image of the rect, then the new
  one) was clean, and so was staying awake (no `Sleep`).
