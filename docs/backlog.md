# Backlog

List of ideas for future work. Each idea must be turned into a feature document (`docs/functions/{subdomain}-spec.md`) before implementation starts, per the AI-Driven Development Convention.

Each item is headed by the date it was added (DD-MM-YY), not a sequence number — that way removing a done item never forces renumbering the rest. When one item depends on another, reference it by date + title.

## 22-08-26 — Spending export & category view for Money

A dedicated screen, separate from the `/web/money` overview, for reviewing and exporting spending as CSV.

**Why:** The money dashboard shows category totals for quick glancing, but there's no way to pull a clean, shareable spending breakdown out of the system — e.g. to hand to someone else or feed into another tool — without going through the raw transaction export or asking the agent.

**Use cases:**
- User should be able to filter the view by date from and date to
- User should be able to see spending distribution across categories for the selected range
- User should be able to act on individual categories (e.g. exclude a category) from controls next to the category list
- User should be able to exclude specific categories entirely from the export
- User should see an export preview listing what will be included, and be able to select/deselect individual items before exporting
- User should be able to replace a category's exported sum with a custom value instead of the real total (e.g. to redact/adjust a figure in the shared file)
- Clicking export downloads a CSV file reflecting the filters, exclusions, preview selection, and any overridden sums

## 22-08-26 — Replace Budgets with Financial Goals

Budgets (`set_budget` / `get_budget_progress` MCP tools, `budgets` table) are set but never surfaced anywhere — no web dashboard tile, no bot flow — so they're effectively dead functionality. Plan: remove budgets entirely and replace them with a Financial Goals feature (e.g. "save X by date", "spend under X on category Y per period").

**Why:** Dead functionality adds maintenance surface (schema, MCP tools, repository methods) with no read path anyone actually uses. Goals reframe the same underlying need (spending discipline / savings targets) as something the dashboard can actively show progress against.

**Depends on:** none yet — needs its own feature document (`docs/functions/goals-spec.md` or a `money-spec.md` section, TBD) with full use cases before Stage 1 planning starts.

## 22-08-26 — Combined line + bar chart for trends

Add a combined chart type to the `action/webui` design system: a single chart overlaying a line series (e.g. running balance or average trend) with bar series (e.g. per-period totals) — an addition to the existing separate `RenderLineChart`/`RenderBarChart` components.

**Why:** Several trend views (e.g. monthly spend bars with a savings-rate line, or category totals with a trend overlay) need both a magnitude-per-period read and a trend-over-time read on the same chart, which today requires two separate charts.
