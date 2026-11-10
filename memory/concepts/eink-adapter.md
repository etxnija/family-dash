---
type: component
tags: [eink, epaper, render, adapter, hexagonal, font]
summary: eink/ e-paper adapter — Stockholm view-model, 1-bit portrait 480x800 renderer (Go Mono, single column, clock/weather band, two-line wrapped entries, content-height sections trimmed by priority, ClockRect top/bottom on multiples of 8), plus verified Waveshare 7.5" V2 driver facts for the panel adapter.
links: [docs/arch/adr/0002-dual-display-adapters-over-shared-interface.md, docs/arch/adr/0003-eink-panel-port-and-refresh-policy.md, docs/arch/adr/0004-portrait-frame-rotated-by-panel-adapter.md, docs/raspberry-pi-setup.md, dashboard-page.md, composition-root.md, eink-view.md]
updated: 2026-11-10
generated: { by: worker, at: 2026-11-10T00:00:00Z }
verified: [{ by: "human", at: 2026-11-10 }, { by: "human", at: 2026-11-10 }, { by: "human", at: 2026-11-10 }]
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
- Glyph coverage: Go Mono has åäö, `…`, `·`, `°`, `–`, `—`
  (`TestFaces_HaveGlyphs`); any new symbol must be added to that test, or it
  renders as a missing-glyph box.
- Mono fonts are wide: at 72 px "HH:MM" is about 216 px of the
  232 px clock rect. Changing the clock font or size means resizing `ClockRect` (keeping
  Y edges on multiples of 8) and rechecking the weather width beside it.

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
- `rect` X edges (native) must be multiples of 8. In portrait this
  applies to the clock rectangle's portrait Y edges (top/bottom), which
  become the native X edges after rotation.
- Anything wider than `rect` is clipped.
- Partial refresh ghosts, so a periodic full refresh is required.
- Call `Sleep()` after use.
- Cross-compile with `GOOS=linux GOARCH=arm GOARM=6 CGO_ENABLED=0`.
