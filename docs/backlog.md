# Backlog

List of ideas for future work. Each idea must be turned into a feature document (`docs/functions/{subdomain}-spec.md`) before implementation starts, per the AI-Driven Development Convention.

Each item is headed by the date it was added (DD-MM-YY), not a sequence number — that way removing a done item never forces renumbering the rest. When one item depends on another, reference it by date + title. Items are listed in rough priority order (top = next), not by date.

## 02-11-26 — Learning: active word review modes instead of plain flashcards

Today a word review on `/web/learning/:id/review` is a flashcard: the user sees the term, opens the answer and grades themselves with one of four buttons. Replace it with review modes where the user produces or picks the answer and the app checks it and sets the score:

- **Match pairs**: several due words at once, each term matched to its translation from a shuffled list (a dropdown per term, no JS)
- **Type the translation**: term shown, the user types the translation
- **Type the term**: translation shown, the user types the term (production, harder)
- **Fill the gap**: a sentence with the word left out, the user types the missing form

Each mode is a different kind of practice, so the review keeps its mode, and the score comes from the check (e.g. exact → 5, right apart from an accent or letter case → 4, wrong → 1), not from self-grading. Mastery and the weekly cycle stay as they are.

Open questions for the feature document: who picks the mode (the agent at `plan_vocabulary`, or a rule like recognition modes for new words and production modes for strong ones); where gap sentences come from (written by the agent when planning, per review, or stored as word examples); how strict the check is for Greek accents and for answers with several valid translations; whether the flashcard stays as one of the modes.

**Why:** a self-graded flashcard is passive recognition and easy to overrate; typing or picking the answer is active recall, closer to real use, and it gives honest scores, so mastery means more.

**Use cases:**
- New words: match pairs and type the translation, to learn to recognize them
- Words the user knows: type the term from the translation, fill the gap in a sentence
- Weekly analysis: the agent sees which mode a word failed in, not only that it failed

**Depends on:** `docs/functions/learning-spec.md` (implemented)

## 02-11-26 — Learning: pronunciation audio for words

A play button next to a word on the flashcard / review page, the vocabulary list and the word page, so the user hears how the term is pronounced (mainly Greek; terms of a plan without a language, like Kubernetes, have no audio).

Two ways to do it, to choose in the feature document:

- **Text-to-speech in the browser** (Web Speech API, `speechSynthesis` with `lang="el-GR"`): free, nothing stored, works for any word right away. But it needs client-side JS, an exception to the "no client-side JS" rule for these pages, and the voice depends on the device: macOS/iOS have a Greek voice, other systems may have none or a poor one, so the same word can sound different on phone and laptop.
- **Audio files prepared on the server** with a TTS API (e.g. Google Cloud TTS, Azure, OpenAI): one stable, good voice everywhere, played with a plain `<audio controls>` element, no JS. Costs a paid API, an API key env var, and storage: generate when a word is added or edited (term changed → regenerate), or on the first request and cache it; store in the database or in file storage.

Open questions: which language a plan's terms are in (a `language` field on the plan, which also tells whether audio applies at all); whether to also voice gap sentences and example phrases; what happens when the TTS API is down (the word saves without audio and gets it later).

**Why:** words are learned from text only, so the user doesn't know how a word sounds and memorizes a wrong pronunciation, especially the stress, which Greek marks but which is easy to ignore when reading.

**Use cases:**
- Hear a new word on its first flashcard
- Replay a word in the vocabulary list before speaking practice
- A future "listen and type the word" review mode

**Depends on:** `docs/functions/learning-spec.md` (implemented); related to 02-11-26 — Learning: active word review modes instead of plain flashcards (a listening mode)

## 23-09-26 — Enforce activity/achievement limits in create_activity / create_achievement

Check the WIP limit (max 6 active activities total, max 3 per `progress_type`) in `create_activity`, and the max-6 cap in `create_achievement`, instead of leaving both counts to be tallied by hand at each weekly review.

**Why:** the document «Ритуалы и процесс» (§2.6, §2.10) treats these limits as load-bearing rules ("новое — только ценой вытеснения"), currently enforced only by the agent counting rows during review — a tool-level check makes the limit hold even outside a review session, instead of depending on the agent remembering to check.

## 25-09-26 — Mechanics/rituals docs for food, workout, finance

Write mechanics and/or rituals documents for food, workout, and finance through `create_doc` / `create_doc_chunk`, same shape as the activities pair («Механика системы», «Ритуалы и процесс») — mechanics normative against the code, rituals covering the actual day-to-day process (when/why, not just what fields exist).

**Why:** `transport/mcp/instructions.md` was trimmed to a dispatcher (subdomain → first tool call) for every subdomain, including food/workout/finance — but unlike activities, they have no `get_doc` fallback yet, so anything beyond "which tool to call first" that used to live in the old instructions text (metaphor scripts, exact wording, detailed procedure) is currently just gone until this is written.

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

**Open question:** define what the manifest is and how the agent uses it; decide in the feature doc whether it overlaps with documents (`docs/functions/docs-spec.md`), e.g. stored as one of those documents instead of a profile field.

## 13-10-26 — Rework life_parts into Values

Rename and rework `life_parts` into `values` — the user's personal values — and make them a first-class, editable entity instead of a read-only tag list.

- A value: name, description, created_at, updated_at. Replaces the `life_parts` table; existing rows are migrated.
- Full MCP for the user's values: create, edit, delete and list (replaces the read-only `list_life_parts`).
- Activity ↔ value link: replaces `activities.life_part_ids` with a link that carries its own free-text description of how the activity supports, advances or protects that value. Existing `life_part_ids` are migrated into links.
- `create_activity` / `edit_activity` / `get_activity_list` and the Progress web pages switch from life part ids to value links with their descriptions.

**Why:** Life parts can only be inserted by hand via SQL and attach to an activity as a bare id, so the link says nothing about why the activity matters. Values are what «Ритуалы и процесс» actually reasons about ("Ценности: Влияние, Мастерство, Независимость, Здоровье, Близкие"); making them editable and requiring each activity to state how it serves a value turns the link into something reviews can check, not just a category tag.

**Use cases:**
- Add, rename or reword a value in conversation with the agent, without a SQL insert.
- When creating an activity, write down how it supports, advances or protects each value it is linked to.
- At a review, read an activity together with its value links and decide whether it still serves them; see which values have no active activity.

**Open question:** decide in the feature doc whether the link description is required or optional, and whether the kind of contribution (supports / advances / protects) is a structured field or stays inside the free text.

## 17-10-26 — Rework OAuth on an embedded KV store (like my_saas)

Rework the OAuth flow in `action/auth/oauth.go` so it works fully, keeping its temporary state in an embedded KV store the same way `my_saas` does (`pkg/kvstore` on `dgraph-io/badger`, see `my_saas/docs/specification/database/kv.md` and `internal/usecase/appauth`).

- Embedded KV store (badger) with TTL as a gateway, behind an interface, with JSON get/set helpers — ported from `my_saas/pkg/kvstore`.
- Authorization codes move from the in-memory `AuthCodes` map to the KV store with a TTL, single-use (deleted on exchange).
- PKCE: store `code_challenge` with the code and verify `code_verifier` at the token endpoint (S256), since it is advertised in the well-known metadata.
- Real dynamic client registration: `RegisterClientHandler` creates and stores a new client (id, secret, redirect URIs) instead of always returning the hardcoded Claude client; `redirect_uri` is checked against the registered ones.
- Refresh tokens: short-lived access token plus a stored, rotating refresh token (`refresh_token` grant) instead of a single 14-day JWT.

**Why:** All OAuth state lives in process memory, so every restart or deploy drops in-flight codes, and the flow only "works" because registration hands every client the same hardcoded credentials. PKCE and redirect URIs are not checked, and when the 14-day token expires the MCP connector has to be re-authorized by hand. `my_saas` already solves this with a zero-ops embedded KV, so the same approach can be reused here without adding Redis.

**Open question:** decide in the feature doc which data goes to KV with TTL (codes, auth state) and which belongs in Postgres (registered clients, refresh tokens), as in `my_saas` where refresh tokens live in the `devices` table.

## 01-11-26 — Move the web UI to a PWA

Turn the web pages into an installable Progressive Web App, so they open from the phone's home screen like a native app instead of as a browser tab.

- Web app manifest: name, icons, theme color, `display: standalone`, linked from the shared `webui` shell.
- Service worker: registered from the shell, served from the root path so its scope covers all pages; caches static assets.
- Mobile fit: the shared shell and `webui` components work in standalone mode (safe areas, no browser back button).

**Why:** The web pages are used on the go, for example logging sets during a workout, and reaching them through a browser tab is slower than opening an app. A PWA gives an app-like entry point without building and maintaining a native client.

**Open question:** decide in the feature doc whether the service worker is an accepted exception to the "no client-side JS" rule for web pages, and how much offline support is needed: only caching static assets and an offline page, or also queuing writes such as logged sets.

## 01-11-26 — New glanceable dashboard for e-ink displays and a desk tablet

Come up with a new visualization for the e-ink displays and for a tablet hanging on the desk: one shared dashboard that shows the most important things at once, or rotates through them one at a time.

- Decide what is worth showing: choose across subdomains rather than mirroring one list per page.
- Design it for both devices: e-ink (black and white, slow refresh, fixed viewport) and the tablet (color, can rotate content).
- Add a gamified layer that makes following through on plans visible and rewarding.
- Replaces the current `/web/progress` and `/web/achievements/eink` screens if the new concept covers them.

**Why:** The goal is to not forget, not give up, and actually do what was planned, with the execution gamified. The e-ink displays were built for this same goal, but the concept of the system has changed a lot since then, so they no longer seem useful and the user has stopped looking at them.

**Open question:** decide in the feature doc what the concept is before any screens: which information earns a place, whether it is one screen or a rotation, what the game mechanic is, and whether the rotation is an accepted exception to the "no client-side JS" rule for web pages.

## 01-11-26 — Dynamic transaction parsing rules in Lua

Move transaction import rules out of Go code into Lua scripts stored in the user's profile settings, so the user can tell their agent to add a new import, with its column parsing and category rules, without any Go change or deploy.

- Account types as a profile setting: each account type carries its own Lua CSV parser. The accounts supported on import come from settings instead of the hardcoded list in `ParserFor` and the import form.
- CSV parsing in Lua: the script for an account type turns a bank export's columns into raw transactions. It replaces the Go `RevolutParser` and `BankOfCyprusParser`, whose logic is ported to Lua.
- Category rules in Lua: the script that turns a merchant and description into a category replaces the hardcoded `InferCategory` rules in `action/money/parser.go`.
- MCP: the agent can read, edit and revert these scripts and add a new account type, so a bad change can be rolled back.
- Import: `action/money/import_web.go` reads the scripts' content from the profile and executes them.

**Why:** Every new bank, export format, merchant or category fix currently needs a code change and a deploy. Keeping the rules as per-user scripts lets the agent handle them in conversation, right when a new export or a wrong category shows up, and the rules become personal data instead of being compiled into the app.

**Depends on:** 02-10-26 — User profile with settings (the scripts and the account types are settings tied to the profile).

**Open question:** decide in the feature doc how the scripts are sandboxed (no file or network access, time and memory limits), what import does when a script fails, how versions are stored for revert (reuse the docs approach or a separate history), what the Lua contract is (does Go read the CSV and pass rows, or does the script get the raw file), how the agent tests a script on a sample file before saving it, whether merchant recognition (`RecognizeMerchant`) and transaction type rules move to Lua too, and whether category rules are one script per user or per account type.
