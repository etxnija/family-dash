# ADR-0002: Dual display adapters over a shared rendering interface

## Status

Accepted

## Context

The dashboard currently renders through `web/`, an HTML adapter for browser
display. A second display target — an e-paper panel (`eink/`) — is planned,
with different capabilities (no CSS, limited refresh, monochrome/greyscale
constraints) and a different rendering pipeline (e.g. image/bitmap output
instead of a template). Both would consume the same domain data
(`dashboard.Event`, homework groups, owners), raising the question of
whether to define a shared rendering `Port` that both adapters implement.

## Decision

Each display target owns its own adapter package and its own view-model,
built directly from domain types — there is no shared rendering interface
between `web/` and a future `eink/`. `web.NewPageData` and any future
`eink.NewPageData`-equivalent independently shape domain data into whatever
form their rendering pipeline needs, rather than conforming to a common
`PageData`/`Render` contract.

## Consequences

- Adding the e-paper adapter means writing its own view-model and layout
  logic from scratch, not reusing `web/page.go`'s types — some duplication
  of "shape domain data for display" logic across adapters is accepted.
- Each adapter is free to evolve its view-model to fit its rendering
  pipeline's real constraints (e.g. bitmap layout vs. HTML/CSS) without
  pressure to keep a shared interface abstract enough to fit both.
- The composition root (`cmd/family-dashboard/main.go`) wires each adapter
  independently rather than depending on one shared display port.
