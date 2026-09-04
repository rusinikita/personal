# Backlog

List of ideas for future work. Each idea must be turned into a feature document (`docs/functions/{subdomain}-spec.md`) before implementation starts, per the AI-Driven Development Convention.

Each item is headed by the date it was added (DD-MM-YY), not a sequence number — that way removing a done item never forces renumbering the rest. When one item depends on another, reference it by date + title.

## 29-08-26 — MCP tools to edit and delete progress points

`create_progress_point` and `search_progress_notes` exist for `ActivityPoint` (`domain/progress.go`), but there's no way to fix a mis-logged value/note or backdated timestamp, or remove a duplicate/mistaken entry, without touching the database directly. Add `edit_progress_point` (mutable fields: value, note, hours_left, progress_at) and `delete_progress_point` MCP tools, both scoped to the owning user's activity like the existing progress tools.

**Why:** Same gap already fixed for activities via `edit_activity` (see `docs/functions/progress-spec.md`) — logged progress points have the same correction need (wrong value tapped, typo in note, wrong day) but no fix path yet.

## 04-09-26 — Goal card UX: last-updated time and per-card refresh icon

On the goal card (`action/webui/templates/components/goal_tiles.html`), show when the goal's progress was last updated, plus a refresh icon on each card to trigger progress refresh for that individual goal (rather than only a global refresh).

**Why:** Right now it's not visible from the card how stale a goal's progress is, and updating progress requires a global action (`refresh_goals_mcp.go`) instead of refreshing one goal at a time.

## 22-08-26 — Combined line + bar chart for trends

Add a combined chart type to the `action/webui` design system: a single chart overlaying a line series (e.g. running balance or average trend) with bar series (e.g. per-period totals) — an addition to the existing separate `RenderLineChart`/`RenderBarChart` components.

**Why:** Several trend views (e.g. monthly spend bars with a savings-rate line, or category totals with a trend overlay) need both a magnitude-per-period read and a trend-over-time read on the same chart, which today requires two separate charts.
