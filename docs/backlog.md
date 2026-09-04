# Backlog

List of ideas for future work. Each idea must be turned into a feature document (`docs/functions/{subdomain}-spec.md`) before implementation starts, per the AI-Driven Development Convention.

Each item is headed by the date it was added (DD-MM-YY), not a sequence number — that way removing a done item never forces renumbering the rest. When one item depends on another, reference it by date + title.

## 04-09-26 — E-ink Goals dashboard (no auth, black-and-white)

A dedicated page mirroring `action/progress/dashboard_web.go`'s `GET /web/progress`: purpose-built fixed-viewport (100vw/100vh), black-and-white, monospace styling for screenshot/e-ink display, unauthenticated (no `WebMiddleware`, same as `/web/progress` — see `transport/web/web.go`'s registration without `webAuth`). Content is just the active-goals tile grid (progress bar + label per goal, the same one `webui.RenderGoalTiles` renders on the dedicated `/web/goals` page), not the full dashboard — no Refresh form, no past/completed table, sized for an e-ink pad's glanceable view.

**Why:** `/web/progress`'s e-ink dashboard already gives a glanceable habit/mood snapshot for a physical always-on display; goals deserve the same kind of ambient, no-interaction view (savings/spend/PR progress at a glance) without opening a full browser session or logging in.

**Depends on:** none — the Goals feature is already implemented (`docs/functions/goals-spec.md`, `action/goals`), including `BuildGoalTiles`/`webui.RenderGoalTiles`, which this page would reuse.

## 29-08-26 — MCP tools to edit and delete progress points

`create_progress_point` and `search_progress_notes` exist for `ActivityPoint` (`domain/progress.go`), but there's no way to fix a mis-logged value/note or backdated timestamp, or remove a duplicate/mistaken entry, without touching the database directly. Add `edit_progress_point` (mutable fields: value, note, hours_left, progress_at) and `delete_progress_point` MCP tools, both scoped to the owning user's activity like the existing progress tools.

**Why:** Same gap already fixed for activities via `edit_activity` (see `docs/functions/progress-spec.md`) — logged progress points have the same correction need (wrong value tapped, typo in note, wrong day) but no fix path yet.

## 22-08-26 — Combined line + bar chart for trends

Add a combined chart type to the `action/webui` design system: a single chart overlaying a line series (e.g. running balance or average trend) with bar series (e.g. per-period totals) — an addition to the existing separate `RenderLineChart`/`RenderBarChart` components.

**Why:** Several trend views (e.g. monthly spend bars with a savings-rate line, or category totals with a trend overlay) need both a magnitude-per-period read and a trend-over-time read on the same chart, which today requires two separate charts.
