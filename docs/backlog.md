# Backlog

List of ideas for future work. Each idea must be turned into a feature document (`docs/functions/{subdomain}-spec.md`) before implementation starts, per the AI-Driven Development Convention.

Each item is headed by the date it was added (DD-MM-YY), not a sequence number — that way removing a done item never forces renumbering the rest. When one item depends on another, reference it by date + title. Items are listed in rough priority order (top = next), not by date.

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

## 23-09-26 — Enforce activity/achievement limits in create_activity / create_achievement

Check the WIP limit (max 6 active activities total, max 3 per `progress_type`) in `create_activity`, and the max-6 cap in `create_achievement`, instead of leaving both counts to be tallied by hand at each weekly review.

**Why:** `action/docs/content/activity-rituals.md` (§2.6, §2.10) treats these limits as load-bearing rules ("новое — только ценой вытеснения"), currently enforced only by the agent counting rows during review — a tool-level check makes the limit hold even outside a review session, instead of depending on the agent remembering to check.

## 25-09-26 — Mechanics/rituals docs for food, workout, finance

Write `action/docs/content/{subject}-mechanics.md` and/or `-rituals.md` for food, workout, and finance, same shape as the activities pair — mechanics normative against the code, rituals covering the actual day-to-day process (when/why, not just what fields exist).

**Why:** `transport/mcp/instructions.md` was trimmed to a dispatcher (subdomain → first tool call) for every subdomain, including food/workout/finance — but unlike activities, they have no `get_doc` fallback yet, so anything beyond "which tool to call first" that used to live in the old instructions text (metaphor scripts, exact wording, detailed procedure) is currently just gone until this is written.

## 27-09-26 — Personal documents: editable, versioned, stored outside the repo

User-owned markdown documents kept in the database instead of the codebase, editable via MCP and the web, with every edit saved as a new version. First two documents: a personal development plan and personal instructions (how the agent should work with the user, preferences, rules).

- A document: slug, title, current body, created_at, updated_at.
- A version per edit: document id, body snapshot, created_at, optional short change note. Old versions are read-only; no hard delete of history.
- MCP: list documents, read one (current or a given version), create, edit (full replace or append), list a document's versions.
- Web: list and read documents, edit form, version history with a diff against the previous version.

**Why:** `action/docs` (see `docs-spec.md`) serves only hand-written convention docs embedded via `go:embed` — changing them is a commit + deploy, and they are shared project content, not personal data. A development plan and personal instructions change often, are edited mid-conversation by the agent or by the user from the phone, and don't belong in the git history of the code. Versioning keeps the "commit history" benefit (what changed in the plan and when) without living in the repo.

**Use cases:**
- The agent pulls the personal instructions at the start of a session, the same way it pulls `get_doc` today.
- Update the development plan after a monthly/quarterly review and later see how it changed over time.
- Edit a document from the web without going through chat.

**Open question:** decide in the feature doc whether this extends `action/docs` (one `list_docs`/`get_doc` over both embedded and DB-stored documents) or is a separate subdomain with its own tools.

## 02-10-26 — Contacts CRM: collecting information about people

New subdomain for a personal contacts CRM — a place to keep what is known about people the user interacts with, instead of keeping it in memory or scattered across chats.

- A contact: name, short description (who this person is, how the user knows them), created_at, updated_at.
- Notes about a contact: free-text facts and interaction notes added over time (what was discussed, interests, important dates), each with created_at.
- MCP: create/edit a contact, add a note, search contacts and notes, read a contact with its notes.
- Web: list and read contacts with their notes.

**Why:** Information about people (what they do, what was discussed last time, what matters to them) currently lives nowhere structured, so it gets lost between conversations. A dedicated subdomain lets the agent save facts mid-conversation and pull them back up before the next meeting or message.

**Use cases:**
- After a meeting or call, tell the agent what was learned about the person and have it saved as a note.
- Before meeting someone, ask the agent for everything known about them.
- Find a person by a remembered detail ("who was the one working on Kubernetes at ...").

**Open question:** decide in the feature doc which fields are structured (birthday, contacts/links, company, tags) and which stay as free-text notes.

## 02-10-26 — User profile with settings

Store users in the database and give each user a profile with their own settings, instead of keeping everything hardcoded for a single implicit user.

- A user: id, name, created_at — the first step to having users as real rows in the DB rather than an implicit single owner.
- Settings: user-configurable values that are currently fixed in code or docs — first of all the WIP limits (max active activities total, max per `progress_type`, max achievements), plus any other tunable parameters found along the way.
- Manifest: a per-user manifest stored in the profile. What exactly it contains and how it is used is to be defined later.
- MCP and web: read and change own settings and manifest.

**Why:** Limits like the WIP caps are personal choices, not universal rules, so they should be adjustable by the user without a code change. Having users in the DB is also the base for anything per-user later (settings, manifest, personal data).

**Depends on:** 23-09-26 — Enforce activity/achievement limits in create_activity / create_achievement (the enforced limits should read their values from the user's settings once this exists).

**Open question:** define what the manifest is and how the agent uses it; decide in the feature doc whether it overlaps with 27-09-26 — Personal documents (e.g. stored as one of those documents instead of a profile field).
