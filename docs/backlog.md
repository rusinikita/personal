# Backlog

List of ideas for future work. Each idea must be turned into a feature document (`docs/functions/{subdomain}-spec.md`) before implementation starts, per the AI-Driven Development Convention.

Each item is headed by the date it was added (DD-MM-YY), not a sequence number — that way removing a done item never forces renumbering the rest. When one item depends on another, reference it by date + title.

## 22-08-26 — Cross-domain Goals feature (replaces Budgets)

Budgets (`set_budget` / `get_budget_progress` MCP tools, `budgets` table) are set but never surfaced anywhere — no web dashboard tile, no bot flow — so they're effectively dead functionality. Grown beyond a money-only replacement: a new `action/goals` subdomain with six flat, sibling goal types — `money_saving` ("save X, optionally by a date"), `money_spend` (the old Budget, renamed), `exercise_max_weight` (reach X kg all-time max on a `workout` exercise, e.g. "bench press 100kg"), `exercise_total_volume` (lift X kg total on an exercise since the goal was created), `activity` (reach X occurrences of a `progress` activity since the goal was created, e.g. "meditate 30 times"), and `manual` (a standalone tally for anything not already tracked elsewhere, e.g. "read 12 books").

**Why:** Dead functionality adds maintenance surface (schema, MCP tools, repository methods) with no read path anyone actually uses. Goals reframe the same underlying need (spending discipline / savings targets / habit consistency) as one feature the dashboard can actively show progress against, instead of siloing it inside `money`.

Web presence is one shared tile component (`webui.GoalTileData`/`RenderGoalTiles`, progress bar + label): the dedicated `/web/goals` page shows every goal as a grid, and Money/Progress-browse/Workouts each embed the same tile grid pre-filtered to their own domain's goal types.

**Depends on:** none — feature document is ready at `docs/functions/goals-spec.md` (Stage 1 complete, pending approval to start Stage 2). `money-spec.md` had budgets removed, and `webui-spec.md`/`progress-spec.md`/`workout-spec.md` gained the shared goal-tile embedding, as part of this.

## 29-08-26 — MCP tools to edit and delete progress points

`create_progress_point` and `search_progress_notes` exist for `ActivityPoint` (`domain/progress.go`), but there's no way to fix a mis-logged value/note or backdated timestamp, or remove a duplicate/mistaken entry, without touching the database directly. Add `edit_progress_point` (mutable fields: value, note, hours_left, progress_at) and `delete_progress_point` MCP tools, both scoped to the owning user's activity like the existing progress tools.

**Why:** Same gap already fixed for activities via `edit_activity` (see `docs/functions/progress-spec.md`) — logged progress points have the same correction need (wrong value tapped, typo in note, wrong day) but no fix path yet.

## 22-08-26 — Combined line + bar chart for trends

Add a combined chart type to the `action/webui` design system: a single chart overlaying a line series (e.g. running balance or average trend) with bar series (e.g. per-period totals) — an addition to the existing separate `RenderLineChart`/`RenderBarChart` components.

**Why:** Several trend views (e.g. monthly spend bars with a savings-rate line, or category totals with a trend overlay) need both a magnitude-per-period read and a trend-over-time read on the same chart, which today requires two separate charts.
