# ADR-0004: Portrait frame, rotated by the panel adapter

## Status

Accepted

## Context

The Waveshare 7.5" panel is landscape-native (800x480), but the display will hang portrait in a picture frame, like
the web page and its mockup. ADR-0003 fixed the layout at 800x480; this decision changes that part.

## Decision

`eink/` renders a portrait 480x800 frame. The PNG written for preview shows exactly what the mounted frame shows.
The `Panel` port also works in portrait coordinates; the driver adapter alone rotates the frame by 90 degrees to the
panel's native 800x480 and maps partial-refresh rectangles to native coordinates. The rotation direction is a single
constant in that adapter, chosen by how the panel is mounted.

Because the native X axis becomes the portrait Y axis, the clock rectangle's top and bottom edges (not its left and
right) must be multiples of 8. Long entries wrap to at most two lines instead of being cut off.

## Consequences

- The renderer stays hardware-free and the PNG preview is the true mounted view.
- Rotation and rect mapping are one adapter's job and need their own tests.
- A full-frame rotation on the Zero W costs a little CPU per refresh, which is acceptable at one refresh a minute.
- Flipping which edge is the top later is a one-constant change in the adapter, not a re-layout.
- Fewer entries fit per section because wrapped entries take two lines.
