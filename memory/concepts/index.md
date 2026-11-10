# Concept Index

Index of durable codebase concepts, patterns, decisions, and components.
Each concept file contains YAML frontmatter metadata (`type`, `tags`, `summary`, `links`, `updated`).

---

- **[sandboxing.md](sandboxing.md)** (`type: decision` | `tags: [sandboxing, worktree, security, permissions]`) — Git worktree isolation and .claude permission rules.
- **[sensors-coverage.md](sensors-coverage.md)** (`type: component` | `tags: [sensors, coverage, lint, pyright]`) — Sensor retry mechanics, type checks, and diff-coverage floor.
- **[metrics.md](metrics.md)** (`type: component` | `tags: [metrics, cost, session, telemetry]`) — Driver call accounting, cost tracking, and session ID recording.
- **[adversarial-review.md](adversarial-review.md)** (`type: pattern` | `tags: [reviewer, review, haiku, clean-slate]`) — Clean-slate Haiku review agent, verdict markers, and non-blocking retry budget.
- **[task-decomposition.md](task-decomposition.md)** (`type: pattern` | `tags: [invest, loop, task, commits]`) — INVEST task sizing, per-task commit hooks, and merge workflow.
- **[webcal-adapter.md](webcal-adapter.md)** (`type: component` | `tags: [webcal, sportadmin, ics, adapter, hexagonal]`) — webcal ICS calendar adapter (SportAdmin source): parsing rules, Stockholm timezone handling, URL-secrecy discipline.
- **[composition-root.md](composition-root.md)** (`type: component` | `tags: [main, config, composition-root, multi-source, cmd, http, lunch, weather, eink, png]`) — cmd/family-dashboard main(): JSON sources+school+location config, adapter construction at startup, Collector/LunchSource/WeatherSource wiring, shared fetchDashboard for the HTML handler and the `-eink-png` export.
- **[dashboard-page.md](dashboard-page.md)** (`type: component` | `tags: [web, page, template, i18n, funcmap, receipt-design, lunch]`) — web/page.go PageData/Render: receipt-style monochrome template, Swedish display translations via FuncMap, data-driven Idag/Homework/Kommande dagar/Skollunch panels.
- **[collector.md](collector.md)** (`type: component` | `tags: [collector, domain, multi-source, owner-tagging]`) — collector.go's Collector type: merges owner-tagged events from multiple EventSources, derives today/homework/upcoming views.
- **[skolmaten-adapter.md](skolmaten-adapter.md)** (`type: component` | `tags: [skolmaten, lunch, rss, adapter, hexagonal]`) — lunch.go's LunchMenu/LunchSource port and the skolmaten RSS adapter, wired into main.go/handleIndex/PageData.Lunch.
- **[smhi-adapter.md](smhi-adapter.md)** (`type: component` | `tags: [smhi, weather, adapter, hexagonal]`) — weather.go's WeatherReading/WeatherSource port and the SMHI point-forecast adapter (temperature, low, sky, precipitation), wired into main.go/handleIndex/PageData.Weather.
- **[eink-view.md](eink-view.md)** (`type: component` | `tags: [eink, e-paper, view-model, i18n, adapter, hexagonal]`) — eink/view.go NewView/View: e-paper view-model of ready-to-draw Swedish strings, built independently of web/.
- **[eink-adapter.md](eink-adapter.md)** (`type: component` | `tags: [eink, epaper, render, adapter, hexagonal, font]`) — eink/ e-paper adapter: 1-bit 800x480 renderer (Go Mono, columns, row limits, byte-aligned ClockRect) and verified Waveshare 7.5" V2 driver facts.
