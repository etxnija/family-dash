# ADR-0003: E-paper display behind a panel port, rendered on the Pi

## Status

Accepted

## Context

ADR-0002 gives the e-paper display its own `eink/` adapter and view-model. It leaves open how the adapter talks to the
panel and when it refreshes. The Waveshare 7.5" panel (800x480, 1-bit) was proven on the Pi Zero W (ARMv6) with the
pure-Go driver `github.com/ditrytus/epd7in5v2`: full and partial refresh both work. The driver is a young third-party
library (v0.1.0) and can only be exercised on the Pi itself.

## Decision

`eink/` owns a small `Panel` port: full refresh of a frame, partial refresh of a region of that frame, and sleep. The
port takes absolute frame coordinates; the real adapter hides the driver's quirks (crop to a rect-sized image,
X edges snapped to multiples of 8). A second adapter writes PNG files so layouts can be built and tested on the Mac.

One binary runs on the Pi and does everything: it collects the data, renders the frame and drives the panel. It is
cross-compiled with `GOOS=linux GOARCH=arm GOARM=6 CGO_ENABLED=0`. The `epd7in5v2` adapter is behind a Linux/ARM
build constraint, so the Mac build never needs it.

Refresh policy: partial refresh of the clock region every minute, full refresh of the whole frame about hourly or when
the content changes, and `Sleep` after every update. The clock comes from the Pi's own local clock.

## Consequences

- The renderer (view-model to 1-bit image) is plain Go and testable without hardware; only the thin driver adapter
  needs the Pi.
- Swapping the driver for a Python helper or another library only touches the one adapter behind `Panel`.
- Update rectangles must be sized from their content: anything wider than the rect is clipped.
- Partial refresh ghosts, so the hourly full refresh is required, not optional.
- Layout is fixed at 800x480 and needs an embedded font; the built-in bitmap font is too small for the final design.
- Deployment (copy the binary, run it as a service) is still manual and is decided separately.
