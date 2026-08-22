# Backlog

List of ideas for future work. Each idea must be turned into a feature document (`docs/functions/{subdomain}-spec.md`) before implementation starts, per the AI-Driven Development Convention.

Each item is headed by the date it was added (DD-MM-YY), not a sequence number — that way removing a done item never forces renumbering the rest. When one item depends on another, reference it by date + title.

## 19-08-26 — Web interface for Money

A **read-only** web interface on top of the existing money functionality for reviewing balance, income/spending trends, and category breakdowns. Adding/editing/deleting transactions stays out of scope — bulk import already exists at `/money/import` (`action/money/import_web.go`), and this UI should link out to it rather than duplicate it.

**Why:** The money system is currently MCP/bot-only, so there's no way to see the overall financial picture (net worth trend, category weight, sync freshness) at a glance without asking the agent. A read-only dashboard makes that a one-look check.

**Use cases:**
- User should be able to see the date of the last transaction sync
- User should be able to view a table of categories sorted by average monthly spend over all time, with "total" and "last month" columns
- User should be able to drill into a category from the table and see its transactions
- User should be able to see total income earned over all time
- User should be able to see the current balance
- User should be able to see net (income minus spend) for the last calendar month
- User should be able to see average monthly savings (net) over all time
- User should be able to see a projected balance 3 months, 6 months, and 1 year out, based on the average monthly savings rate
- User should be able to navigate from the money dashboard to the existing transaction import page (`/money/import`)

## 19-08-26 — Web interface for Workouts

A **read-only** web interface on top of the workout functionality for reviewing personal records and per-exercise trends. Logging/editing workouts stays in the Telegram bot — this is view-only.

**Why:** Personal records and progression trends are hard to review in a chat interface. A read-only web UI gives a scannable table of records and a drill-down view for tracking progression on a specific exercise.

**Use cases:**
- User should be able to view a table of personal records, sorted by how many times each exercise has been performed
- User should be able to drill into a specific exercise from the table
- User should be able to view a trend chart of weight over time for the selected exercise
- User should be able to view a trend chart of reps over time for the selected exercise
