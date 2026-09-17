# Backlog

List of ideas for future work. Each idea must be turned into a feature document (`docs/functions/{subdomain}-spec.md`) before implementation starts, per the AI-Driven Development Convention.

Each item is headed by the date it was added (DD-MM-YY), not a sequence number — that way removing a done item never forces renumbering the rest. When one item depends on another, reference it by date + title. Items are listed in rough priority order (top = next), not by date.

## 16-09-26 — Formalize activities & finance workflow, render as web doc, expose via MCP for session context

Write up the actual process/conventions for how activities (`action/progress`) and finances (`action/money`) are meant to be used day-to-day — what the user does manually vs. what the agent does — as documentation, render it on the web, and expose it through MCP (e.g. a resource or a `get_workflow_docs`-style tool) so it can be injected into agent sessions as context.

**Why:** Goal is mostly for the user's own clarity — working through this system keeps surfacing confusion about what belongs where (activity vs. idea vs. journal note, which progress_type fits what, etc.); writing the process down forces that decision, and exposing it to the agent keeps every session consistent with it instead of re-deriving conventions ad hoc each time.

## 16-09-26 — Reconsider life_parts: unused in MCP tools, web UI, and actual usage

`life_parts` (categorization table + `life_part_ids` on activities) exists in the schema and can be set via `create_activity`/`edit_activity`, but nothing reads or filters by it in the web browse view or in any MCP tool output, and it isn't part of the user's actual workflow. Decide whether to build real usage (filtering, display, stats grouped by life part) or drop the concept entirely.

**Why:** Unused categorization is dead weight — it should either earn its place with real filtering/display, or be removed rather than left as schema noise nobody looks at.

## 16-09-26 — Ideas table + MCP tools (separate from activities)

Add a new table (e.g. `ideas`: id, user_id, title, description, created_at, updated_at) plus MCP tools (`create_idea`, `edit_idea` to append/grow the description over time, `list_ideas`) for unformed thoughts that aren't ready to become an activity — no `progress_type`, no `frequency_days`, no progress points. Not auto-converted into an activity — promoting an idea is a manual action (create a new activity referencing the idea's text); the idea stays in its list afterward as a historical record.

**Why:** Ideas don't fit the `activities` schema — `progress_type` and `frequency_days` are `NOT NULL`/`CHECK`-constrained, so a not-yet-decided idea would need fake defaults to live there. A separate table keeps the activities list to genuinely committed, trackable things, and lets an idea's description accumulate freely across multiple sessions without periodic check-in semantics. Came out of a brainstorm on distinguishing "not yet thought through" (idea) from "decided but not started" (activity with future `started_at`) and "started then paused" (see the activity status backlog item above).
