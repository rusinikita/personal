# Backlog

List of ideas for future work. Each idea must be turned into a feature document (`docs/functions/{subdomain}-spec.md`) before implementation starts, per the AI-Driven Development Convention.

Each item is headed by the date it was added (DD-MM-YY), not a sequence number — that way removing a done item never forces renumbering the rest. When one item depends on another, reference it by date + title. Items are listed in rough priority order (top = next), not by date.

## 21-09-26 — Web UI for logging workout sets (new page under Workouts)

Add workout-logging web pages under the existing Workouts section:

- **History page** (default landing): last 10 workouts, each showing its date, and per exercise the list of exercises done plus the first set logged for that exercise.
- **"New workout" button**: does not create a workout row itself — it just navigates to a `new` screen.
- **Workout screen**: shows the set-adding form — exercise selector sorted by frequency of use (most-used exercise first), plus inputs for difficulty/weight and rep count.
- **Lazy workout creation**: a workout row is only actually created when the first set is submitted from the `new` screen (mirroring how `log_workout_set` today only creates a new workout on the first exercise logged that day/session) — at that point the page redirects from the `new` path to the path with the real workout ID.

**Why:** Sets can currently only be logged via MCP/chat; a lightweight web form lets the user log sets directly mid-workout (e.g. from their phone at the gym) without going through the agent, while keeping the "workout is created lazily on first set" behavior consistent with the existing MCP tool instead of pre-creating empty workouts that never get sets.

**Use cases:**
- Glance at the last 10 workouts (date, exercises, first set of each) without opening chat.
- Start a workout screen mid-session and only have it become a real workout once a set is actually logged.
- Pick the next exercise quickly from a frequency-sorted list instead of scanning the full exercise list.

## 16-09-26 — Ideas table + MCP tools (separate from activities)

Add a new table (e.g. `ideas`: id, user_id, title, description, created_at, updated_at) plus MCP tools (`create_idea`, `edit_idea` to append/grow the description over time, `list_ideas`) for unformed thoughts that aren't ready to become an activity — no `progress_type`, no `frequency_days`, no progress points. Not auto-converted into an activity — promoting an idea is a manual action (create a new activity referencing the idea's text); the idea stays in its list afterward as a historical record.

**Why:** Ideas don't fit the `activities` schema — `progress_type` and `frequency_days` are `NOT NULL`/`CHECK`-constrained, so a not-yet-decided idea would need fake defaults to live there. A separate table keeps the activities list to genuinely committed, trackable things, and lets an idea's description accumulate freely across multiple sessions without periodic check-in semantics. Came out of a brainstorm on distinguishing "not yet thought through" (idea) from "decided but not started" (activity with future `started_at`) and "started then paused" (see the activity status backlog item above).

## 21-09-26 — New subdomain: learning (learning_plan, skill, learning_exercise, vocabulary)

New subdomain (`action/learning`) for structured, long-running study tracking — first two use cases are learning Greek and learning Kubernetes, which is why the model needs to cover both a language (vocabulary-heavy) and a technical skill (exercise/practice-heavy) without forcing one shape onto the other.

- `learning_plan`: the top-level course/track being followed (e.g. "Greek A1", "Kubernetes fundamentals") — name, description/goal, created_at, status.
- `skill`: a specific topic or competency within a plan (e.g. "present tense verb conjugation", "pods & deployments") — name, learning_plan_id, some notion of proficiency/mastery level, created_at.
- `learning_exercise`: a practice attempt/drill tied to a skill (e.g. a conjugation drill, a `kubectl` exercise) — logs that practice happened and how it went, analogous to how `set` logs a workout attempt against an `exercise`.
- `vocabulary`: term entries for language learning specifically (Greek word, translation, notes) — likely tied to `learning_plan_id` and/or `skill_id`, with some review/recall tracking (last reviewed, confidence) given the spaced-repetition nature of vocabulary.

**Why:** Currently there's no structured way to track progress through a language or a technical skill — this would live either nowhere or awkwardly inside the generic `activities`/progress-point model, which doesn't capture skills, exercises, or vocabulary as first-class things. A dedicated subdomain models the actual shape of studying: a plan broken into skills, skills practiced via exercises, and (for language specifically) a growing vocabulary list with recall tracking.

**Use cases:**
- Track a Greek study plan broken into skills (grammar topics, tenses) plus a running vocabulary list with recall/review state.
- Track a Kubernetes study plan broken into skills (networking, workloads, storage) practiced via hands-on exercises, without a vocabulary component.
- See progress per skill within a plan rather than one flat undifferentiated activity.

## 21-09-26 — Split notes out of activity_progress into their own table

Move `activity_progress.note` (free-text, variable-length) out of the progress-point row into a separate table referencing the progress point it belongs to, instead of a column mixed in with `activity_progress`'s otherwise narrow, structured numeric/timestamp data.

**Why:** `activity_progress` should stay a uniform table of point-per-progress_type values; a prose-heavy `note` column sitting alongside it doesn't belong there and blocks giving notes their own index strategy. A dedicated notes table can carry a full-text index (`search_progress_notes` today does a plain `ILIKE` scan) or a vector/embedding index for semantic search, without either concern touching the core points table.

**Use cases:**
- Full-text or semantic (vector) search over notes without scanning/indexing the whole `activity_progress` table.
- Keep `activity_progress` lean if it ever needs its own indexing/partitioning strategy independent of note content.

## 23-09-26 — Separate entity for inbox notes

Introduce a dedicated entity for inbox notes — quick captures that land somewhere first and get sorted later, instead of being forced into an existing entity (activity, progress note, idea) at the moment of capture. The concrete representation (table shape, fields, whether/how an inbox note gets processed into another entity, MCP tools / web UI) is intentionally left undecided and will be defined when this item is turned into a feature document.

**Why:** Capturing a thought today means immediately deciding where it belongs; a separate inbox entity lets capture stay fast and pushes the "what is this?" decision to a later review step.

## 23-09-26 — Rename goals to achievements

Rename the `goals` subdomain to "achievements" everywhere it surfaces — table, `goal_type`, domain models, `action/goals` package, MCP tools (`create_goal`, `update_goal`, `get_goal_progress`, `log_goal_progress`, `refresh_goals`), web routes/pages (`/web/goals`, embedded tiles), and `docs/functions/goals-spec.md`. Pure rename — no behavior change.

**Why:** "Goal" causes confusion: these aren't life goals, they're a gamification tool — measurable targets (save X, lift X kg, N-day streak) whose point is the satisfaction of hitting them. Calling them achievements matches what they actually are and frees "goal" from implying something they don't model.

## 23-09-26 — Soft delete for progress points, with resolution tracking

Add a `deleted_at` + `resolution` field to progress points (values: `dropped`, `merged`, `expired`, `→ step`/`→ activity`/`→ note` with a reference id), replacing today's hard `delete_progress_point`. Deleted points are excluded from `get_activity_stats` and `search_progress_notes` by default.

**Why:** The weekly/monthly review ritual (`action/docs/content/activity-rituals.md`, §2.4) requires every inbox note to leave a recorded outcome when it's cleared out — dropped, merged into a duplicate, promoted into a step/activity/note, or expired after three monthly reviews with no traction — but the current hard delete destroys that decision instead of recording it.

## 23-09-26 — Duplicate-surfacing counter on progress points

Add a counter field to progress points that tracks how many times a duplicate idea has resurfaced, instead of tallying it informally in the note's text.

**Why:** The inbox ritual's spike trigger ("idea surfaced 3 times") and someday-expiry rule ("3 monthly reviews with no new bump") both key off this count (`action/docs/content/activity-rituals.md`, §2.3, §2.5) — without a real field, detecting either mechanically means re-parsing note history instead of reading one value.

## 23-09-26 — Enforce activity/goal limits in create_activity / create_goal

Check the WIP limit (max 6 active activities total, max 3 per `progress_type`) in `create_activity`, and the max-6 cap in `create_goal`, instead of leaving both counts to be tallied by hand at each weekly review.

**Why:** `action/docs/content/activity-rituals.md` (§2.6, §2.10) treats these limits as load-bearing rules ("новое — только ценой вытеснения"), currently enforced only by the agent counting rows during review — a tool-level check makes the limit hold even outside a review session, instead of depending on the agent remembering to check.

## 25-09-26 — Mechanics/rituals docs for food, workout, finance

Write `action/docs/content/{subject}-mechanics.md` and/or `-rituals.md` for food, workout, and finance, same shape as the activities pair — mechanics normative against the code, rituals covering the actual day-to-day process (when/why, not just what fields exist).

**Why:** `transport/mcp/instructions.md` was trimmed to a dispatcher (subdomain → first tool call) for every subdomain, including food/workout/finance — but unlike activities, they have no `get_doc` fallback yet, so anything beyond "which tool to call first" that used to live in the old instructions text (metaphor scripts, exact wording, detailed procedure) is currently just gone until this is written.

## 25-09-26 — Sidebar navigation between docs on /web/docs pages

Add a sidebar to `GET /web/docs/{topic}` pages listing every available doc (from `docs.Topics()`, current one highlighted), so the user can jump between docs directly instead of going back to the `/web/docs` index each time.

**Why:** With two docs today and more planned (see "25-09-26 — Mechanics/rituals docs for food, workout, finance"), mechanics and rituals docs cross-reference each other constantly — switching between them via the index page is an extra round-trip for every jump.
