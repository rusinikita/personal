# Achievements - Complete Specification

## Overview

Cross-domain system for tracking personal achievements with MCP (Model Context Protocol) interface and a read-mostly web dashboard (one write route: a "Refresh" action, see HTTP Handlers). An achievement is exactly one of seven types, each with its own required fields and its own way of computing current progress — there is no shared "money"/"exercise"/"activity" grouping, every type sits at the same level:

- **`money_saving`** — reach X in accumulated balance ("save X, optionally by a date")
- **`money_spend`** — don't spend more than X on a category (the old `Budget`, renamed)
- **`exercise_max_weight`** — reach X kg all-time max on a specific `workout` exercise (see `workout-spec.md`) — e.g. "bench press 100kg"
- **`exercise_total_volume`** — lift X kg total (weight × reps, summed) on a specific exercise since the achievement was created — e.g. "lift 5000kg total this quarter"
- **`activity_occurrence_count`** — reach X occurrences of a `progress` activity since the achievement was created (see `progress-spec.md`) — e.g. "meditate 30 times"
- **`activity_streak_count`** — reach a streak of X consecutive check-ins on a `progress` activity, where consecutive means the gap between two check-ins is within the activity's own `frequency_days` — e.g. "30-day meditation streak"
- **`manual`** — reach X of a standalone running tally the achievement owns itself, for anything not already tracked elsewhere — e.g. "read 12 books this year"

All seven share the same shape (name, target value, optional deadline) and live in one `achievements` table, distinguished by a single `achievement_type` column, with a shared `current_value` column holding every achievement's most-recently-known progress. `manual` writes it directly (`log_achievement_progress`'s increment) — it owns no other data to derive from, so it's always exact. The other six (`money_saving`, `money_spend`, `exercise_max_weight`, `exercise_total_volume`, `activity_occurrence_count`, `activity_streak_count`) derive their value from data that already exists elsewhere, but that derivation is **cached, not live**: it runs once at `create_achievement` (an initial snapshot) and again only when explicitly asked for via the `refresh_achievements` MCP tool or the Achievements page's "Refresh" button — reads (`get_achievement_progress`, the web dashboard, embedded tile grids) always return whatever was last cached, never recompute on the spot. An achievement's progress can be stale between refreshes; that's the deliberate trade for reads that don't fan out into every other subdomain's tables on every page load.

On the web, an achievement always renders as the same **tile** (name, progress bar, label, optional deadline — see `webui-spec.md`'s `AchievementTileData`/`RenderAchievementTiles`), in one of two places: the dedicated `GET /web/achievements` page shows every achievement as a grid, and each of Money (`GET /web/money`), Progress browse (`GET /web/progress/browse`), and Workouts (`GET /web/workouts`) embeds the same tile grid filtered down to its own domain's achievement types — `money_saving`/`money_spend` on Money, `activity_occurrence_count`/`activity_streak_count` on Progress browse, `exercise_max_weight`/`exercise_total_volume` on Workouts. `manual` achievements have no domain page to embed into, so they only ever appear on the dedicated Achievements page.

## Rename Map (goals → achievements, 23-09-26)

Pure rename of the former `goals` subdomain — no behavior change. "Goal" read as a life goal; these are gamification bars (see `activity-rituals.md` §2.10), so the name now says so. Enum *values* (`money_saving`, `manual`, …) don't contain "goal" and stay as-is.

| Area | Old | New |
|---|---|---|
| Table / column | `goals`, `goal_type` | `achievements`, `achievement_type` |
| Constraints / indexes | `check_goal_type`, `check_goal_period`, `idx_goals_user_period`, `idx_goals_user_type` | `check_achievement_type`, `check_achievement_period`, `idx_achievements_user_period`, `idx_achievements_user_type` |
| Migration file | `gateways/db/migrations/z_goals.sql` | `gateways/db/migrations/z_achievements.sql` (see Database Schema) |
| Domain (`domain/goal.go` → `domain/achievement.go`) | `Goal`, `GoalType`, `GoalType*` consts, `AllGoalTypes`, `GoalUpdate`, `GoalFilter` | `Achievement`, `AchievementType`, `AchievementType*`, `AllAchievementTypes`, `AchievementUpdate`, `AchievementFilter` |
| Repository (`gateways/interfaces.go`, `db/repository.go`, `db/mock.go`) | `CreateGoal`, `UpdateGoal`, `GetGoal`, `ListGoals` (+ private `goalDetails`, `scanGoal`, `goalColumns`, …) | `CreateAchievement`, `UpdateAchievement`, `GetAchievement`, `ListAchievements` (+ `achievementDetails`, …) |
| Package | `action/goals` (`*_goal_*` / `refresh_goals_mcp.go` files) | `action/achievements` (`*_achievement_*` / `refresh_achievements_mcp.go`) |
| MCP tools | `create_goal`, `update_goal`, `get_goal_progress`, `log_goal_progress`, `refresh_goals` | `create_achievement`, `update_achievement`, `get_achievement_progress`, `log_achievement_progress`, `refresh_achievements` |
| MCP JSON fields | `goal_id`, `goal_type`, `goal`, `goals` | `achievement_id`, `achievement_type`, `achievement`, `achievements` |
| Web routes | `GET /web/goals`, `POST /web/goals/refresh`, `GET /web/goals/eink` | `GET /web/achievements`, `POST /web/achievements/refresh`, `GET /web/achievements/eink`; the old `GET /web/goals/eink` stays as a 301 redirect (a physical e-ink display points at it), no other old URLs or MCP tool aliases kept |
| Nav (`action/webui/nav.go`) | `NavGoals`, label "Goals" | `NavAchievements`, label "Achievements" |
| Embedded tiles (`action/webui`) | `GoalTileData`, `GoalTilesData`, `RenderGoalTiles`, `components/goal_tiles.html`, CSS `webui-goal-tile*`, `--webui-goal-over-*`; callers' `BuildGoalTiles`, `moneyGoalTypes`/`progressGoalTypes`/`workoutGoalTypes` | `AchievementTileData`, `AchievementTilesData`, `RenderAchievementTiles`, `components/achievement_tiles.html`, `webui-achievement-tile*`, `--webui-achievement-over-*`; `BuildAchievementTiles`, `money/progress/workoutAchievementTypes` |
| Errors / tool text | `delete_activity` "a goal still references it" | "an achievement still references it" |
| Agent-facing text | `transport/mcp/instructions.md`, `action/docs/content/activity-mechanics.md` (§5 "Goal", tool table), `activity-rituals.md` (§2.10 "Goal = achievement" and other "goal" mentions), `activity-riturals-sorces.md` (entity mentions only, not cited paper titles) | "Achievement"; the "Goal = achievement / name is historical" notes become just "Achievement" |
| Tests | `tests/goals_test.go`, `tests/goals_dashboard_web_test.go`, `Test*Goal*` names in other test files | `tests/achievements_test.go`, `tests/achievements_dashboard_web_test.go`, `Test*Achievement*` |
| Specs | `docs/functions/goals-spec.md`; references in `money-`, `progress-`, `workout-`, `webui-`, `docs-spec.md` | this file; references updated |

Not renamed: plain-English "goal" that isn't this entity (e.g. `progress-spec.md` Overview, `domain/progress.go` comments, `CLAUDE_*_INSTRUCTIONS.md`), and history in `docs/wontdo.md`.

## Best Practices Applied

- **Multi-user Support**: `achievements` has `user_id` for data isolation
- **Rename in place, data kept**: `ApplyMigrations` re-runs every `migrations/*.sql` on each startup, so `z_goals.sql` is replaced (not kept alongside — it would recreate an empty `goals` table every start) by `z_achievements.sql`: an idempotent rename preamble for existing DBs, then the same `CREATE TABLE IF NOT EXISTS` under the new names for fresh ones. The `z_` prefix is kept so it still runs after `progress.sql`/`workout.sql` (FKs)
- **One flat discriminator, no nested sub-types, ever**: `achievements.achievement_type` is one of seven values, all siblings. Two earlier drafts introduced a second-level discriminator — first `metric_source` nesting `exercise`/`activity` under a shared `count` type, then `exercise_metric` nesting `max_weight`/`total_volume` under a shared `exercise` type — both times because two things that differ in calculation the same way one achievement type already differs from another (`money_saving` vs. `money_spend`) got treated as sub-cases instead of siblings. The rule going forward: if two variants need different required fields or a different progress calculation, they're different `achievement_type` values, full stop — no exceptions carved out for "this one's basically the same type"
- **`money_spend` is `Budget` renamed, not redesigned**: `category` + `target_value` cap, matched the same way (`category` prefix match against transactions). `category` is required (non-empty) for this type only
- **`money_saving` snapshots the starting balance**: `baseline_balance_eur` is captured once at creation time (`GetBalance` as of `starts_at`) and never recomputed — progress is "how much have I saved *since setting this achievement*," not "what's my total balance relative to the target." Required (non-null) for this type only
- **`exercise_max_weight` reads an all-time max, not a delta since creation**: via `workout`'s existing `GetPersonalRecords(userID, exercise_id).MaxWeight` — a PR-chasing achievement is an absolute bar, so an achievement is already met if the user hit that weight before creating it, no baseline needed
- **`exercise_total_volume` sums `weight_kg × reps` since `starts_at`**: a new achievement-scoped query (`GetExerciseVolume`) — unlike max weight, "lift 5000kg total this quarter" is inherently a since-creation tally, so it can't reuse `GetPersonalRecords`
- **Both exercise types require `exercise_id`, nothing else exercise-specific**: no metric field to also set — the type itself *is* the metric, so `create_achievement` only needs to know which exercise, not which exercise plus which of two modes
- **`activity_occurrence_count` achievements count occurrences since achievement creation**: current value is `CountProgress({ActivityID: activity_id, From: starts_at})` (existing `progress` repository method, see `progress-spec.md`) — one progress point logged for that activity = one occurrence. This assumes the activity already exists (e.g. a `habit_progress` "Meditation" activity) — creating one is out of scope for `create_achievement`, it's `progress`'s `create_activity`. `activity_id` is required for this type only. Named specifically (not just `activity`) so future `progress`-linked mechanics — the same way `money`/`exercise` each have two mechanics — can become their own sibling `achievement_type` values instead of overloading this one. `activity_streak_count` (below) is the first such sibling. This computation runs at `create_achievement` and `refresh_achievements` only — see the caching bullet below
- **`activity_streak_count` counts consecutive check-ins, evaluated as of whenever it was last computed**: fetch the activity's `frequency_days` (`GetActivity`, existing `progress` method) and its points via `ListProgress({ActivityID: activity_id, To: at})` ordered newest-first (existing `progress` method, no new query needed), then walk from the most recent point backward, counting while each gap to the next-older point is `<= frequency_days`. If the *first* gap — from `at` back to the most recent point itself — already exceeds `frequency_days`, the streak has already lapsed and `current_value = 0` even though history exists; this matches ordinary streak semantics (a missed check-in resets it), not an all-time best. `activity_id` is required for this type, same as `activity_occurrence_count`. `at` is `now` at whichever moment `create_achievement` or `refresh_achievements` ran it — see the caching bullet below for why this is no longer literally "as of now" on every read
- **`manual` is a plain counter on the achievement row, not an append-only log**: `current_value` lives directly on `achievements` and `log_achievement_progress` does a single increment. An earlier draft added an `achievement_progress_entries` table mirroring `activity_progress`'s append-only shape (one row per log call, with note/timestamp) for history and per-entry notes — dropped as unnecessary weight for what this type actually needs: a running total, not an audit trail. The trade-off is real (no history of *when* progress happened, no per-increment notes) and accepted deliberately, not overlooked
- **`current_value` is its own `DECIMAL` column, not packed into `details`**: an earlier draft packed it into the JSONB column alongside `category`/`unit`/`baseline_balance_eur` as "just another sparse per-type scalar." Reconsidered for two reasons: it's the scalar with the most write paths (`log_achievement_progress`, `update_achievement` for `manual`, and now `create_achievement`/`refresh_achievements` for the other six), and — since the caching change below — it's no longer even type-scoped, every achievement has one. A plain `UPDATE ... SET current_value = ...` is simpler and cheaper than `jsonb_set` regardless; `category`/`unit`/`baseline_balance_eur` stay in `details` since they're genuinely write-once-at-creation and type-scoped, the sparse case JSONB actually fits
- **`current_value` is cached, refreshed on demand — not recomputed on every read**: an earlier draft had `get_achievement_progress` (and every page/tile render built on it) run each of the per-type calculations above live, on every call. Reconsidered: `exercise_total_volume`'s sum, `activity_streak_count`'s full point-history walk, and a `money_saving`/`money_spend` balance/spend query all cost real work, and the dedicated Achievements page plus three embedded tile grids (Money/Progress-browse/Workouts) could each trigger that work on every load — for numbers that mostly don't change between one page view and the next. Now that computation only runs at `create_achievement` (initial snapshot) and `refresh_achievements` (on demand, see below); every read — `get_achievement_progress`, `ListAchievements`, `BuildAchievementTiles` — is a plain `SELECT` against the cached `current_value` column. `manual` is unaffected either way, since it was always a plain column, never computed
- **No `AchievementProgress` type, no `GetAchievementProgress` method — everything comes from `Achievement`**: an earlier draft wrapped `Achievement` in an `AchievementProgress{ Achievement; CurrentValue; RemainingValue }` struct returned by its own `GetAchievementProgress(userID, at, types)` repository method. Once `current_value` became a plain cached field on `Achievement` itself (see the bullets above), that wrapper stopped adding anything — `CurrentValue` was a straight duplicate of `Achievement.CurrentValue`, and `RemainingValue` is a one-line subtraction, not a query result. Reconsidered: `ListAchievements(AchievementFilter{ActiveOnly, At, Types})` (extended with the two fields `GetAchievementProgress` used to own) is the one read path for a set of achievements, full stop, and `Achievement.RemainingValue()` is a plain getter method called on whatever a caller already has — no second type, no second method, no round-trip a getter can't replace
- **`refresh_achievements` shares its computation with `create_achievement`'s initial snapshot, not a second copy**: both call the same internal per-type logic (see the "Refresh Achievements" sequence diagram) — `create_achievement` runs it once right after validating a new achievement (so it doesn't sit at a stale/zero value until the first refresh; a backdated `exercise_max_weight` achievement can even start out already met), `refresh_achievements` runs it again later, for one achievement (`achievement_id` provided) or every active achievement owned by the user (`achievement_id` omitted). `manual` achievements are skipped by `refresh_achievements` — there's nothing to derive, `current_value` is already authoritative
- **Cross-domain reads go straight through the existing shared repository, no new abstraction**: per `docs/architecture.md` there is a single repository interface for the whole app, so `action/achievements` calls `GetBalance`/`GetPersonalRecords`/`CountProgress` the same way `action/money`/`action/workout`/`action/progress` already do internally — no adapter interfaces, no duplicated queries. Where no existing method fits (`money_spend`'s category-prefix sum, `exercise_total_volume`'s volume sum), Achievements defines its own scoped query against that domain's tables, the same way the old `Budget`/`BudgetProgress` query worked directly against `transactions`. These calls now happen only from `create_achievement`/`refresh_achievements`, never from a read path (see the caching bullet above)
- **Real foreign keys across subdomains, deliberately**: `achievements.exercise_id`/`activity_id` are true `REFERENCES exercises(id)`/`REFERENCES activities(id)` — a departure from `money`'s "flat schema, no FK overhead" convention (see `money-spec.md`), justified here because an achievement pointing at a deleted exercise/activity is a real data-integrity bug, not just a display nuisance
- **`ends_at` is nullable for every type — null means no deadline, just tracked progress**: an achievement can be open-ended ("save X, whenever") or dated ("save X by December")
- **`unit` is a display-only label, for `exercise_max_weight`/`exercise_total_volume`/`activity_occurrence_count`/`activity_streak_count`/`manual` only**: free text (e.g. `"kg"`, `"sessions"`, `"books"`) shown next to the value, never used in any calculation — enforced null for `money_saving`/`money_spend` by CHECK
- **An achievement's progress window is `[starts_at, ends_at or the moment it was last computed]`**: for `money_spend` and `exercise_total_volume`, spend/volume-so-far sums activity in that window; for `money_saving`, `exercise_max_weight`, `activity_occurrence_count`, and `activity_streak_count`, the calculation evaluates as of whenever `create_achievement`/`refresh_achievements` last ran it, regardless of `ends_at` — there's no "window to close" for a point-in-time read (balance, all-time max) or an already-since-creation-bounded count. `manual`'s `current_value` has no window at all — it's just whatever it currently is. `ends_at` only gates whether an achievement still shows as *active* for a given `get_achievement_progress(date)` call — it has never gated *when* the value itself was computed
- **`get_achievement_progress`'s `date` param filters which achievements are active, not which instant their values reflect**: since `current_value` is a cached column now, `date` only feeds the `starts_at <= date AND (ends_at IS NULL OR ends_at >= date)` active-window check — it does not make the read compute progress "as of" that date. Progress is always whatever was last cached, no matter what `date` is passed
- **Web dashboard is read-mostly, one page, two sections, plus one Refresh action**: active achievements as a **tile grid** with progress bars (`webui.RenderAchievementTiles`, see `webui-spec.md`) and past/completed achievements (`ends_at` before now) as a plain **table** below — final target/deadline only, no progress bar, since the window already closed and there's nothing to show progress toward. A table reads better than tiles for a historical list that's just going to grow. Creating achievements, editing achievements, and logging manual progress stay MCP-only (`create_achievement`, `update_achievement`, `log_achievement_progress`); the one exception is a "Refresh" button (`POST /web/achievements/refresh`, see HTTP Handlers) — the web equivalent of the `refresh_achievements` MCP tool, since asking someone to open a chat just to refresh a number they're already looking at defeats the point of a dashboard
- **The same tile grid is embedded on three other pages, not re-derived**: Money, Progress browse, and Workouts each render a `webui.RenderAchievementTiles` grid pre-filtered to their own domain's `achievement_type`s (see Overview) — `AchievementFilter.Types` (nil/empty = all, used by the dedicated Achievements page and the `get_achievement_progress` MCP tool) so each embedding page only ever queries its own achievements via `ListAchievements`, not the whole table
- **`action/achievements` exports the tile-building step too, not just the data read**: `BuildAchievementTiles(ctx, db, userID, at, types)` calls `ListAchievements` and maps each `Achievement` to a `webui.AchievementTileData` (the same per-`achievement_type` progress-label/percent formatting the dedicated page uses, reading `CurrentValue` and `RemainingValue()` straight off it) in one place — Money/Progress-browse/Workouts call this directly and pass the result straight to `webui.RenderAchievementTiles`, instead of each subdomain re-implementing "how do I turn a `money_spend` achievement into a percent and a label." This is the "cross-domain reads go straight through the shared repository" principle (above) extended one level up the stack: cross-domain *rendering* reuse, not just reads
- **An embedded tile grid disappears when its domain has no achievements of its own; the dedicated page never does**: `BuildAchievementTiles` returning zero tiles means Money/Progress-browse/Workouts render nothing for that section (`AchievementTilesData.EmptyMessage` left unset — see `webui-spec.md`) rather than an empty grid with a message, since a page with no financial achievements shouldn't dedicate screen space to saying so. `GET /web/achievements` always sets `EmptyMessage` ("No active achievements yet") because on that page, achievements *are* the content — an empty state there is itself informative
- **Type-specific scalars are packed into one `details JSONB` column, not several nullable columns**: `category`, `unit`, and `baseline_balance_eur` are each populated for at most one or two of the seven `achievement_type`s, written once at `create_achievement` and never again, and are never filtered/indexed on `achievements` itself (each is read once per achievement, not searched across achievements) — exactly the sparse, write-once-case JSONB fits, instead of several near-always-null columns each with its own CHECK. `exercise_id`/`activity_id` deliberately stay out of `details` and remain real columns — see the FK bullet above, unaffected by this. `current_value` used to be here too — see the bullet above for why it moved out
- **`details`'s per-type CHECKs use key-existence, not `IS NOT NULL`**: e.g. `check_spend_category` becomes `details ? 'category'` instead of `category IS NOT NULL` — same enforcement, just phrased for a JSONB column. `current_value` needs no such CHECK at all any more — now that every `achievement_type` populates it, it's simply `NOT NULL` at the column level (see SQL DDL); the old `check_manual_counter` (type-scoped) is gone
- **`domain.Achievement` keeps typed fields — JSON packing is a storage-layer-only detail**: the repository mapper (`mapper_achievement.go`, per `docs/architecture.md`) is the only place that marshals `Category`/`Unit`/`BaselineBalanceEUR` into/out of the `details` column; `action/achievements` and the MCP tool handlers never see raw JSON, only the same typed `Achievement` struct as before this column existed. `CurrentValue` needs no such marshaling — it's a plain, always-populated `db:"current_value"` field like `TargetValue`
- **Exactly two write methods on the repository, ever: `CreateAchievement` and `UpdateAchievement`**: no `IncrementAchievementCurrentValue`, no `SetAchievementCurrentValue` — every write to an existing achievement, no matter the caller or the reason, goes through the same generic `UpdateAchievement(userID, domain.AchievementUpdate)`. `log_achievement_progress`'s "+delta" and `refresh_achievements`' recompute both *compute an absolute new value themselves* (reading the current one first, or deriving it from another subdomain) and hand it to `UpdateAchievement` as `AchievementUpdate.CurrentValue`, the same field the `update_achievement` MCP tool sets for a manual correction. This trades a theoretically-atomic `current_value = current_value + $delta` in SQL for a read-then-set in Go — a real, accepted race window (two concurrent `log_achievement_progress` calls on the same achievement could clobber each other) that doesn't matter for a single-user personal tool, in exchange for one write method instead of three
- **`updated_at` is bumped by every write path that touches an achievement row**: `create_achievement`, `update_achievement`, `log_achievement_progress`, and `refresh_achievements` all end up setting it to `NOW()` — mechanically the first is `CreateAchievement` and the other three are all `UpdateAchievement` (see the bullet above) — so one column answers "when was this achievement (or its cached progress) last touched" regardless of which of the four callers wrote it. `refresh_achievements` skipping a `manual` achievement (nothing to recompute) means it doesn't call `UpdateAchievement` for that achievement at all, so its `updated_at` doesn't move either — the column only moves when something actually changed
- **`update_achievement` (the MCP tool) exposes corrections, not structure, and not the other six types' cached value**: `name`/`target_value`/`ends_at`(+`clear_ends_at`)/`category`/`unit` are user-settable for every applicable type (same partial-update convention as `money-spec.md`'s `edit_transactions` — nil means unchanged); `current_value` is user-settable *only* when the achievement is `manual`. `achievement_type`, `exercise_id`, `activity_id`, and `baseline_balance_eur` are never user-editable — the first three would change what the achievement fundamentally *is*, and `baseline_balance_eur` is a deliberate one-time snapshot whose whole point is staying fixed. This restriction lives in the `update_achievement` MCP tool handler, not in `domain.AchievementUpdate` or `DB.UpdateAchievement` themselves (see the bullet above — those stay generic so `refresh_achievements`/`log_achievement_progress` can reuse them): a user hand-editing one of the other six types' `current_value` would just get clobbered by the next `refresh_achievements` anyway, and a mismatch between the cache and reality is a signal to refresh, not to paper over with a manual edit
- **The two all-types achievement pages group by category, embedded single-domain grids stay chronological**: `/web/achievements` and `/web/achievements/eink` both call `BuildAchievementTiles(..., types=nil)` — the only two callers that ever ask for every `achievement_type` mixed together — so `BuildAchievementTiles` treats `types == nil` as the signal to additionally sort the achievements it got back from `ListAchievements` into four category buckets, in order: `activities` (`activity_occurrence_count`, `activity_streak_count`) → `money` (`money_saving`, `money_spend`) → `gym` (`exercise_max_weight`, `exercise_total_volume`) → `others` (`manual`), `id` ascending within a bucket. Money/Progress-browse/Workouts pass a non-nil `types` subset (their own 1-2 achievement types) and are left exactly as `ListAchievements` returned them — `ORDER BY starts_at`, unchanged — since a single-domain grid has nothing to group by category and the chronological order was never a problem there. This is a Go-side sort inside `BuildAchievementTiles`, not a `ListAchievements`/SQL change, so it needs no repository or mock changes
- **Achievement tiles link to their underlying page, per `achievement_type`, where one exists**: `toAchievementTileData` sets `webui.AchievementTileData.LinkURL` from whichever of `Category`/`ExerciseID`/`ActivityID` the achievement's type populates — `money_spend` → `/web/money/transactions?category=...` (the same URL shape `action/money`'s `categoryLinkURL` builds, reconstructed inline in `action/achievements` since `action/money` already imports `action/achievements` for its embedded tile grid and importing back would cycle), `exercise_max_weight`/`exercise_total_volume` → `/web/workouts/{exercise_id}`, `activity_occurrence_count`/`activity_streak_count` → `/web/progress/browse/{activity_id}`. `money_saving` and `manual` get no link (`LinkURL: ""`) — `money_saving` has no `Category` of its own to filter by (unlike `money_spend`) and linking to the bare, unfiltered `/web/money` dashboard wasn't judged worth the special case; `manual` has no underlying page at all. This applies everywhere a tile is rendered — dedicated page, e-ink page, and all three embedded grids — since a working drill-down is useful regardless of which page shows the tile
- **Validation happens in the MCP tool handler, before any write — DB CHECKs are a backstop, never the error a caller is meant to see**: every rule that has a matching CHECK constraint (`target_value > 0`, `ends_at > starts_at`, `category` required/forbidden per `achievement_type`, `unit` forbidden on `money_saving`/`money_spend`, `exercise_id`/`activity_id` required/forbidden per `achievement_type`) is checked in Go, by `create_achievement` and `update_achievement`, *before* the `INSERT`/`UpdateAchievement` call, and fails with a specific, human-readable message naming the actual problem (e.g. "category is required for money_spend achievements," "target_value must be greater than 0," "unit is not applicable to money_saving achievements") — never a raw Postgres constraint-violation message like `check_target_value` or `check_spend_category`. `exercise_id`/`activity_id` get an extra ownership/existence check (`GetExercise`/`GetActivity`) with its own clear error ("exercise not found") rather than surfacing as a foreign-key violation. The table's CHECK constraints stay in the schema regardless — they're a second line of defense against a future bug or a write that bypasses `action/achievements` entirely, not the intended way a caller learns their input was invalid. `achievement_type` itself is validated as one of the seven known values before anything else runs

## Architecture Diagrams

### Entity Relation Diagram

```mermaid
erDiagram
    ACHIEVEMENTS }o--o| EXERCISES : "tracks metric of (exercise_max_weight|exercise_total_volume only)"
    ACHIEVEMENTS }o--o| ACTIVITIES : "reads progress points of (activity_occurrence_count|activity_streak_count only)"

    ACHIEVEMENTS {
        bigserial id PK
        bigint user_id
        varchar name "e.g. Emergency Fund, Bench Press 100kg, Read 12 Books"
        varchar achievement_type "money_saving|money_spend|exercise_max_weight|exercise_total_volume|activity_occurrence_count|activity_streak_count|manual"
        bigint exercise_id FK "exercise_max_weight|exercise_total_volume only"
        bigint activity_id FK "activity_occurrence_count|activity_streak_count only"
        decimal target_value "target amount/weight/volume/count"
        decimal current_value "cached for every achievement_type — written by create_achievement/refresh_achievements (six types) or log_achievement_progress (manual), never computed at read time"
        jsonb details "category|unit|baseline_balance_eur — shape varies by achievement_type, see Best Practices"
        timestamptz starts_at
        timestamptz ends_at "nullable — null means no deadline"
        timestamptz created_at
        timestamptz updated_at "bumped by create_achievement, update_achievement, log_achievement_progress, and refresh_achievements"
    }

    EXERCISES {
        int id PK
        string name
    }

    ACTIVITIES {
        bigint id PK
        string name
    }
```

### C4 Context Diagram

```mermaid
graph TB
    User[User/Claude MCP Client]
    Browser[Human user / browser]

    subgraph "Achievements System"
        MCP[MCP Server]
        AchievementsWeb[GET /web/achievements handler]
        BuildTiles[BuildAchievementTiles helper<br/>Achievement → webui.AchievementTileData]
        DB[(PostgreSQL Database)]

        MCP -->|SQL queries| DB
        AchievementsWeb --> BuildTiles
    end

    subgraph "Embedding pages (action/webui consumers)"
        MoneyH[action/money — GET /web/money]
        ProgressH[action/progress — GET /web/progress/browse]
        WorkoutH[action/workout — GET /web/workouts]
    end

    User -->|create_achievement| MCP
    User -->|update_achievement| MCP
    User -->|get_achievement_progress, cached read| MCP
    User -->|log_achievement_progress| MCP
    User -->|refresh_achievements, recomputes current_value| MCP
    Browser -->|GET /web/achievements| AchievementsWeb
    Browser -->|POST /web/achievements/refresh| AchievementsWeb
    Browser --> MoneyH
    Browser --> ProgressH
    Browser --> WorkoutH

    MoneyH -.->|"BuildAchievementTiles types: money_saving / money_spend"| BuildTiles
    ProgressH -.->|"BuildAchievementTiles types: activity_occurrence_count / activity_streak_count"| BuildTiles
    WorkoutH -.->|"BuildAchievementTiles types: exercise_max_weight / exercise_total_volume"| BuildTiles
    BuildTiles --> DB

    DB -.->|achievements table| DB
    DB -.->|reads: transactions table, money domain| DB
    DB -.->|reads: sets/exercises tables, workout domain| DB
    DB -.->|reads: activity_progress table, progress domain| DB

    style User fill:#e1f5ff
    style Browser fill:#e1f5ff
    style MCP fill:#ffe1e1
    style AchievementsWeb fill:#ffe1e1
    style DB fill:#e1ffe1
```

### Sequence Diagram: Create Achievement

```mermaid
sequenceDiagram
    participant User
    participant MCP
    participant DB

    User->>MCP: create_achievement(name, achievement_type, target_value, ...)
    MCP->>MCP: achievement_type must be one of the seven known values,<br/>else error "unknown achievement_type"
    MCP->>MCP: target_value > 0, else error<br/>"target_value must be greater than 0"
    MCP->>MCP: ends_at > starts_at (if set), else error<br/>"ends_at must be after starts_at"

    alt achievement_type = money_spend
        MCP->>MCP: require non-empty category, else error<br/>"category is required for money_spend achievements"
        MCP->>MCP: reject unit if set, else error<br/>"unit is not applicable to money_spend achievements"
    else achievement_type = money_saving
        MCP->>MCP: reject category/unit if set (same shape of error as above)
        MCP->>DB: GetBalance(userID, first_transaction_at, starts_at)
        DB-->>MCP: current_balance_eur
        MCP->>MCP: baseline_balance_eur = current_balance_eur
    else achievement_type = exercise_max_weight or exercise_total_volume
        MCP->>MCP: require exercise_id, reject category if set
        MCP->>DB: GetExercise(exercise_id, userID)
        DB-->>MCP: exercise, or not found —<br/>error "exercise not found" either way (missing vs. not-owned look the same)
    else achievement_type = activity_occurrence_count or activity_streak_count
        MCP->>MCP: require activity_id, reject category if set
        MCP->>DB: GetActivity(activity_id, userID)
        DB-->>MCP: activity, or not found —<br/>error "activity not found" either way
    else achievement_type = manual
        MCP->>MCP: reject category if set<br/>current_value = 0
    end

    alt achievement_type != manual
        MCP->>MCP: compute initial current_value —<br/>same per-type logic as refresh_achievements,<br/>see the "Refresh Achievements" diagram below
    end

    MCP->>DB: INSERT INTO achievements (..., current_value)
    DB-->>MCP: achievement id
    MCP-->>User: created achievement
```

### Sequence Diagram: Update Achievement

```mermaid
sequenceDiagram
    participant User
    participant MCP
    participant DB

    User->>MCP: update_achievement(achievement_id, name?, target_value?,<br/>ends_at?/clear_ends_at?, category?, unit?, current_value?)
    MCP->>DB: GetAchievement(achievement_id, userID)
    DB-->>MCP: achievement, or not found —<br/>error "achievement not found" either way

    MCP->>MCP: target_value > 0 if set, else error<br/>"target_value must be greater than 0"
    MCP->>MCP: ends_at > achievement.starts_at if set, else error<br/>"ends_at must be after starts_at"
    MCP->>MCP: category only if achievement.achievement_type = money_spend,<br/>else error "category can only be set on money_spend achievements"<br/>(and "category is required for money_spend achievements" if clearing it to empty)
    MCP->>MCP: unit only if achievement.achievement_type allows it,<br/>else error "unit is not applicable to this achievement's achievement_type"
    MCP->>MCP: current_value only if achievement.achievement_type = manual,<br/>else error "current_value can only be corrected on manual achievements"

    MCP->>DB: UpdateAchievement(userID, AchievementUpdate{...})
    DB->>DB: UPDATE achievements SET (only the provided fields), updated_at = NOW()<br/>WHERE id = achievement_id AND user_id = userID
    DB-->>MCP: ok
    MCP-->>User: updated achievement
```

### Sequence Diagram: Get Achievement Progress

```mermaid
sequenceDiagram
    participant User
    participant MCP
    participant DB

    User->>MCP: get_achievement_progress(date = now)
    MCP->>DB: ListAchievements(AchievementFilter{UserID, ActiveOnly: true, At: date, Types: nil})
    DB->>DB: SELECT * FROM achievements<br/>WHERE user_id=? AND starts_at <= date<br/>AND (ends_at IS NULL OR ends_at >= date)<br/>AND (types IS NULL OR achievement_type = ANY(types))
    DB-->>MCP: []Achievement — current_value read straight from the cached column,<br/>no per-type computation here (see "Refresh Achievements" below for how it's kept up to date)
    MCP->>MCP: for each achievement, remaining_value = achievement.RemainingValue()<br/>(plain getter, no extra query — see Best Practices)
    MCP-->>User: []Achievement, each with its remaining_value
```

### Sequence Diagram: Refresh Achievements

The only place the six derived types' `current_value` gets (re)computed — shared by `create_achievement`'s initial snapshot (see above) and the `refresh_achievements` MCP tool / `POST /web/achievements/refresh` (see below):

```mermaid
sequenceDiagram
    participant Caller as MCP or web handler
    participant DB

    alt achievement_id given
        Caller->>DB: GetAchievement(achievement_id, userID)
        DB-->>Caller: achievement — error if not found/owned,<br/>or if achievement_type = manual (nothing to recompute)
        Caller->>Caller: achievements = [achievement]
    else achievement_id omitted
        Caller->>DB: ListAchievements({UserID: userID, ActiveOnly: true})
        DB-->>Caller: active achievements
        Caller->>Caller: achievements = active achievements, excluding achievement_type = manual
    end

    loop For each achievement
        alt achievement_type = money_spend
            Caller->>DB: GetCategorySpend(userID, category, starts_at, COALESCE(ends_at, now))
            DB-->>Caller: spent_eur
            Caller->>Caller: current_value = spent_eur
        else achievement_type = money_saving
            Caller->>DB: GetBalance(userID, first_transaction_at, now)
            DB-->>Caller: current_balance_eur
            Caller->>Caller: current_value = current_balance_eur - baseline_balance_eur
        else achievement_type = exercise_max_weight
            Caller->>DB: GetPersonalRecords(userID, exercise_id)
            DB-->>Caller: PersonalRecords
            Caller->>Caller: current_value = PersonalRecords.MaxWeight.WeightKg
        else achievement_type = exercise_total_volume
            Caller->>DB: GetExerciseVolume(userID, exercise_id, starts_at)
            DB-->>Caller: total_volume_kg
            Caller->>Caller: current_value = total_volume_kg
        else achievement_type = activity_occurrence_count
            Caller->>DB: CountProgress({ActivityID: activity_id, From: starts_at})
            DB-->>Caller: occurrence count
            Caller->>Caller: current_value = count
        else achievement_type = activity_streak_count
            Caller->>DB: GetActivity(activity_id, userID)
            DB-->>Caller: activity (for FrequencyDays)
            Caller->>DB: ListProgress({ActivityID: activity_id, To: now}) — newest first
            DB-->>Caller: []ActivityPoint
            Caller->>Caller: walk newest-to-oldest, counting while<br/>each gap to the next point <= FrequencyDays —<br/>gap from now to the newest point itself must also qualify,<br/>else current_value = 0 (streak already lapsed)
        end
        Caller->>DB: UpdateAchievement(userID, AchievementUpdate{ID: achievementID, CurrentValue: &current_value})
        DB->>DB: UPDATE achievements SET current_value = $current_value, updated_at = NOW()<br/>WHERE id = $achievement_id AND user_id = $user_id
    end

    Caller-->>Caller: return count refreshed (or the single updated achievement, if achievement_id was given)
```

### Sequence Diagram: Log Manual Achievement Progress

```mermaid
sequenceDiagram
    participant User
    participant MCP
    participant DB

    User->>MCP: log_achievement_progress(achievement_id, delta = 1)
    MCP->>DB: GetAchievement(achievement_id, userID)
    DB-->>MCP: achievement
    MCP->>MCP: error if achievement_type != manual
    MCP->>MCP: new_value = achievement.current_value + delta<br/>(read-then-set in Go, no atomic DB increment — see Best Practices)
    MCP->>DB: UpdateAchievement(userID, AchievementUpdate{ID: achievement_id, CurrentValue: &new_value})
    DB-->>MCP: ok
    MCP-->>User: updated achievement (current_value = new_value, remaining_value)
```

### Sequence Diagram: Web Achievements Page

```mermaid
sequenceDiagram
    participant Browser
    participant Handler as action/achievements web handler
    participant Achievements as action/achievements BuildAchievementTiles
    participant DB
    participant Webui as action/webui

    Browser->>Handler: GET /web/achievements
    Handler->>Achievements: BuildAchievementTiles(ctx, db, userID, now, types=nil)
    Achievements->>DB: ListAchievements(AchievementFilter{UserID, ActiveOnly: true, At: now, Types: nil})<br/>— plain cached read, see "Get Achievement Progress" above
    DB-->>Achievements: []Achievement (all types)
    Achievements->>Achievements: types == nil, so sort by category bucket<br/>(activities → money → gym → others), then id ascending<br/>(see Best Practices — skipped when types is a non-nil subset)
    Achievements->>Achievements: map each to webui.AchievementTileData<br/>(Name, ProgressLabel, PercentComplete, Deadline, LinkURL, OverTarget)<br/>reading CurrentValue/RemainingValue() straight off the Achievement
    Achievements-->>Handler: []webui.AchievementTileData
    Handler->>DB: ListAchievements({ActiveOnly: false}) filtered to ends_at < now
    DB-->>Handler: past/completed achievements (no live recompute)
    Handler->>Handler: build TableData: Name, Type, Target, Deadline (past/completed section)
    Handler->>Webui: RenderAchievementTiles(active, EmptyMessage: "No active achievements yet"),<br/>RenderTable (past), RenderPage
    Webui-->>Browser: 200 text/html — page includes a "Refresh" form (see below)
```

### Sequence Diagram: Web Achievements Refresh

```mermaid
sequenceDiagram
    participant Browser
    participant Handler as action/achievements web handler
    participant Refresh as shared Refresh Achievements logic
    participant DB

    Browser->>Handler: POST /web/achievements/refresh (form submit, no body)
    Handler->>Refresh: refresh(ctx, db, userID, achievement_id=nil)
    Refresh->>DB: ListAchievements({UserID: userID, ActiveOnly: true}), then<br/>per-type recompute + UpdateAchievement(CurrentValue) for each non-manual achievement
    DB-->>Refresh: updated
    Refresh-->>Handler: count refreshed
    Handler-->>Browser: 302 redirect to GET /web/achievements
```

### Sequence Diagram: Embedded Achievement Tiles

```mermaid
sequenceDiagram
    participant Browser
    participant Handler as e.g. action/money GET /web/money
    participant Achievements as action/achievements BuildAchievementTiles
    participant DB
    participant Webui as action/webui

    Browser->>Handler: GET /web/money
    Handler->>Handler: fetch its own page data (balance, categories, ...)
    Handler->>Achievements: BuildAchievementTiles(ctx, db, userID, now,<br/>types=[money_saving, money_spend])
    Achievements->>DB: ListAchievements(AchievementFilter{UserID, ActiveOnly: true, At: now, Types: types})
    DB-->>Achievements: []Achievement (money-only)
    Achievements->>Achievements: types is a non-nil subset, so no category re-sort —<br/>keeps ListAchievements' starts_at order (see Best Practices)
    Achievements->>Achievements: map to []webui.AchievementTileData (LinkURL set for<br/>money_spend only — money_saving has no Category to link with)
    Achievements-->>Handler: []webui.AchievementTileData (may be empty)
    Handler->>Webui: RenderAchievementTiles(tiles, EmptyMessage: "")<br/>— empty Tiles + empty EmptyMessage renders nothing
    Webui-->>Handler: template.HTML fragment
    Handler->>Webui: RenderPage(..., Content: its own fragments + the achievement tiles fragment)
    Webui-->>Browser: 200 text/html — page renders normally with no achievements section if Tiles was empty
```

Progress browse (`GET /web/progress/browse`) and Workouts (`GET /web/workouts`) follow the identical shape, just with `types=[activity_occurrence_count, activity_streak_count]` and `types=[exercise_max_weight, exercise_total_volume]` respectively.

## Database Schema

### SQL DDL

`gateways/db/migrations/z_achievements.sql` (replaces `z_goals.sql`, see Best Practices). Rename preamble (table, column, check/PK/FK constraints, indexes, sequence) — each step is a no-op once done or on a fresh DB:

```sql
ALTER TABLE IF EXISTS goals RENAME TO achievements;

DO $$
DECLARE
    tbl regclass := to_regclass('achievements');
    old_name text;
    new_name text;
BEGIN
    IF tbl IS NULL THEN
        RETURN;
    END IF;

    IF EXISTS (SELECT 1 FROM pg_attribute
               WHERE attrelid = tbl AND attname = 'goal_type' AND NOT attisdropped) THEN
        ALTER TABLE achievements RENAME COLUMN goal_type TO achievement_type;
    END IF;

    FOREACH old_name IN ARRAY ARRAY[
        'check_goal_type', 'check_goal_period',
        'goals_pkey', 'goals_exercise_id_fkey', 'goals_activity_id_fkey'
    ] LOOP
        new_name := replace(replace(old_name, 'goals_', 'achievements_'), 'goal_', 'achievement_');
        IF EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = tbl AND conname = old_name) THEN
            EXECUTE format('ALTER TABLE achievements RENAME CONSTRAINT %I TO %I', old_name, new_name);
        END IF;
    END LOOP;
END $$;

ALTER INDEX IF EXISTS idx_goals_user_period RENAME TO idx_achievements_user_period;
ALTER INDEX IF EXISTS idx_goals_user_type RENAME TO idx_achievements_user_type;
ALTER SEQUENCE IF EXISTS goals_id_seq RENAME TO achievements_id_seq;
```

Then the table itself (fresh DBs):

```sql
CREATE TABLE IF NOT EXISTS achievements (
    id                    BIGSERIAL PRIMARY KEY,
    user_id               BIGINT NOT NULL,
    name                  VARCHAR(255) NOT NULL,
    achievement_type      VARCHAR(25) NOT NULL,            -- 'money_saving', 'money_spend', 'exercise_max_weight', 'exercise_total_volume', 'activity_occurrence_count', 'activity_streak_count', 'manual'
    exercise_id           BIGINT REFERENCES exercises(id), -- exercise_max_weight|exercise_total_volume only
    activity_id           BIGINT REFERENCES activities(id), -- activity_occurrence_count|activity_streak_count only
    target_value          DECIMAL(12,2) NOT NULL,
    current_value         DECIMAL(12,2) NOT NULL DEFAULT 0, -- cached for every achievement_type — see Best Practices for who writes it and when
    details               JSONB NOT NULL DEFAULT '{}',     -- remaining type-specific scalars, shape varies by achievement_type — see Best Practices:
                                                            --   money_spend: {"category": "food/cafe"}
                                                            --   money_saving: {"baseline_balance_eur": 1234.56}
                                                            --   exercise_max_weight / exercise_total_volume: {"unit"?: "kg"}
                                                            --   activity_occurrence_count / activity_streak_count: {"unit"?: "sessions"}
                                                            --   manual: {"unit"?: "books"}
    starts_at             TIMESTAMPTZ NOT NULL,
    ends_at               TIMESTAMPTZ,                     -- nullable — null means no deadline
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(), -- bumped by create_achievement (insert) and every UpdateAchievement call (update_achievement, log_achievement_progress, refresh_achievements)

    CONSTRAINT check_achievement_type CHECK (achievement_type IN (
        'money_saving', 'money_spend', 'exercise_max_weight', 'exercise_total_volume',
        'activity_occurrence_count', 'activity_streak_count', 'manual'
    )),
    CONSTRAINT check_spend_category CHECK (
        (achievement_type = 'money_spend' AND details ? 'category' AND details->>'category' <> '')
        OR (achievement_type <> 'money_spend' AND NOT (details ? 'category'))
    ),
    CONSTRAINT check_saving_baseline CHECK (
        (achievement_type = 'money_saving' AND details ? 'baseline_balance_eur')
        OR (achievement_type <> 'money_saving' AND NOT (details ? 'baseline_balance_eur'))
    ),
    CONSTRAINT check_exercise_link CHECK (
        (achievement_type IN ('exercise_max_weight', 'exercise_total_volume') AND exercise_id IS NOT NULL)
        OR (achievement_type NOT IN ('exercise_max_weight', 'exercise_total_volume') AND exercise_id IS NULL)
    ),
    CONSTRAINT check_activity_link CHECK (
        (achievement_type IN ('activity_occurrence_count', 'activity_streak_count') AND activity_id IS NOT NULL)
        OR (achievement_type NOT IN ('activity_occurrence_count', 'activity_streak_count') AND activity_id IS NULL)
    ),
    CONSTRAINT check_unit_scope CHECK (
        NOT (details ? 'unit') OR achievement_type IN (
            'exercise_max_weight', 'exercise_total_volume', 'activity_occurrence_count', 'activity_streak_count', 'manual'
        )
    ),
    CONSTRAINT check_target_value CHECK (target_value > 0),
    CONSTRAINT check_achievement_period CHECK (ends_at IS NULL OR ends_at > starts_at)
);

CREATE INDEX IF NOT EXISTS idx_achievements_user_period ON achievements(user_id, starts_at, ends_at);
CREATE INDEX IF NOT EXISTS idx_achievements_user_type   ON achievements(user_id, achievement_type);
```

`UpdateAchievement` is the only write path for an existing achievement, for every caller — the `update_achievement` MCP tool, `refresh_achievements`' recompute, and `log_achievement_progress`'s increment all build a `domain.AchievementUpdate` and go through it (see Best Practices: no separate increment/set methods). It's a dynamic partial `UPDATE` over whichever `AchievementUpdate` fields are non-nil, always an absolute `SET`, never a `current_value = current_value + $delta` — `log_achievement_progress` computes the new absolute value in Go first (`GetAchievement` then add `delta`), the same as `refresh_achievements` computes its new absolute value from other subdomains' data first:

```sql
UPDATE achievements
SET name = COALESCE($name, name),
    target_value = COALESCE($target_value, target_value),
    current_value = COALESCE($current_value, current_value),
    category = ...,     -- only touched when Category is set on the AchievementUpdate (see Best Practices for details vs. real columns)
    unit = ...,
    ends_at = CASE WHEN $clear_ends_at THEN NULL WHEN $ends_at IS NOT NULL THEN $ends_at ELSE ends_at END,
    updated_at = NOW()
WHERE id = $achievement_id AND user_id = $user_id;
```

## Go Code Structure

### Domain Models

```go
package achievements

import "time"

// AchievementType is the kind of achievement being tracked — seven flat, sibling values,
// each with its own required fields and progress calculation (see Best
// Practices). There is no shared "money"/"exercise"/"activity" grouping —
// an achievement type is never nested inside another via a second discriminator field.
type AchievementType string

const (
    AchievementTypeMoneySaving             AchievementType = "money_saving"             // reach TargetValue in accumulated balance
    AchievementTypeMoneySpend              AchievementType = "money_spend"              // don't exceed TargetValue spent on Category
    AchievementTypeExerciseMaxWeight       AchievementType = "exercise_max_weight"      // reach TargetValue kg all-time max for ExerciseID
    AchievementTypeExerciseTotalVolume     AchievementType = "exercise_total_volume"    // reach TargetValue kg total volume for ExerciseID since StartsAt
    AchievementTypeActivityOccurrenceCount AchievementType = "activity_occurrence_count" // reach TargetValue occurrences of ActivityID since StartsAt
    AchievementTypeActivityStreakCount     AchievementType = "activity_streak_count"    // reach TargetValue consecutive check-ins of ActivityID, evaluated as of now
    AchievementTypeManual                  AchievementType = "manual"                   // reach TargetValue; CurrentValue incremented directly via log_achievement_progress
)

// Achievement represents any of the seven achievement types — see Best Practices for
// which fields apply to which AchievementType. Category/Unit/BaselineBalanceEUR are
// stored packed into the achievements.details JSONB column, not their own columns —
// mapper_achievement.go marshals/unmarshals them, so this struct's shape is
// unaffected by that storage choice. CurrentValue is a real, always-populated
// column (not packed, not nullable) — cached, not computed live, see Best
// Practices for who writes it (create_achievement/refresh_achievements for six types,
// log_achievement_progress for manual) and why reads never recompute it.
type Achievement struct {
    ID                 int64      `json:"id" db:"id"`
    UserID             int64      `json:"user_id" db:"user_id"`
    Name               string     `json:"name" db:"name"`
    AchievementType           AchievementType   `json:"achievement_type" db:"achievement_type"`
    ExerciseID         *int64     `json:"exercise_id,omitempty" db:"exercise_id"`   // exercise_max_weight|exercise_total_volume only
    ActivityID         *int64     `json:"activity_id,omitempty" db:"activity_id"`   // activity_occurrence_count|activity_streak_count only
    TargetValue        float64    `json:"target_value" db:"target_value"`
    CurrentValue       float64    `json:"current_value" db:"current_value"` // cached — see Best Practices, not omitempty since every achievement_type populates it
    Category           *string    `json:"category,omitempty"`             // money_spend only — packed into details
    Unit               *string    `json:"unit,omitempty"`                 // exercise_max_weight|exercise_total_volume|activity_occurrence_count|activity_streak_count|manual only — packed into details
    BaselineBalanceEUR *float64   `json:"baseline_balance_eur,omitempty"` // money_saving only — packed into details
    StartsAt           time.Time  `json:"starts_at" db:"starts_at"`
    EndsAt             *time.Time `json:"ends_at,omitempty" db:"ends_at"` // nil means no deadline
    CreatedAt          time.Time  `json:"created_at" db:"created_at"`
    UpdatedAt          time.Time  `json:"updated_at" db:"updated_at"` // bumped by create_achievement, update_achievement, log_achievement_progress, refresh_achievements
}

// achievementDetails is the JSON shape stored in achievements.details — assembled from
// and unpacked back into Achievement's typed fields by mapper_achievement.go only.
// Never referenced outside that mapper; action/achievements works with Achievement directly.
// CurrentValue is deliberately not in here — see Best Practices.
type achievementDetails struct {
    Category           *string  `json:"category,omitempty"`
    Unit               *string  `json:"unit,omitempty"`
    BaselineBalanceEUR *float64 `json:"baseline_balance_eur,omitempty"`
}

// AchievementUpdate is a partial edit of an existing achievement — every field but ID is
// optional; nil/false means "leave unchanged" (same convention as
// money-spec.md's TransactionUpdate). It's the payload for DB.UpdateAchievement,
// the repository's only write method besides CreateAchievement — three different
// callers build one: the update_achievement MCP tool (user-facing correction —
// Name/TargetValue/EndsAt/ClearEndsAt/Category/Unit, or CurrentValue but
// only for manual achievements), refresh_achievements (CurrentValue only, for the other
// six types' recomputed snapshot), and log_achievement_progress (CurrentValue
// only, set to current+delta computed in Go — see Best Practices for why
// there's no separate atomic-increment method). Structural/one-time fields
// — AchievementType, ExerciseID, ActivityID, BaselineBalanceEUR — aren't on this
// struct at all; they're never editable, see Best Practices.
type AchievementUpdate struct {
    ID           int64      `json:"id"`
    Name         *string    `json:"name,omitempty"`
    TargetValue  *float64   `json:"target_value,omitempty"`
    EndsAt       *time.Time `json:"ends_at,omitempty"`       // set a new deadline
    ClearEndsAt  bool       `json:"clear_ends_at,omitempty"` // explicitly remove the deadline; ignored if EndsAt is also set
    Category     *string    `json:"category,omitempty"`      // money_spend only — DB CHECK rejects it otherwise
    Unit         *string    `json:"unit,omitempty"`          // exercise_max_weight|exercise_total_volume|activity_occurrence_count|activity_streak_count|manual only — DB CHECK rejects it otherwise
    CurrentValue *float64   `json:"current_value,omitempty"` // no DB-level type restriction (see Best Practices) — which achievement_type may set it is a rule each caller enforces itself
}

// RemainingValue is a plain getter, not a stored/duplicated field — every
// caller that used to read an AchievementProgress.RemainingValue (an earlier draft
// had that as its own struct wrapping Achievement) now just calls this on the Achievement
// it already has. No repository round-trip, no separate type.
func (g Achievement) RemainingValue() float64 {
    return g.TargetValue - g.CurrentValue // negative means over/exceeded
}

// AchievementFilter defines query parameters for listing achievements — the one read path
// for more than one achievement at a time (see Best Practices: there is no
// separate GetAchievementProgress, ListAchievements covers it with these two extra fields).
type AchievementFilter struct {
    UserID     int64
    ActiveOnly bool        // starts_at <= At AND (ends_at IS NULL OR ends_at >= At)
    At         time.Time   // moment ActiveOnly's window is evaluated against; zero value means now
    Types      []AchievementType  // nil/empty = every type; non-empty scopes to a subset (e.g. Money's embedded tile grid)
}
```

### Repository Interface

```go
type DB interface {
    // Write
    CreateAchievement(ctx context.Context, g *domain.Achievement) (int64, error)

    // UpdateAchievement is the only way to write to an existing achievement — the entire
    // repository has exactly two write methods, this and CreateAchievement (see
    // Best Practices). A dynamic partial update over whichever
    // domain.AchievementUpdate fields are non-nil/true (achievementID is AchievementUpdate.ID,
    // not a separate parameter), bumping updated_at. Always an absolute
    // SET, never a delta — a caller that needs "+delta" semantics (only
    // log_achievement_progress does) reads the current value first and computes
    // the new absolute value itself; atomicity of that read-then-set isn't
    // a concern for a single-user personal tool (see Best Practices). This
    // method does no business validation of its own — every caller (the
    // update_achievement MCP tool, refresh_achievements, log_achievement_progress) validates
    // and builds a well-formed AchievementUpdate before calling it, per the
    // "validate in the tool, not the DB" bullet in Best Practices. The
    // table's own CHECK constraints (per-achievement_type field applicability,
    // target_value > 0, ends_at > starts_at) still apply underneath and
    // will reject a malformed write, but that is a backstop against a bug,
    // never the caller's expected/only source of a validation error.
    // CurrentValue has no per-type CHECK at all (every achievement_type populates
    // it now — see Best Practices) — it's each caller's own job (the
    // update_achievement MCP tool for manual corrections, refresh_achievements for the
    // other six, log_achievement_progress for manual's increment) to only set it
    // when appropriate for that achievement's type.
    UpdateAchievement(ctx context.Context, userID int64, update domain.AchievementUpdate) error

    // Read
    GetAchievement(ctx context.Context, achievementID int64, userID int64) (*domain.Achievement, error)

    // ListAchievements is the one query method for more than one achievement — no
    // separate GetAchievementProgress (see Best Practices). ActiveOnly+At covers
    // what a "progress as of a date" read needs (starts_at <= At AND
    // (ends_at IS NULL OR ends_at >= At)), Types scopes to a subset of
    // AchievementType. A plain read of already-cached current_value columns, no
    // per-achievement_type computation — that only happens in create_achievement/
    // refresh_achievements (see Best Practices). Callers get remaining_value by
    // calling Achievement.RemainingValue() on each result, not from this method.
    ListAchievements(ctx context.Context, filter domain.AchievementFilter) ([]domain.Achievement, error)

    // GetCategorySpend sums transactions.amount_eur where category starts
    // with the given prefix, within [from, to] — powers money_spend achievements.
    // Same query the old Budget/BudgetProgress used, now achievement-scoped.
    GetCategorySpend(ctx context.Context, userID int64, category string, from, to time.Time) (float64, error)

    // GetExerciseVolume sums sets.weight_kg * sets.reps for an exercise
    // since a given time — powers exercise_total_volume achievements.
    GetExerciseVolume(ctx context.Context, userID int64, exerciseID int64, since time.Time) (float64, error)

    // Cross-domain reads used directly inside create_achievement's initial
    // snapshot and refresh_achievements' recompute, not redefined here — see
    // money-spec.md's GetBalance, workout-spec.md's GetPersonalRecords, and
    // progress-spec.md's CountProgress/GetActivity/ListProgress (the latter
    // two power activity_streak_count's walk — see Best Practices, no new
    // progress-domain method needed). All live on the same shared
    // repository (docs/architecture.md), so action/achievements calls them the
    // same way their own actions do. Never called from ListAchievements —
    // see the caching bullet in Best Practices.
}
```

## MCP Tools

### create_achievement
Creates a new achievement. Required: `name`, `achievement_type` (one of the seven values — errors with an unknown-type message if it isn't), `target_value` (errors "target_value must be greater than 0" if not). Type-specific requirements, each with its own clear error if missing/misapplied: `money_spend` requires non-empty `category` ("category is required for money_spend achievements") and rejects `unit`; `money_saving` auto-captures `baseline_balance_eur` via `GetBalance` as of `starts_at` (not caller-supplied) and also rejects `category`/`unit`; `exercise_max_weight`/`exercise_total_volume` require `exercise_id`, checked for existence/ownership via `GetExercise` ("exercise not found") and reject `category`; `activity_occurrence_count`/`activity_streak_count` require `activity_id`, checked via `GetActivity` ("activity not found") and reject `category`; `manual` initializes `current_value` to 0 and rejects `category`. `exercise_max_weight`/`exercise_total_volume`/`activity_occurrence_count`/`activity_streak_count`/`manual` accept an optional `unit` display label; the other two reject it ("unit is not applicable to money_saving/money_spend achievements"). `ends_at` is optional for every type (omit for no deadline); when provided, errors "ends_at must be after starts_at" if it isn't. All of this validates in Go before any DB write — see Best Practices. For the six non-`manual` types, also computes and stores an initial `current_value` snapshot (same logic as `refresh_achievements`, see Best Practices) so the achievement isn't stuck at zero until the first refresh — a backdated achievement can even start out already met.

### update_achievement
Edits mutable fields of an existing achievement by ID: `name`, `target_value`, `ends_at` (or `clear_ends_at` to remove the deadline), and, when they apply to the achievement's own `achievement_type`, `category` (money_spend only) and `unit`. `current_value` is only editable this way for `manual` achievements — a direct correction (e.g. "actually I've read 15 books not 12"), distinct from `log_achievement_progress`'s delta-based `+1`. For the other six types, `current_value` is cached/derived and not editable via `update_achievement` at all — use `refresh_achievements` to bring it up to date instead (see Best Practices). All fields optional except `achievement_id`; only provided fields change. `achievement_type`, `exercise_id`, `activity_id`, and `baseline_balance_eur` are not editable at all. Validates the same rules as `create_achievement`, with the same specific error messages, for whichever fields are provided — "achievement not found" if `achievement_id` doesn't exist or isn't owned by the user, "target_value must be greater than 0", "ends_at must be after starts_at", "category can only be set on money_spend achievements" (and the reverse — required, not just settable, when the achievement is money_spend), "unit is not applicable to `<achievement_type>` achievements", "current_value can only be corrected on manual achievements" — never a raw DB constraint error (see Best Practices). Bumps `updated_at`.

### refresh_achievements
Recomputes and persists `current_value` for the six derived achievement types (`money_saving`/`money_spend`/`exercise_max_weight`/`exercise_total_volume`/`activity_occurrence_count`/`activity_streak_count`) — the same per-`achievement_type` calculation `create_achievement` uses for its initial snapshot, run again on demand instead of automatically (see Best Practices for why this is opt-in, not live). Optional `achievement_id`: omitted refreshes every active achievement owned by the user; provided refreshes just that one achievement — errors "achievement not found" if it doesn't exist or isn't owned, or "manual achievements have no derived progress to refresh; use log_achievement_progress or update_achievement instead" if it's `manual`. Returns the count of achievements refreshed, or the single updated achievement when `achievement_id` is given. Bumps `updated_at` on every achievement it actually recomputes.

### get_achievement_progress
Returns all achievements active as of a given date (defaults to now, i.e. `starts_at <= date` and `ends_at` null or `>= date`), each with `remaining_value` from `Achievement.RemainingValue()` — a plain getter over the cached `current_value` column, no per-`achievement_type` computation (see Best Practices; call `refresh_achievements` first if the numbers might be stale). Always calls `ListAchievements` with `Types: nil` (every type) — the MCP tool has no type filter of its own; that's a web-only need (see HTTP Handlers).

### log_achievement_progress
Increments an `achievement_type = manual` achievement's `current_value` by `delta` (defaults to 1), returning the new value. Errors "achievement not found" if `achievement_id` doesn't exist or isn't owned by the user, or "log_achievement_progress only applies to manual achievements" if the achievement is one of the other six types.

## HTTP Handlers

### GET /web/achievements
**Auth**: `WebMiddleware` session cookie (see `auth-spec.md`), same as every other `/web/*` dashboard

Read-mostly, single page, built on the shared `action/webui` design system (see `webui-spec.md`):
- A **"Refresh" button** at the top of the page — a plain `<form method="POST" action="/web/achievements/refresh">` with one submit button, rendered locally by `action/achievements`' own page template (not a shared `webui` component, same convention as `money-spec.md`'s export checkbox form) — see `POST /web/achievements/refresh` below
- **Active achievements** tile grid via `BuildAchievementTiles(ctx, db, userID, now, types=nil)` + `webui.RenderAchievementTiles` — every achievement type, each tile showing name, progress bar, label (e.g. `"€420 / €1,000 (42%)"`, `"82kg / 100kg"`, `"3200kg / 5000kg"`, `"12 / 30 sessions"`, `"14 / 30 day streak"`), and deadline when set. Values come straight from the cached `current_value` column (see Best Practices) — this page load never recomputes anything itself, only "Refresh" does. `EmptyMessage` is always set ("No active achievements yet")
- **Past/completed achievements** table below (achievements with `ends_at < now`): Name, Type, Target, Deadline — no recompute, since the window already closed
- No add/edit form otherwise — creating and editing achievements, and logging manual progress, stay MCP-only (`create_achievement`, `update_achievement`, `log_achievement_progress`)

### POST /web/achievements/refresh
**Auth**: same `WebMiddleware` session cookie

The one write route among the `/web/*` dashboards — every other one (Money, Progress browse, Workouts) is strictly read-only/GET-only, see `money-spec.md`/`workout-spec.md` — the web equivalent of the `refresh_achievements` MCP tool with no `achievement_id` (refreshes every active achievement owned by the user; see the "Refresh Achievements" and "Web Achievements Refresh" sequence diagrams above). Takes no body/params — the "Refresh" button on `/web/achievements` is the only caller. On success, redirects (`302`) back to `GET /web/achievements`, so the page reloads showing the freshly recomputed values.

### GET /web/achievements/eink
**Auth**: none — unauthenticated, same as `GET /web/progress` (see `progress-spec.md`)

Purpose-built fixed-viewport (`100vw`/`100vh`), black-and-white, monospace screenshot page for a physical e-ink display, mirroring `action/progress/dashboard_web.go`'s `/web/progress`. Content is just the active-achievements tile grid — `BuildAchievementTiles(ctx, db, userID, now, types=nil)` (grouped by category, see Best Practices) + `webui.RenderAchievementTiles`, restyled black-and-white by `action/achievements`' own inline CSS (same `webui-achievement-tile*` classes, no shared design-system stylesheet) — no Refresh form, no past/completed table. Tuned for its own low card count rather than reusing `/web/progress`'s dense sizing: larger name/label/deadline font sizes than the shared design system's default, and rounded card corners (`border-radius`) instead of the sharp `border: 1px solid #000` a dense multi-row layout called for.

### GET /web/goals/eink (legacy redirect)
**Auth**: none

Pre-rename URL of the e-ink page, kept because a physical display is configured with it: `301` to `/web/achievements/eink`, query string preserved (`achievements.LegacyEinkRedirectWebHandler`).

### Embedded achievement tiles on Money, Progress browse, and Workouts
Each of `GET /web/money`, `GET /web/progress/browse`, and `GET /web/workouts` (see `money-spec.md`, `progress-spec.md`, `workout-spec.md`) calls `BuildAchievementTiles` with its own domain's `types` and embeds the resulting `webui.RenderAchievementTiles` fragment on its existing page — no new route. `EmptyMessage` is left unset on all three, so a domain with no achievements of its own renders no achievements section at all rather than an empty grid (see Best Practices).

### `achievements.BuildAchievementTiles(ctx context.Context, db DB, userID int64, at time.Time, types []domain.AchievementType) ([]webui.AchievementTileData, error)`
Not a route — an exported Go function, the single place that turns an `Achievement` into a `webui.AchievementTileData`: calls `ListAchievements(AchievementFilter{UserID: userID, ActiveOnly: true, At: at, Types: types})` (a plain cached read, see Best Practices); when `types` is `nil` (the two all-types pages, `/web/achievements` and `/web/achievements/eink`), additionally re-sorts the result into the four category buckets described in Best Practices before mapping — a non-nil `types` subset skips this and keeps `ListAchievements`'s `starts_at` order. For each achievement it then formats `ProgressLabel`/`PercentComplete` per `achievement_type` from its already-computed `CurrentValue`/`RemainingValue()` — pure display formatting, no further computation or DB calls — sets `Deadline` from `EndsAt` (empty if nil), sets `LinkURL` per `achievement_type` (see Best Practices), and sets `OverTarget` when `CurrentValue > TargetValue` for `money_spend` only (the one type where exceeding is a bad thing). Called by `GET /web/achievements` and `GET /web/achievements/eink` (both with `types=nil`) and by the Money/Progress-browse/Workouts handlers (each with their own `types` subset) — see the "Embedded Achievement Tiles" sequence diagram above.

## Configuration

- **Log Achievement Progress Default Delta**: 1 (one call = one occurrence, for `log_achievement_progress` when `delta` is omitted)
- **Money Achievement Progress**: transfers excluded from `money_spend` spent calculation (same as `get_balance`)
- **Exercise Achievement Volume**: only sets with both `weight_kg` and `reps` set count toward `exercise_total_volume` (same restriction `get_personal_records` already applies)
- **Activity Streak Lookback**: `activity_streak_count` fetches progress points via `ListProgress` with no `Limit` cap — a personal-tool activity's total history is small enough that walking the full set is cheap; revisit only if a single activity's point count grows unusually large

## E2E Tests

Rename only — same scenarios, same assertions, new names:

- `tests/goals_test.go` → `tests/achievements_test.go`, `tests/goals_dashboard_web_test.go` → `tests/achievements_dashboard_web_test.go`; every `Test*Goal*` → `Test*Achievement*` (incl. `TestMoneyDashboard_EmbedsOwnAchievementTiles`, `TestProgressBrowse_EmbedsOwnAchievementTiles`, `TestWorkoutsDashboard_EmbedsOwnAchievementTiles`, `TestDeleteActivity_BlockedByAchievementReference`)
- Assertions follow the new routes (`/web/achievements*`), JSON fields (`achievement_id`, `achievement_type`), CSS classes (`webui-achievement-tile*`) and error text
- `tests/docs_test.go`, `tests/webui_design_system_test.go`: update any "goal" strings they assert on
- New: `TestAchievementsEink_LegacyGoalsURLRedirects` — `GET /web/goals/eink?x=1` returns `301` with `Location: /web/achievements/eink?x=1`
