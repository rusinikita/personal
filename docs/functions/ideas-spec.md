# Ideas - Complete Specification

## Overview

An **idea** is a raw thought captured outside any existing activity. It lives through the inbox lifecycle from `action/docs/content/activity-rituals.md` §2.1–2.5:

```
capture → inbox → weekly review ─┬─ resolved (dropped / merged / promoted)
                                 └─ someday ── monthly review ─┬─ resolved (dropped / expired)
                                                               └─ stays someday
inbox | someday ── trigger fired, chosen at weekly step 9 ──→ spike
spike → next weekly review ─┬─ go: resolved (promoted), displacing an activity if WIP is full
                            ├─ needed, weaker than every current activity: resolved (blocked)
                            ├─ not needed: resolved (dropped)
                            ├─ no answer: someday
                            └─ spike not done: someday, or stays spike if chosen again
resolved (blocked) ── monthly portfolio check ─┬─ slot freed: re-resolved as promoted, no new spike
                                               └─ re-resolved as dropped / expired
```

A spike is earned by one of three triggers (§2.5, trigger 3 is new):
1. The idea surfaced 3 times (`surface_count ≥ 3`, computed).
2. The idea carries an explicit risk to an area of life (health, injury, family).
3. The idea hits a goal the current portfolio covers poorly: a track or 2-year goal with no active activity while a slot is taken by an activity tied to no goal (§3.4 step 1), a different approach to a track whose activity is stuck (§2.8), or a direct hit on a goal of the year while current activities serve it weakly. It is checked against the activity list and «0. Зачем», not by weighing the benefit (that is the spike's job).

Triggers aren't stored: at spike choice (§3.2 step 9) the agent flags trigger 1 from `surface_count`, and the user names ideas with triggers 2 and 3.

It replaces the planned «Inbox» service activity (§2.2), where each thought would have been a `value=0` progress point. Nothing is ever hard-deleted: leaving the list means getting a `resolution`.

Interfaces:
- **Web**: `GET /web/ideas` has a quick capture form (§3.1) plus the `inbox` ideas as cards. `POST /web/ideas` creates an idea. `GET /web/ideas/spike` shows the ideas being spiked this week. `GET /web/ideas/search` finds ideas in any status.
- **MCP**: `create_idea` (when the user dictates, §5.1), `list_ideas`, `search_ideas` (find similar ideas by text), `update_idea` (append text, change status), `resolve_idea`.

Out of scope: notes on activities, the diary, retro summaries. Those stay progress points.

## Architecture Diagrams

### Entity Relation Diagram

```mermaid
erDiagram
    IDEAS |o..o{ IDEAS : "merged into (merged_into_id)"
    ACTIVITY_PROGRESS |o..o{ IDEAS : "promotes (resolved_progress_point_id)"
    ACTIVITIES ||--o{ ACTIVITY_PROGRESS : "points"
    ACTIVITY_PROGRESS |o..o{ STEPS : "creates (created_by_progress_point_id)"

    IDEAS {
        bigint id PK
        bigint user_id
        text body "user's own words, appended over time"
        varchar status "inbox | someday | spike | resolved"
        varchar resolution "NULL unless resolved"
        bigint merged_into_id FK "only for merged"
        bigint resolved_progress_point_id FK "only for promoted"
        timestamp resolved_at "NULL unless resolved"
        timestamp created_at
        timestamp updated_at
    }
```

#### Entities and Tables

| Entity | Table | Role here |
|---|---|---|
| Idea | `ideas` (new) | A raw thought outside existing activities, with its inbox lifecycle |
| Progress point | `activity_progress` (existing) | The point a `promoted` idea became: a check-in on an existing activity or the initial point of a new one |
| Activity | `activities` (existing) | Reached through the point's `activity_id`, not linked from `ideas` |
| Step | `steps` (existing) | Reached through `steps.created_by_progress_point_id` of the promoting point, not linked from `ideas` |

#### Idea Statuses

| `status` | Meaning | Reviewed at | Next |
|---|---|---|---|
| `inbox` | Captured, not reviewed yet | Weekly review (§3.2 step 1) | `someday`, `spike` or `resolved` |
| `someday` | Kept after a weekly review: waits for a new +1, a spike or expiry; also where an idea goes when a spike gave no answer or wasn't done | Monthly review (§3.3 step 2); spike choice (§3.2 step 9) | `spike` or `resolved` (`dropped`, `expired`) |
| `spike` | A trigger fired and the idea was chosen for a spike this week | Next weekly review, go / no-go (§3.2 step 2) | `resolved` (go: `promoted`; not needed: `dropped`; needed, not now: `blocked`) or `someday` |
| `resolved` | A decision has been made; terminal, hidden from lists by default | Quarterly review, by `resolved_at` (§3.4 step 2) | — |

#### Idea Resolutions

Set only when `status = resolved`.

| `resolution` | When | Link |
|---|---|---|
| `dropped` | Decided not to do it, at any review, including "not needed" after a spike | — |
| `merged` | A duplicate: the new idea is merged into the older one, which gets +1 to `surface_count` | `merged_into_id` → the older idea |
| `expired` | A `someday` idea or a `blocked` one with no movement for more than 3 months, closed at a monthly review | — |
| `blocked` | After a spike: needed, but weaker than every current activity while the WIP limit is full. Can later be re-resolved as `promoted`, `dropped` or `expired` | — |
| `promoted` | Became progress: a step or note on an existing activity, or a new activity after a go or from `blocked` | `resolved_progress_point_id` → the point; NULL if that point was later deleted |

"No answer" after a spike is not a resolution: the idea goes back to `someday`, and the findings are already in `body`.

### C4 Context Diagram

```mermaid
graph TB
    User((User))
    Agent[AI agent via MCP]

    subgraph App[personal app]
        Web["GET /web/ideas<br/>POST /web/ideas<br/>GET /web/ideas/spike<br/>GET /web/ideas/search"]
        MCP["create_idea<br/>list_ideas<br/>search_ideas<br/>update_idea<br/>resolve_idea"]
        Repo[Repository]
    end

    DB[(PostgreSQL: ideas)]
    Progress[(steps / activities / activity_progress)]

    User -->|quick capture, search| Web
    User -->|dictates, reviews| Agent
    Agent --> MCP
    Web --> Repo
    MCP --> Repo
    Repo --> DB
    Repo -->|resolve_idea target check| Progress
```

## Database Schema

### SQL DDL

```sql
-- gateways/db/migrations/z_ideas.sql
CREATE TABLE IF NOT EXISTS ideas (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    body TEXT NOT NULL CHECK (body <> ''),
    status VARCHAR(20) NOT NULL DEFAULT 'inbox' CHECK (status IN ('inbox', 'someday', 'spike', 'resolved')),
    resolution VARCHAR(20) CHECK (resolution IN ('dropped', 'merged', 'expired', 'promoted', 'blocked')),
    merged_into_id BIGINT,
    resolved_progress_point_id BIGINT,
    resolved_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_idea_merged_into FOREIGN KEY (merged_into_id) REFERENCES ideas(id),
    CONSTRAINT fk_idea_resolved_point FOREIGN KEY (resolved_progress_point_id) REFERENCES activity_progress(id) ON DELETE SET NULL,
    -- resolution and resolved_at are set iff status = resolved
    CONSTRAINT check_idea_resolution CHECK ((status = 'resolved') = (resolution IS NOT NULL)),
    CONSTRAINT check_idea_resolved_at CHECK ((status = 'resolved') = (resolved_at IS NOT NULL)),
    -- merged_into_id iff merged; point link only for promoted (NULL after the point is deleted)
    CONSTRAINT check_idea_merged_into CHECK (COALESCE(resolution = 'merged', FALSE) = (merged_into_id IS NOT NULL)),
    CONSTRAINT check_idea_resolved_point CHECK (resolved_progress_point_id IS NULL OR resolution = 'promoted')
);

CREATE INDEX IF NOT EXISTS idx_ideas_user_status ON ideas(user_id, status);
CREATE INDEX IF NOT EXISTS idx_ideas_merged_into_id ON ideas(merged_into_id) WHERE merged_into_id IS NOT NULL;
```

## Go Code Structure

### Domain Models

```go
// domain/idea.go

type IdeaStatus string

const (
    IdeaStatusInbox     IdeaStatus = "inbox"
    IdeaStatusSomeday   IdeaStatus = "someday"
    IdeaStatusSpike     IdeaStatus = "spike"
    IdeaStatusResolved  IdeaStatus = "resolved"
)

type IdeaResolution string

const (
    IdeaResolutionDropped    IdeaResolution = "dropped"
    IdeaResolutionMerged     IdeaResolution = "merged"
    IdeaResolutionExpired    IdeaResolution = "expired"
    IdeaResolutionPromoted   IdeaResolution = "promoted"
    IdeaResolutionBlocked    IdeaResolution = "blocked"
)

type Idea struct {
    ID            int64           `json:"id" db:"id"`
    UserID        int64           `json:"-" db:"user_id"`
    Body          string          `json:"body" db:"body" jsonschema:"The user's own words; spike outcomes are appended"`
    Status        IdeaStatus      `json:"status" db:"status" jsonschema:"inbox, someday, spike or resolved"`
    Resolution    *IdeaResolution `json:"resolution,omitempty" db:"resolution" jsonschema:"Set only when status is resolved"`
    MergedIntoID  *int64          `json:"merged_into_id,omitempty" db:"merged_into_id" jsonschema:"Older idea this one was merged into (resolution merged)"`
    ResolvedProgressPointID *int64 `json:"resolved_progress_point_id,omitempty" db:"resolved_progress_point_id" jsonschema:"Progress point the idea was promoted into (resolution promoted); its activity and created steps show what it became"`
    ResolvedAt    *time.Time      `json:"resolved_at,omitempty" db:"resolved_at"`
    CreatedAt     time.Time       `json:"created_at" db:"created_at"`
    UpdatedAt     time.Time       `json:"updated_at" db:"updated_at"`
    // Read-only, computed by ListIdeas/GetIdea from ideas merged into this one
    SurfaceCount   int       `json:"surface_count" db:"surface_count" jsonschema:"1 + number of duplicates merged into this idea; 3+ is a spike trigger"`
    LastSurfacedAt time.Time `json:"last_surfaced_at" db:"last_surfaced_at" jsonschema:"Latest created_at of this idea and its merged duplicates"`
}

// IdeaFilter defines query parameters for listing ideas
type IdeaFilter struct {
    UserID       int64
    Statuses     []IdeaStatus // empty = every open status (all but resolved)
    Resolutions  []IdeaResolution // only resolved ideas with these resolutions, e.g. [blocked] for the portfolio check
    ResolvedFrom *time.Time   // resolved_at >= ResolvedFrom
    ResolvedTo   *time.Time   // resolved_at < ResolvedTo
    Limit        int          // > 0: only the Limit newest ideas, ordered by created_at DESC; 0 = all, created_at ASC
}

// IdeaSearchFilter defines parameters for a single-variant body search
type IdeaSearchFilter struct {
    UserID   int64
    Query    string       // required, ILIKE substring on body
    Statuses []IdeaStatus // empty = every status, resolved included
}

// IdeaResolve is the input of ResolveIdea
type IdeaResolve struct {
    IdeaID        int64
    UserID        int64
    Resolution    IdeaResolution
    MergedIntoID  *int64
    ResolvedProgressPointID *int64
    ResolvedAt    time.Time
}
```

### Repository Interface

```go
// Ideas
CreateIdea(ctx context.Context, idea *domain.Idea) (int64, error)
GetIdea(ctx context.Context, ideaID int64, userID int64) (*domain.Idea, error) // fills SurfaceCount/LastSurfacedAt
ListIdeas(ctx context.Context, filter domain.IdeaFilter) ([]domain.Idea, error) // fills SurfaceCount/LastSurfacedAt; ordered by created_at ASC, or the Limit newest DESC when Limit > 0
SearchIdeas(ctx context.Context, filter domain.IdeaSearchFilter) ([]domain.Idea, error) // one variant; fills SurfaceCount/LastSurfacedAt; the action merges variants and counts match_count
UpdateIdea(ctx context.Context, idea *domain.Idea) error // body, status, updated_at; caller has already merged the change onto a fetched idea
ResolveIdea(ctx context.Context, resolve domain.IdeaResolve) error // sets status=resolved + resolution fields (overwrites them when re-resolving a blocked idea); for merged also re-points ideas merged into IdeaID to MergedIntoID, in one transaction
```

## MCP Tools

### create_idea
Captures an idea in `inbox` with the user's own words as `body`. Used when the user dictates a thought (§5.1).

### list_ideas
Lists the user's ideas with `surface_count` and `last_surfaced_at`. Optional `statuses` (default: every open status), `resolutions` (e.g. `blocked` for the monthly portfolio check) and `resolved_from` / `resolved_to` for the quarterly review.

### search_ideas
Searches the user's ideas by `body` with 1-5 `query_variants` (ILIKE), optional `statuses` (default: every status, resolved included). Returns ideas with `surface_count`, `last_surfaced_at`, `resolution`, `merged_into_id` and `match_count`, ranked by `match_count` DESC, then `created_at` DESC. Same pattern as `search_progress_notes`.

### update_idea
Appends text to an idea's `body` (`append_body`) and/or changes `status` along the allowed transitions. Fails on a `resolved` idea or a transition that is not allowed.

### resolve_idea
Resolves an idea in any open status with a `resolution`, or re-resolves a `blocked` idea as `promoted`, `dropped` or `expired`. `blocked` is accepted only from `spike`. `merged` requires `merged_into_id` (own, unresolved, not itself). `promoted` requires `resolved_progress_point_id`, checked to exist and belong to the user. `dropped` / `expired` take neither.

## HTTP Handlers

### GET /web/ideas
Capture form (one textarea) on top, links «Spike ideas» to `GET /web/ideas/spike` and «Search ideas» to `GET /web/ideas/search`, then a grid of `inbox` idea cards, newest first: body, `<footer>` with `×N` on the left when surface count > 1 and created date on the right. Nav item «Ideas».

### POST /web/ideas
Creates an idea in `inbox` from the form's `body` (empty → re-render with an error), then `303` redirect to `GET /web/ideas`.

### GET /web/ideas/spike
Grid of `spike` idea cards, newest first, same cards as the main page: body, `<footer>` with `×N` on the left when surface count > 1 and created date on the right. Link back to `/web/ideas`.

### GET /web/ideas/search
Search input joined with a «Search» button (`q`, comma-separated phrases). Empty `q` → the 50 newest ideas in any status. Non-empty `q` → same search as `search_ideas` over every status, ranked by match count, then newest first; more than 5 phrases → inline error. Cards: body, `<footer>` with status, resolution and `×N` on the left and created date on the right. Link back to `/web/ideas`.

## Configuration

- **Append separator**: `"\n\n"` between the existing `body` and the appended text

## E2E Tests

`tests/ideas_test.go` (MCP) and `tests/ideas_web_test.go` (HTTP), table-driven, state asserted through the repository only.

- `TestCreateIdea`: created in `inbox` with body; empty body fails
- `TestListIdeas`: default hides `resolved`; `statuses` filter; `resolutions: [blocked]` returns only blocked ideas; `resolved_from` / `resolved_to` window; other user's ideas not returned
- `TestSearchIdeas`: case-insensitive substring match; ranked by `match_count`, then newest first; an idea matched by several variants appears once; default includes resolved ideas; `statuses` filter; merged idea returned with `merged_into_id`; 0 or more than 5 variants, or an empty variant, fails; other user's ideas not returned
- `TestUpdateIdea`: append adds `"\n\n" + text`; every allowed transition succeeds; setting the current status is a no-op; transitions that are not allowed fail (e.g. `someday → inbox`, `spike → inbox`); any change on a `resolved` idea fails
- `TestResolveIdea`:
  - `dropped` / `expired` set `status=resolved`, `resolution`, `resolved_at`
  - `merged` into an older idea → older idea's `surface_count` = 2 and `last_surfaced_at` = duplicate's `created_at`
  - merging B (with one duplicate) into A → A's `surface_count` = 3, the duplicate now points to A
  - `merged` into itself, into a resolved idea, into another user's idea, or without `merged_into_id` fails
  - `promoted` with the user's own point succeeds; missing point id, other user's point, or non-existent point fails
  - deleting the promoting point (`delete_progress_point`) keeps the idea `promoted` with `resolved_progress_point_id` = NULL
  - `spike` idea resolved as `blocked` succeeds; `blocked` from `inbox` or `someday` fails
  - `blocked` idea re-resolved as `promoted` / `dropped` / `expired` succeeds and updates `resolved_at`
  - re-resolving any other resolved idea fails
- `TestCreateStep_CreatedByProgressPoint` (in `tests/progress_steps_test.go`): link saved; other user's point or a point of another activity fails
- `TestIdeasWeb`: `GET /web/ideas` shows `inbox` ideas as cards with created date and the spike and search links, hides `someday` / `spike` / resolved, shows surface count; `POST /web/ideas` creates and redirects `303`; empty body re-renders with an error and creates nothing
- `TestIdeasSpikeWeb`: shows `spike` ideas as cards, newest first, with the back link; hides `inbox` / `someday` / resolved and other user's ideas
- `TestIdeasSearchWeb`: no `q` shows the 50 newest ideas of any status with status and resolution in the footer (51st oldest is hidden); `q` with two comma-separated phrases finds ideas matching either, case-insensitive, resolved included; more than 5 phrases shows an error; other user's ideas not shown

## Follow-ups after implementation

- Update `activity-rituals.md` §2.2–2.5, §3.1, §3.2 step 1–2, §3.3 step 2, §3.4 step 2 (inbox is `ideas`, no «Inbox» activity, spike is the idea's `spike` status instead of a step on «Inbox», outcome appended to the idea), and the «Inbox» rows in `activity-mechanics.md`
- Add `action/ideas` to `docs/architecture.md`, route ideas in `transport/mcp/instructions.md` (`search_ideas` before `create_idea`)
- Update `progress-spec.md`: `create_step` input `created_by_progress_point_id`
- `activity-rituals.md` §2.5: trigger 3 (benefit over the current portfolio), spike outcomes (go with displacement / resolution blocked / dropped / someday), the spike limit (1 or 2 per week, also §3.2 step 9); §3.3 step 1: review ideas resolved as `blocked` at the portfolio check
- Remove the backlog item

## Changelog

- **03-10-26** — migrated to the new spec template: removed Best Practices Applied and the sequence diagrams together with every reference to them; Configuration narrowed to constants; added Changelog (Architecture Diagrams, Configuration, Changelog)
- **29-09-26** — web UI reworked into inbox cards, a spike page and a search page (Overview, Architecture Diagrams, Go Code Structure, HTTP Handlers, E2E Tests)
- **29-09-26** — initial version: ideas inbox with MCP tools and a web capture page
