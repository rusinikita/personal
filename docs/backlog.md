# Backlog

List of ideas for future work. Each idea must be turned into a feature document (`docs/functions/{subdomain}-spec.md`) before implementation starts, per the AI-Driven Development Convention.

Each item is headed by the date it was added (DD-MM-YY), not a sequence number — that way removing a done item never forces renumbering the rest. When one item depends on another, reference it by date + title.

## 22-08-26 — Replace Budgets with Financial Goals

Budgets (`set_budget` / `get_budget_progress` MCP tools, `budgets` table) are set but never surfaced anywhere — no web dashboard tile, no bot flow — so they're effectively dead functionality. Plan: remove budgets entirely and replace them with a Financial Goals feature (e.g. "save X by date", "spend under X on category Y per period").

**Why:** Dead functionality adds maintenance surface (schema, MCP tools, repository methods) with no read path anyone actually uses. Goals reframe the same underlying need (spending discipline / savings targets) as something the dashboard can actively show progress against.

**Depends on:** none yet — needs its own feature document (`docs/functions/goals-spec.md` or a `money-spec.md` section, TBD) with full use cases before Stage 1 planning starts.

## 22-08-26 — Combined line + bar chart for trends

Add a combined chart type to the `action/webui` design system: a single chart overlaying a line series (e.g. running balance or average trend) with bar series (e.g. per-period totals) — an addition to the existing separate `RenderLineChart`/`RenderBarChart` components.

**Why:** Several trend views (e.g. monthly spend bars with a savings-rate line, or category totals with a trend overlay) need both a magnitude-per-period read and a trend-over-time read on the same chart, which today requires two separate charts.
