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

## 25-09-26 — Inbox: separate entity for the inbox → someday → spike lifecycle

New entity (table name to be decided together with the schema — not `notes`; plus MCP tools and a web capture form) implementing the inbox mechanics from `action/docs/content/activity-rituals.md` §2.1–2.5 and the inbox steps of the rituals (§3.2 step 1–2, §3.3 step 2, §3.4 step 2). It replaces the «Inbox» service activity from §2.2 (not created yet), where each thought would have been a `value=0` progress point. Other kinds of notes (notes on activities, diary, retro) are not part of this item.

**Data requirements:**

> ⚠️ The schema below is a draft that the user rejected. When implementing, first agree on the database schema with the user (a separate step at the start of Stage 1, before the rest of the feature doc). Don't take the fields below as a given.

- `id`, `user_id`, `created_at`, `updated_at`.
- `type`: `idea` | `spike`.
    - `idea`: a captured thought, outside existing activities (§2.2).
    - `spike`: the outcome of a spike on an idea (§2.5). It has the 4 fixed sections: what changes in 3 months if done / what happens if not / cost, what gets displaced (WIP limit) / first physical step.
- `body`: the user's own text, required. No-go findings get appended to the idea's `body` (§2.5).
- `idea_id`: for `spike` only, a required FK to the idea it examined.
- `status`, for `idea` only:
    - `inbox`: captured, not reviewed yet.
    - `someday`: stayed in the inbox after a weekly review (§2.3).
    - `resolved`: a decision has been made.
    - The weekly review takes all `inbox`, the monthly review all `someday`, so the "since last retro" boundary through activity 70 isn't needed.
- Resolution, replacing deletion (§2.4); set only when `status = resolved`:
    - `resolution`: `dropped` | `merged` | `expired` | `to_step` | `to_activity` | `to_note`.
    - `resolved_at`.
    - The target as a typed nullable FK:
        - `resolved_to_note_id`: for `merged`, the older idea.
        - `resolved_to_step_id`: for `to_step`.
        - `resolved_to_activity_id`: for `to_activity` (go after a spike).
        - `resolved_to_progress_point_id`: for `to_note`, the note in the relevant activity.
    - Resolved ideas are hidden from lists by default. The quarterly review reads them by `resolved_at` for the quarter, to count inbox conversion. There is no hard delete.
- Duplicate counter (§2.3):
    - `surface_count`: default 1. A duplicate gives the old idea +1, and the new one is resolved as `merged` into it. At 3 it becomes a spike trigger (§2.5).
    - `last_surfaced_at`: updated on +1. It is the basis for `expired` candidates: 3 monthly reviews without promotion and without a new +1.
- Constraints:
    - `status`, `surface_count`, `last_surfaced_at` are only for `idea`.
    - `idea_id` is only for `spike`.
    - `resolution` + `resolved_at` are set if and only if `status = resolved`.
- Indexes on `(user_id, type, status)` and on `idea_id`.

**Tools (to be specified in the feature doc):**
- Capture an idea: a web form (§3.1) plus MCP, for when the user dictates it (§5.1).
- List by status.
- Append to an idea's `body`.
- Move to `someday`.
- Merge a duplicate.
- Resolve with a target.
- Record a spike outcome.

**Open question:** in §2.5 a spike is a step on «Inbox». Without that activity a step has nothing to attach to. Decide in the feature doc where the spike's slot/timebox lives (a calendar event only, or something else).

**Why:** `activity_progress` is an event log. Editing and deleting its points is an exception, so soft delete with a resolution, a duplicate counter, and appending to text don't fit it. Inbox ideas have their own lifecycle (capture → review → someday → spike → decision) and get an entity built for that instead of a hack on points. After implementation, update §2.2–2.5 of the rituals doc and the «Inbox» row in `activity-mechanics.md`.

**Use cases:**
- Quick capture of a thought via the web, without choosing an activity.
- Weekly review: every `inbox` idea gets a decision. The agent suggests duplicates to merge and spike candidates (`surface_count` ≥ 3).
- Go / no-go on last week's spike, from its `spike` note.
- Monthly review: the whole `someday` list, `expired` candidates by `last_surfaced_at`.
- Quarterly review: inbox conversion by `resolution` for the quarter.

## 23-09-26 — Rename goals to achievements

Rename the `goals` subdomain to "achievements" everywhere it surfaces — table, `goal_type`, domain models, `action/goals` package, MCP tools (`create_goal`, `update_goal`, `get_goal_progress`, `log_goal_progress`, `refresh_goals`), web routes/pages (`/web/goals`, embedded tiles), and `docs/functions/goals-spec.md`. Pure rename — no behavior change.

**Why:** "Goal" causes confusion: these aren't life goals, they're a gamification tool — measurable targets (save X, lift X kg, N-day streak) whose point is the satisfaction of hitting them. Calling them achievements matches what they actually are and frees "goal" from implying something they don't model.

## 23-09-26 — Enforce activity/goal limits in create_activity / create_goal

Check the WIP limit (max 6 active activities total, max 3 per `progress_type`) in `create_activity`, and the max-6 cap in `create_goal`, instead of leaving both counts to be tallied by hand at each weekly review.

**Why:** `action/docs/content/activity-rituals.md` (§2.6, §2.10) treats these limits as load-bearing rules ("новое — только ценой вытеснения"), currently enforced only by the agent counting rows during review — a tool-level check makes the limit hold even outside a review session, instead of depending on the agent remembering to check.

## 25-09-26 — Mechanics/rituals docs for food, workout, finance

Write `action/docs/content/{subject}-mechanics.md` and/or `-rituals.md` for food, workout, and finance, same shape as the activities pair — mechanics normative against the code, rituals covering the actual day-to-day process (when/why, not just what fields exist).

**Why:** `transport/mcp/instructions.md` was trimmed to a dispatcher (subdomain → first tool call) for every subdomain, including food/workout/finance — but unlike activities, they have no `get_doc` fallback yet, so anything beyond "which tool to call first" that used to live in the old instructions text (metaphor scripts, exact wording, detailed procedure) is currently just gone until this is written.

## 25-09-26 — Sidebar navigation between docs on /web/docs pages

Add a sidebar to `GET /web/docs/{topic}` pages listing every available doc (from `docs.Topics()`, current one highlighted), so the user can jump between docs directly instead of going back to the `/web/docs` index each time.

**Why:** With two docs today and more planned (see "25-09-26 — Mechanics/rituals docs for food, workout, finance"), mechanics and rituals docs cross-reference each other constantly — switching between them via the index page is an extra round-trip for every jump.
