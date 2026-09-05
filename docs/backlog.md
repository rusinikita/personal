# Backlog

List of ideas for future work. Each idea must be turned into a feature document (`docs/functions/{subdomain}-spec.md`) before implementation starts, per the AI-Driven Development Convention.

Each item is headed by the date it was added (DD-MM-YY), not a sequence number — that way removing a done item never forces renumbering the rest. When one item depends on another, reference it by date + title. Items are listed in rough priority order (top = next), not by date.

## 05-09-26 — Show activity description on the browse list and detail screen

`domain.Activity.Description` already exists and renders on `/web/progress`'s e-ink page (`activity-desc` div), but `/web/progress/browse`'s activity table and `/web/progress/browse/{id}`'s detail view (`action/progress/browse_web.go`) don't show it at all.

**Why:** The description is only visible on the screenshot dashboard today — anyone using the full browse/detail pages to check an activity has no way to see why it was created or what it's tracking.

## 05-09-26 — Generic entity-ordering table (fractional indexing) + web drag-and-drop editing

A reusable `entity_order` table, not tied to one subdomain, so any list (starting with `/web/progress`'s activities and `/web/goals/eink`'s goals) can have a user-controlled display order instead of whatever `ListActivities`/`ListGoals` returns — replacing `/web/progress`'s current partial workaround (`TOP_ACTIVITY_ID` env var, a comma-separated priority list read in `buildDashboardDataFromDB`). Schema: `user_id BIGINT, entity_type SMALLINT, entity_id BIGINT, idx TEXT COLLATE "C"`, composite `PRIMARY KEY (user_id, entity_type, entity_id)` (serves the per-entity upsert/lookup on every move) plus a secondary index on `(user_id, entity_type, idx)` (serves "list this user's slots for one type, in order" — the PK's column order can't serve that sort). `idx` is a fractional-indexing key (string-based, not float, to avoid precision exhaustion — see e.g. Figma's `fractional-indexing` or Jira's LexoRank), `COLLATE "C"` so Postgres compares it byte-wise rather than locale-aware. `entity_id` is polymorphic (points at `activities`/`goals`/... depending on `entity_type`) so it can't carry a real FK — each domain's delete path must clean up its own `entity_order` row.

Editing is web-only, no MCP tool: drag-and-drop via SortableJS (already precedented — `layout.html` loads `chart.js` from jsdelivr the same way) on each orderable list, posting the moved item's new neighbors to one generic endpoint (e.g. `POST /web/order` with `{entity_type, entity_id, before_entity_id, after_entity_id}`), which computes a new key between them and upserts the single row.

**Why:** Activity/goal order on the e-ink displays is currently whatever the DB query happens to return, not what's actually most useful to glance at first, and the ad-hoc env-var workaround doesn't generalize to goals or scale to reordering from the web itself.

## 29-08-26 — MCP tools to edit and delete progress points

`create_progress_point` and `search_progress_notes` exist for `ActivityPoint` (`domain/progress.go`), but there's no way to fix a mis-logged value/note or backdated timestamp, or remove a duplicate/mistaken entry, without touching the database directly. Add `edit_progress_point` (mutable fields: value, note, hours_left, progress_at) and `delete_progress_point` MCP tools, both scoped to the owning user's activity like the existing progress tools.

**Why:** Same gap already fixed for activities via `edit_activity` (see `docs/functions/progress-spec.md`) — logged progress points have the same correction need (wrong value tapped, typo in note, wrong day) but no fix path yet.

## 05-09-26 — Bigger text + rounded corners on e-ink goal cards

On `/web/goals/eink`'s cards specifically: larger text (name/label sizes are currently tuned for the dense `/web/progress` layout, cramped for a goal card with fewer items) and rounded corners (currently sharp `border: 1px solid #000`, see `dashboard_web_eink.go`'s inline CSS).

**Why:** The goals e-ink page was styled as a quick reuse of the progress page's density-first look rather than tuned for its own content, which has far fewer items per screen.

## 05-09-26 — Goal card drill-down link

On the goal card (`webui.GoalTileData`/`RenderGoalTiles`, `action/webui/templates/components/goal_tiles.html`), add a link to the relevant drill-down page per `goal_type`: `money_saving`/`money_spend` → the category filter page (`/web/money/transactions?category=...`, same URL `action/money/dashboard_web.go`'s `categoryLinkURL` builds), `exercise_max_weight`/`exercise_total_volume` → that exercise's page (`/web/workouts/{exercise_id}`), `activity_occurrence_count`/`activity_streak_count` → that activity's page (`/web/progress/browse/{activity_id}`). `manual` goals get no link — they have no underlying page to point to.

**Why:** A goal card only shows the cached progress number today; clicking through to the underlying category/exercise/activity to see the detail behind that number requires manually navigating there instead of one click from the card.

## 04-09-26 — Goal card UX: last-updated time and per-card refresh icon

On the goal card (`action/webui/templates/components/goal_tiles.html`), show when the goal's progress was last updated, plus a refresh icon on each card to trigger progress refresh for that individual goal (rather than only a global refresh).

**Why:** Right now it's not visible from the card how stale a goal's progress is, and updating progress requires a global action (`refresh_goals_mcp.go`) instead of refreshing one goal at a time. Cheaper than it looks: `Goal.UpdatedAt` and single-goal refresh (`refreshOne`, already used by `refresh_goals`'s `goal_id` param) both already exist — this is template + one small web route, not new computation.

## 22-08-26 — Combined line + bar chart for trends

Add a combined chart type to the `action/webui` design system: a single chart overlaying a line series (e.g. running balance or average trend) with bar series (e.g. per-period totals) — an addition to the existing separate `RenderLineChart`/`RenderBarChart` components.

**Why:** Several trend views (e.g. monthly spend bars with a savings-rate line, or category totals with a trend overlay) need both a magnitude-per-period read and a trend-over-time read on the same chart, which today requires two separate charts.
