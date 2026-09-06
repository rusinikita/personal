# Backlog

List of ideas for future work. Each idea must be turned into a feature document (`docs/functions/{subdomain}-spec.md`) before implementation starts, per the AI-Driven Development Convention.

Each item is headed by the date it was added (DD-MM-YY), not a sequence number — that way removing a done item never forces renumbering the rest. When one item depends on another, reference it by date + title. Items are listed in rough priority order (top = next), not by date.

## 04-09-26 — Goal card UX: last-updated time and per-card refresh icon

On the goal card (`action/webui/templates/components/goal_tiles.html`), show when the goal's progress was last updated, plus a refresh icon on each card to trigger progress refresh for that individual goal (rather than only a global refresh).

**Why:** Right now it's not visible from the card how stale a goal's progress is, and updating progress requires a global action (`refresh_goals_mcp.go`) instead of refreshing one goal at a time. Cheaper than it looks: `Goal.UpdatedAt` and single-goal refresh (`refreshOne`, already used by `refresh_goals`'s `goal_id` param) both already exist — this is template + one small web route, not new computation.

## 05-09-26 — Generic entity-pinning table + web selector editing

A reusable `entity_pin` table, not tied to one subdomain, so any list (starting with `/web/progress`'s activities and `/web/goals/eink`'s goals) can have a handful of user-chosen entities pinned to fixed top positions instead of whatever `ListActivities`/`ListGoals` returns — replacing `/web/progress`'s current partial workaround (`TOP_ACTIVITY_ID` env var, a comma-separated priority list read in `buildDashboardDataFromDB`). Deliberately **pinning, not full sorting** (superseded an earlier fractional-indexing/drag-and-drop design — see `docs/functions/pins-spec.md` for the full rationale): the user only pins the few entities they want at the top; everything else keeps its existing order and renders after the pinned ones, so there's no need to manually place every item in a list. Schema: `user_id BIGINT, entity_type SMALLINT, position SMALLINT, entity_id BIGINT`, composite `PRIMARY KEY (user_id, entity_type, position)` (also directly serves "list this user's pins for one type, in order" — no secondary index needed) plus `UNIQUE (user_id, entity_type, entity_id)` so one entity never holds two positions. `position` is a small contiguous 1-based integer (not a fractional-indexing key — the pinned set per type is always small and user-curated, so plain integers with a cheap shift on insert/remove are simpler). `entity_id` is polymorphic (points at `activities`/`goals`/... depending on `entity_type`) so it can't carry a real FK — each domain's delete path must clean up its own `entity_pin` row.

Editing is web-only, no MCP tool, no client-side JS: each row gets a plain `<select>` (not pinned / position 1..N) inside a small form, `POST /web/pins` with `{entity_type, entity_id, position}`, same write-then-redirect pattern as the existing `POST /web/goals/refresh`.

**Why:** Activity/goal order on the e-ink displays is currently whatever the DB query happens to return, not what's actually most useful to glance at first, and the ad-hoc env-var workaround doesn't generalize to goals or scale to pinning from the web itself. A full manual-sort-everything UI was considered and rejected as more complexity (and more stale/garbage ordering data for entities nobody bothers to place) than the actual need, which is just keeping a few important goals/habits/projects/promises pinned to the top.
