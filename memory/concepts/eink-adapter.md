---
type: component
tags: [eink, epaper, render, adapter, hexagonal, font]
summary: eink/ e-paper adapter — Stockholm view-model, 1-bit 800x480 renderer (Go Mono, two columns, row limits, "…" cutting, byte-aligned ClockRect), plus verified Waveshare 7.5" V2 driver facts for the panel adapter.
links: [docs/arch/adr/0002-dual-display-adapters-over-shared-interface.md, docs/arch/adr/0003-eink-panel-port-and-refresh-policy.md, docs/raspberry-pi-setup.md, dashboard-page.md, composition-root.md, eink-view.md]
updated: 2026-11-10
generated: { by: worker, at: 2026-11-10T00:00:00Z }
verified: [{ by: "human", at: 2026-11-10 }]
---

# E-paper Adapter (`eink/`)

## Context & Rationale
`eink/` turns the same domain data as `web/` into a black/white 800x480
frame (`*image.Gray`) for the Waveshare 7.5" panel. Per
[ADR-0002](../../docs/arch/adr/0002-dual-display-adapters-over-shared-interface.md)
it is deliberately independent of `web/`: it has its own view-model and
renderer and imports nothing from `web`. Today the frame is only written to
a PNG (`-eink-png`, see [composition-root.md](composition-root.md)); the
panel driver comes later, behind the port in
[ADR-0003](../../docs/arch/adr/0003-eink-panel-port-and-refresh-policy.md).

## Key Rules & Mechanics
- `NewView` (`eink/view.go`, details in [eink-view.md](eink-view.md))
  converts every time to `Europe/Stockholm` and builds ready-to-draw
  Swedish strings; `Render(View)` only lays them out.
- `Render` draws anti-aliased text and rules on white, then `threshold`s
  every pixel at 128 (<128 → 0, else 255), so the output is strictly 1-bit.
- Two columns: left (X 16–288: clock, weather, SKOLLUNCH) and right
  (X 312–784: today, homework, upcoming), with a white 24 px gutter and no
  rule between them.
- Row limits are named constants (`maxEventRows`, `maxHomeworkRows`,
  `maxUpcomingRows`, `maxLunchRows`); `limitRows` keeps `limit-1` rows and
  adds `"+N till"` when a section overflows. Each section has fixed room
  for its maximum, so positions don't move with content.
- Over-wide text is cut with `"…"` (`fitText`; `fitParts` keeps homework
  day labels and upcoming times intact). Every region is drawn onto a
  clipped `SubImage` (header, columns, clock), so nothing spills outside it.
- `ClockRect()` (X 16–272, Y 64–152) is the only place the clock is drawn.
  Its X edges are multiples of 8 because one byte holds 8 horizontal
  pixels and `DisplayPart` requires byte-aligned X edges.
- Fonts are Go Mono and Go Mono Bold from `golang.org/x/image/font/gofont`
  (embedded, parsed once at init; `renderMu` serializes `Render` since
  faces aren't concurrency-safe). x/image is pinned at v0.36.0 (see the
  x/image gotcha below for the version thresholds).
- All layout numbers (sizes, baselines, column edges, limits, clock rect)
  live in `eink/layout.go`; tweak the layout there, not in `render.go`.

## Gotchas & Failure Modes
- x/image has two version thresholds, both read from the module cache:
  - v0.37.0–v0.45.0 declare `go 1.25.0`. Any of them makes `go mod tidy`
    rewrite our `go 1.25` directive, which is why we pin v0.36.0.
  - v0.46.0 and later declare `go 1.26.0` and need Go 1.26.
- Glyph coverage: Go Mono has åäö, `…`, `·`, `°`, `–`, `—`
  (`TestFaces_HaveGlyphs`); any new symbol must be added to that test, or it
  renders as a missing-glyph box.
- Mono fonts are wide: at 72 px "HH:MM" is about 216 px of the
  256 px clock rect. Changing the clock font or size means resizing `ClockRect` (keeping
  X edges on multiples of 8) and rechecking the left column.

## Hardware and driver facts for the later panel adapter
Verified on a Pi Zero W with the Waveshare 7.5" V2 and
`github.com/ditrytus/epd7in5v2` v0.1.0 (see
[raspberry-pi-setup.md](../../docs/raspberry-pi-setup.md)):
- The panel is 800x480 and 1-bit only (no greyscale).
- `DisplayImage` is a full refresh: about 4 s after `Init`, about 1.5 s
  after `InitFast`.
- `InitPart` followed by `DisplayPart(img, rect)` is a partial refresh,
  about 0.4 s.
- `DisplayPart` takes an image the size of `rect` with its origin at (0,0),
  not the full frame. Passing the full frame shows the frame's top-left
  corner inside `rect`.
- `rect` X edges must be multiples of 8.
- Anything wider than `rect` is clipped.
- Partial refresh ghosts, so a periodic full refresh is required.
- Call `Sleep()` after use.
- Cross-compile with `GOOS=linux GOARCH=arm GOARM=6 CGO_ENABLED=0`.
