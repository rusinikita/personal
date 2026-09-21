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

## 16-09-26 — Formalize activities & finance workflow, render as web doc, expose via MCP for session context

Write up the actual process/conventions for how activities (`action/progress`) and finances (`action/money`) are meant to be used day-to-day — what the user does manually vs. what the agent does — as documentation, render it on the web, and expose it through MCP (e.g. a resource or a `get_workflow_docs`-style tool) so it can be injected into agent sessions as context.

**Why:** Goal is mostly for the user's own clarity — working through this system keeps surfacing confusion about what belongs where (activity vs. idea vs. journal note, which progress_type fits what, etc.); writing the process down forces that decision, and exposing it to the agent keeps every session consistent with it instead of re-deriving conventions ad hoc each time.

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
