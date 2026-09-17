# Backlog

List of ideas for future work. Each idea must be turned into a feature document (`docs/functions/{subdomain}-spec.md`) before implementation starts, per the AI-Driven Development Convention.

Each item is headed by the date it was added (DD-MM-YY), not a sequence number — that way removing a done item never forces renumbering the rest. When one item depends on another, reference it by date + title. Items are listed in rough priority order (top = next), not by date.

## 16-09-26 — MCP tool to edit activity's progress_type (and remap existing point values)

Extend `edit_activity` (or add a dedicated tool) to let `progress_type` of an existing activity be changed after creation, together with a way to rewrite the `value` of its existing `activity_progress` points to match the new type's -2..+2 semantics (see `get_progress_type_examples`) instead of leaving them stale under the old type's meaning.

**Why:** `edit_activity` currently can't touch `progress_type` at all (`edit_activity_mcp.go` only accepts name/description/frequency_days/life_part_ids/started_at/ended_at) — activities created under the wrong type can't be reclassified without losing history. Real case: "Менторинг и консультации" and "Разговаривать с мамой и сестрой" are filed as `habit_progress` but are recurring commitments that fit `promise_state` much better. Goal is to give the agent a tool to do the reclassification itself, including remapping existing point values, not just a raw column update.

## 16-09-26 — Activity status field (active/paused/finished/dropped) + deferred_until

Add an explicit `status` column to `activities` (`active | paused | finished | dropped`) instead of inferring state purely from `started_at`/`ended_at`, plus a `deferred_until` timestamp usable when an activity is paused. Paused activities disappear from the e-ink dashboard (`dashboard_web.go`) and are shown in their own separate section on both the web browse view (`browse_web.go`) and MCP's `get_activity_list`.

**Why:** `ended_at` alone can't distinguish "finished" (goal reached) from "dropped" (abandoned) — same field today, but different real outcomes with different meaning for stats. There's also no way to mark an in-progress activity as paused without abusing `started_at`/`ended_at`. Came out of a brainstorm on activities having too many implicit states packed into two timestamp fields.

## 16-09-26 — Add progress_points via web interface

Add a web form/route to create `activity_progress` points directly from the browser (e.g. from `/web/progress/browse/{id}`), instead of requiring the MCP tool `create_progress_point` every time.

**Why:** Right now the only way to log a check-in is through the MCP tool via the agent — a plain web form, same write-then-redirect pattern as `POST /web/goals/refresh`, would let quick logging happen without opening a chat session.

## 16-09-26 — Formalize activities & finance workflow, render as web doc, expose via MCP for session context

Write up the actual process/conventions for how activities (`action/progress`) and finances (`action/money`) are meant to be used day-to-day — what the user does manually vs. what the agent does — as documentation, render it on the web, and expose it through MCP (e.g. a resource or a `get_workflow_docs`-style tool) so it can be injected into agent sessions as context.

**Why:** Goal is mostly for the user's own clarity — working through this system keeps surfacing confusion about what belongs where (activity vs. idea vs. journal note, which progress_type fits what, etc.); writing the process down forces that decision, and exposing it to the agent keeps every session consistent with it instead of re-deriving conventions ad hoc each time.

## 16-09-26 — Reconsider life_parts: unused in MCP tools, web UI, and actual usage

`life_parts` (categorization table + `life_part_ids` on activities) exists in the schema and can be set via `create_activity`/`edit_activity`, but nothing reads or filters by it in the web browse view or in any MCP tool output, and it isn't part of the user's actual workflow. Decide whether to build real usage (filtering, display, stats grouped by life part) or drop the concept entirely.

**Why:** Unused categorization is dead weight — it should either earn its place with real filtering/display, or be removed rather than left as schema noise nobody looks at.

## 16-09-26 — Ideas table + MCP tools (separate from activities)

Add a new table (e.g. `ideas`: id, user_id, title, description, created_at, updated_at) plus MCP tools (`create_idea`, `edit_idea` to append/grow the description over time, `list_ideas`) for unformed thoughts that aren't ready to become an activity — no `progress_type`, no `frequency_days`, no progress points. Not auto-converted into an activity — promoting an idea is a manual action (create a new activity referencing the idea's text); the idea stays in its list afterward as a historical record.

**Why:** Ideas don't fit the `activities` schema — `progress_type` and `frequency_days` are `NOT NULL`/`CHECK`-constrained, so a not-yet-decided idea would need fake defaults to live there. A separate table keeps the activities list to genuinely committed, trackable things, and lets an idea's description accumulate freely across multiple sessions without periodic check-in semantics. Came out of a brainstorm on distinguishing "not yet thought through" (idea) from "decided but not started" (activity with future `started_at`) and "started then paused" (see the activity status backlog item above).
