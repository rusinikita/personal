# Won't Do

Ideas that were written up in the backlog and then deliberately decided against, kept here instead of deleted so the reasoning isn't lost and the idea doesn't get re-proposed without knowing why it was rejected the first time. Each item keeps its original description and `**Why:**` (the case for doing it) plus a `**Won't do because:**` explaining the reversal.

## 04-09-26 — Goal card UX: last-updated time and per-card refresh icon

On the goal card (`action/webui/templates/components/goal_tiles.html`), show when the goal's progress was last updated, plus a refresh icon on each card to trigger progress refresh for that individual goal (rather than only a global refresh).

**Why:** Right now it's not visible from the card how stale a goal's progress is, and updating progress requires a global action (`refresh_goals_mcp.go`) instead of refreshing one goal at a time. Cheaper than it looks: `Goal.UpdatedAt` and single-goal refresh (`refreshOne`, already used by `refresh_goals`'s `goal_id` param) both already exist — this is template + one small web route, not new computation.

**Won't do because:** Goal tiles of type `activity_occurrence_count`/`activity_streak_count` are computed straight from activities (see `goals.BuildGoalTiles`, `progress-spec.md`), and this session concluded activities need an explicit `status` (active/paused/finished/dropped, see backlog) — a paused activity's streak shouldn't just keep counting as broken, and a dropped one probably shouldn't count at all once it's dropped. Until that status field lands and tile computation is updated to account for it, "last updated" staleness and a manual per-card refresh are measuring/acting on a notion of activity state that's about to change shape — premature UI investment on a model mid-flux. Revisit once status ships, possibly bundled with it.

## 05-09-26 — Generic entity-pinning table + web selector editing

A reusable `entity_pin` table, not tied to one subdomain, so any list (starting with `/web/progress`'s activities and `/web/goals/eink`'s goals) can have a handful of user-chosen entities pinned to fixed top positions instead of whatever `ListActivities`/`ListGoals` returns — replacing `/web/progress`'s current partial workaround (`TOP_ACTIVITY_ID` env var, a comma-separated priority list read in `buildDashboardDataFromDB`). Deliberately **pinning, not full sorting** (superseded an earlier fractional-indexing/drag-and-drop design — see `docs/functions/pins-spec.md` for the full rationale): the user only pins the few entities they want at the top; everything else keeps its existing order and renders after the pinned ones, so there's no need to manually place every item in a list. Schema: `user_id BIGINT, entity_type SMALLINT, position SMALLINT, entity_id BIGINT`, composite `PRIMARY KEY (user_id, entity_type, position)` (also directly serves "list this user's pins for one type, in order" — no secondary index needed) plus `UNIQUE (user_id, entity_type, entity_id)` so one entity never holds two positions. `position` is a small contiguous 1-based integer (not a fractional-indexing key — the pinned set per type is always small and user-curated, so plain integers with a cheap shift on insert/remove are simpler). `entity_id` is polymorphic (points at `activities`/`goals`/... depending on `entity_type`) so it can't carry a real FK — each domain's delete path must clean up its own `entity_pin` row.

Editing is web-only, no MCP tool, no client-side JS: each row gets a plain `<select>` (not pinned / position 1..N) inside a small form, `POST /web/pins` with `{entity_type, entity_id, position}`, same write-then-redirect pattern as the existing `POST /web/goals/refresh`.

**Why:** Activity/goal order on the e-ink displays is currently whatever the DB query happens to return, not what's actually most useful to glance at first, and the ad-hoc env-var workaround doesn't generalize to goals or scale to pinning from the web itself. A full manual-sort-everything UI was considered and rejected as more complexity (and more stale/garbage ordering data for entities nobody bothers to place) than the actual need, which is just keeping a few important goals/habits/projects/promises pinned to the top.

**Won't do because:** The actual problem this solves — important activities/goals not naturally showing up near the top of e-ink lists — turned out in this session to be mostly a side effect of not having an explicit activity `status`: with no way to mark an activity paused or dropped, it still sorts by `COALESCE((last_point_at::date + frequency_days) - CURRENT_DATE, 999999) ASC` (`repository.go`) and floats to the top looking like the most overdue thing, crowding out what's actually important — which is what pinning was compensating for. Once paused/dropped activities are filtered out of that sort instead of polluting it, most of the practical need for a whole generic pinning table + web selector goes away. Not worth the schema (`entity_pin`, composite keys) and UI right now — reassess after the status field ships, in case there's still a real case for manually reordering among genuinely-active items.

## 16-09-26 — Ideas table + MCP tools (separate from activities)

Add a new table (e.g. `ideas`: id, user_id, title, description, created_at, updated_at) plus MCP tools (`create_idea`, `edit_idea` to append/grow the description over time, `list_ideas`) for unformed thoughts that aren't ready to become an activity — no `progress_type`, no `frequency_days`, no progress points. Not auto-converted into an activity — promoting an idea is a manual action (create a new activity referencing the idea's text); the idea stays in its list afterward as a historical record.

**Why:** Ideas don't fit the `activities` schema — `progress_type` and `frequency_days` are `NOT NULL`/`CHECK`-constrained, so a not-yet-decided idea would need fake defaults to live there. A separate table keeps the activities list to genuinely committed, trackable things, and lets an idea's description accumulate freely across multiple sessions without periodic check-in semantics.

**Won't do because:** Absorbed by the backlog item "25-09-26 — Inbox: separate entity for the inbox → someday → spike lifecycle". Once `activity-rituals.md` (§2.1–2.5) was written, an "idea" turned out to be exactly an inbox capture with a lifecycle (inbox → someday → spike → decision). A standalone `ideas` table with only create/edit/list would duplicate that entity without its lifecycle.

## 21-09-26 — Split notes out of activity_progress into their own table

Move `activity_progress.note` (free-text, variable-length) out of the progress-point row into a separate table referencing the progress point it belongs to, instead of a column mixed in with `activity_progress`'s otherwise narrow, structured numeric/timestamp data. A later revision (25-09-26, "Move note text out of progress points into notes") turned it into a one-time migration of every non-empty `note` into the new notes entity, linked via `progress_point_id` with `ON DELETE CASCADE`, keeping the external API of `create_progress_point`/`edit_progress_point`/`search_progress_notes` unchanged.

**Why:** `activity_progress` should stay a uniform table of point-per-progress_type values; a prose-heavy `note` column sitting alongside it doesn't belong there and blocks giving notes their own index strategy. A dedicated notes table can carry a full-text index (`search_progress_notes` today does a plain `ILIKE` scan) or a vector/embedding index for semantic search, without either concern touching the core points table.

**Won't do because:** No real need for it right now. The inbox entity doesn't depend on this migration, so it was dropped rather than kept as a follow-up.

## 23-09-26 — Soft delete for progress points, with resolution tracking

Add a `deleted_at` + `resolution` field to progress points (values: `dropped`, `merged`, `expired`, `→ step`/`→ activity`/`→ note` with a reference id), replacing today's hard `delete_progress_point`. Deleted points are excluded from `get_activity_stats` and `search_progress_notes` by default.

**Why:** The weekly/monthly review ritual (`action/docs/content/activity-rituals.md`, §2.4) requires every inbox note to leave a recorded outcome when it's cleared out — dropped, merged into a duplicate, promoted into a step/activity/note, or expired after three monthly reviews with no traction — but the current hard delete destroys that decision instead of recording it.

**Won't do because:** `activity_progress` is an event log: editing and deleting a point is an exceptional correction, not part of a normal lifecycle, so soft delete with resolution statuses doesn't fit it. The requirement only existed because inbox thoughts were going to be `value=0` points on an «Inbox» activity. Inbox thoughts now get their own entity, and that entity carries the resolution (see backlog "25-09-26 — Inbox: separate entity for the inbox → someday → spike lifecycle").

## 23-09-26 — Duplicate-surfacing counter on progress points

Add a counter field to progress points that tracks how many times a duplicate idea has resurfaced, instead of tallying it informally in the note's text.

**Why:** The inbox ritual's spike trigger ("idea surfaced 3 times") and someday-expiry rule ("3 monthly reviews with no new bump") both key off this count (`action/docs/content/activity-rituals.md`, §2.3, §2.5) — without a real field, detecting either mechanically means re-parsing note history instead of reading one value.

**Won't do because:** It has the same cause as the soft-delete item above. A resurfacing counter is a property of an inbox idea, not of a check-in event, so it doesn't belong on `activity_progress`. It moves to the inbox entity.

## 25-09-26 — Notes: a separate entity with types, lifecycle, and optional links

A general-purpose `notes` table for all free text that isn't a progress event: types `idea` / `spike` / `activity` / `journal`, the idea lifecycle (`inbox` → `someday` → `resolved` with a resolution), a duplicate counter, and optional links to `activities`, `life_parts`, `workouts` (`ON DELETE SET NULL`). Later iterations also discussed:
- dropping `journal`;
- a lifecycle-free `note` type tied to progress points;
- a `source_note_id` back-link for ideas extracted from diary/retro entries during rituals.

**Why:** One place for every kind of note, each with a type-specific lifecycle and a link to what it's about (an activity, a workout, a life part), instead of free text scattered across progress points.

**Won't do because:** Too broad for the need. The only concrete requirement is the inbox mechanics from `activity-rituals.md`, so the backlog item was narrowed to that ("25-09-26 — Inbox: separate entity for the inbox → someday → spike lifecycle"). The name `notes` was also rejected for the narrowed feature and its table.
  - Diary and weekly-retro entries stay check-in notes on their activities. They are reviewed by date windows in rituals and need no lifecycle.
  - `journal` duplicated what an activity link already expresses.
  - Links to workouts and life parts had no use case in the documented process.
