# Docs - Complete Specification

## Overview

New subdomain (`action/docs`) that serves a small set of hand-written, human-owned convention documents — not user data, not CRUD content — to two audiences: a read-only web page (for the user) and an MCP tool (for the agent, to pull in on demand, mid-session, when it needs more than the always-loaded `instructions` const in `transport/mcp/mcp.go` gives it). Docs are plain files, not database rows.

This spec is the mechanism only. Content lives as files under `action/docs/content/` (see that directory), written directly in their final location rather than inlined here, so Stage 2 has nothing left to do but wire them up once approved. Scope for this round is the activities subject only, split into two documents by kind rather than one document per subdomain:

- `action/docs/content/activity-mechanics.md` — what exists and how it's built: `action/progress` (activity, progress point, step, life part) and `action/achievements` (grouped here rather than in its own doc since an achievement is a target layered on top of an activity/exercise/transaction, not a fourth data model of its own). Normative against the code — fact-checked field-by-field and tool-by-tool against `action/progress`/`action/achievements` on 23-09-26, no discrepancies found. Source of truth is always the code; this doc is updated when they drift.
- `action/docs/content/activity-rituals.md` — the operating manual: when, why, and what a ritual (day/week/month/quarter) actually does, object lifecycles (inbox → someday → spike → activity), WIP limits, degradation rules, and the agent's contract (what it does unprompted, what it proposes, what it never does). This is the part a mechanical description of the schema can't give — process and judgment calls, not data shape.

`finance-workflow.md`, `workout-workflow.md`, and `nutrition-workflow.md` were drafted in an earlier pass of this spec and then deleted — mechanically listing tool/field mechanics per subdomain wasn't the actual ask; the activities rewrite above (mechanics + rituals, researched separately) is. Extending this same two-document pattern to other subjects (finance, workout, nutrition) is future scope, not part of this round — when/if that happens, it's just more files under `action/docs/content/`, discovered automatically (see Best Practices below), not a new mechanism.

`CLAUDE_LIFECOACH_INSTRUCTIONS.md` (repo root) is the unreferenced, stale precursor to the activities content — to be deleted once this spec + content are approved, not before. `CLAUDE_WORKOUT_INSTRUCTIONS.md` and `CLAUDE_NUTRITION_INSTRUCTIONS.md` have no replacement yet (out of scope this round) and stay untouched.

**`instructions` moves to an embedded file, and stays short**: `transport/mcp/mcp.go`'s `instructions` const becomes `//go:embed instructions.md` + `var instructions string` (`transport/mcp/instructions.md`) instead of a Go string literal — same reasoning as embedding the docs content itself: markdown as a file diffs and edits cleanly, a Go string literal doesn't. Content is rewritten as a dispatcher, not patched (see the dedicated section near the end of this doc) — it must not grow to carry per-domain detail; that detail lives in `action/docs/content/` instead, pulled in only when needed.

## Best Practices Applied

- **One canonical source per document, no DB table**: each doc is a plain markdown file committed under `action/docs/content/` and loaded via `go:embed`. It's edited like code (PR review, git history), not through any CRUD tool — matches how `life_parts` rows are "seeded by hand, no write tool" (see `progress-spec.md`) for content that changes rarely and deliberately.
- **Same source, two renderers, real markdown this time**: the MCP tool returns the embedded text verbatim (an LLM reads markdown fine as-is, no HTML needed). The web handler renders it to HTML with `github.com/russross/blackfriday/v2` (`blackfriday.Run(content, blackfriday.WithExtensions(blackfriday.CommonExtensions))`) before dropping it into the existing `webui` page shell — matches the precedent already set in the sibling `my_saas` project (`cmd/static_gen/main.go`), same library, same `CommonExtensions` set. A hand-rolled subset renderer (headings/paragraphs/lists only) was the original plan but doesn't hold once content uses tables, checkboxes, fenced code blocks (ASCII state diagrams), and blockquotes — all of which `activity-mechanics.md`/`activity-rituals.md` actually use. Pico CSS (classless, see `webui-spec.md`) styles blackfriday's plain `<table>`/`<blockquote>`/`<pre>`/`<code>` output correctly with no custom CSS.
- **New subdomain, not folded into progress or money**: the content spans multiple existing subdomains and belongs to none of them — forcing it into one would make that package own a concern (serving static docs) unrelated to its actual data model. `action/docs` is a single small package (one content directory, one web handler, two MCP tools) rather than a new pattern; nothing here is a repository/database concern, so there's no repository interface addition.
- **One route, a `topic` selector — not one route per doc**: two documents today, more later (other subjects, or a third kind per subject), but each is the same shape (a slug → embedded text). `GET /web/docs/{topic}` and `get_doc(topic)` both key off the same slug — the slug is exactly the content file's basename minus `.md` (`activity-mechanics`, `activity-rituals`). Adding a doc later is one more embedded file, not a new mechanism and not a code change.
- **Topic list is discovered, not hardcoded — list then load, two tools**: no `DocTopic` enum of fixed constants. `action/docs` reads its own embedded directory (`fs.ReadDir` over the `go:embed content/*.md` filesystem) to get the live topic list, title, and one-line description per doc — title is each file's first `# ` heading, description is the first blockquote line that follows it (both current docs already open with exactly that shape: `# Heading` then a `> ` intro line). `list_docs` exposes that discovery as its own tool (slug + title + description per doc); `get_doc(topic)` then loads one by slug — the same two-step shape as loading a skill/knowledge entry (list what's available, then pull the one you need), not one tool with a dynamically-built description. Both `GET /web/docs` (index) and `list_docs` call the same `docs.Topics()` helper, so both tools stay ordinary package-level `var MCPDefinition = mcp.Tool{...}` values, same as every other tool in this codebase — no dynamic schema/description construction needed. A `topic` unknown to `get_doc` is a runtime error (matches this codebase's existing convention of validating pseudo-enum strings in the handler, e.g. `progress_type`/`status` in `action/progress` — no native JSON-schema `enum` is used anywhere here), not a compile-time-checked type.
- **Achievements sits inside the mechanics/rituals docs, not its own doc**: an achievement (`action/achievements`) is a target layered on top of data another subdomain already owns (an activity, an exercise, a transaction) — it's cross-domain by nature, same as the doc content itself. Splitting it into its own topic would fragment one coherent "how do I track and aim for things" narrative across docs a reader has to cross-reference.
- **Doc pages get a sidebar from the same `docs.Topics()` list**: `GET /web/docs/{topic}` renders an `<aside class="docs-sidebar"><nav>` next to the doc (Pico stacks `aside nav` links vertically), listing every topic's title as a link, the current one marked `aria-current="page"` (reuses the shell's existing `nav a[aria-current="page"]` highlight). Lets the user jump between docs that cross-reference each other without going back to the index. Built with a small package-local `html/template` in `docs_web.go` (same as the existing index template), not a new `webui` component — only docs pages need it. The only style addition is one two-column grid rule (`.docs-layout`: fixed-width sidebar + doc, stacking to one column on narrow screens) in the `webui` custom CSS layer (`layout.html`), the one place custom CSS lives
- **No write path, no versioning UI**: there's no `create_doc`/`edit_doc` MCP tool. Updating content is a normal code change (edit the embedded file, commit, deploy) — same trust model as `life_parts` seeding.

## Architecture Diagrams

### Entity Relation Diagram

No database entities — this subdomain has no table. Content lives entirely in `go:embed`-ed files, versioned in git.

### C4 Context Diagram

```mermaid
graph TB
    User[User - Browser]
    Agent[Claude / MCP Client]

    subgraph "Docs System"
        Web[GET /web/docs/:topic]
        List[list_docs MCP tool]
        Get[get_doc MCP tool]
        FS[Embedded content files<br/>action/docs/content/*.md]

        Web -->|Topics: slugs+titles+descriptions for index + doc-page sidebar, renders content via blackfriday| FS
        List -->|Topics: slugs+titles+descriptions| FS
        Get -->|reads one doc's raw markdown| FS
    end

    User -->|views| Web
    Agent -->|discovers topics| List
    Agent -->|loads one doc by slug| Get

    style User fill:#e1f5ff
    style Agent fill:#e1f5ff
    style List fill:#ffe1e1
    style Get fill:#ffe1e1
    style Web fill:#ffe1e1
    style FS fill:#e1ffe1
```

### Sequence Diagram: Agent Discovers and Loads a Doc Mid-session

```mermaid
sequenceDiagram
    participant Agent as Claude
    participant MCP as MCP Server
    participant FS as Embedded content

    Agent->>MCP: list_docs()
    MCP->>FS: docs.Topics()
    FS-->>MCP: slug + title + description per doc
    MCP-->>Agent: topic list
    Note over Agent: Picks the topic matching the unclear convention
    Agent->>MCP: get_doc(topic="activity-rituals")
    MCP->>FS: read embedded bytes for topic
    FS-->>MCP: raw markdown content
    MCP-->>Agent: doc text
    Note over Agent: Used as context for the weekly-review<br/>flow that prompted the lookup
```

### Sequence Diagram: Web View

```mermaid
sequenceDiagram
    participant Browser
    participant Handler as GET /web/docs/:topic
    participant FS as Embedded content
    participant BF as blackfriday
    participant Webui as action/webui

    Browser->>Handler: GET /web/docs/activity-mechanics
    Handler->>FS: read embedded bytes for topic
    FS-->>Handler: raw markdown content
    Handler->>BF: blackfriday.Run(content, CommonExtensions)
    BF-->>Handler: HTML fragment
    Handler->>FS: docs.Topics()
    FS-->>Handler: slug + title per doc
    Handler->>Handler: wrap in .docs-layout: sidebar (all topics, current aria-current="page") + doc fragment
    Handler->>Webui: webui.RenderPage(w, PageData{Title, Nav, Content: layout})
    Webui-->>Browser: full HTML page (shared layout/nav)
```

## Database Schema

### SQL DDL

None — no table for this subdomain.

## Go Code Structure

### Domain Models

```go
// Topic describes one discovered doc: its slug (content filename minus
// ".md"), title (the file's first "# " heading), and one-line description
// (the first blockquote line following the heading).
type Topic struct {
    Slug        string
    Title       string
    Description string
}

// Topics returns the current topic list, derived from the embedded content
// directory — not a fixed set. Called by list_docs and by the
// GET /web/docs index handler; both read the same list, never a hardcoded one.
func Topics() ([]Topic, error) { ... }

// ListDocsInput takes nothing — the tool always returns the full
// current topic list.
type ListDocsInput struct{}

type ListDocsOutput struct {
    Topics []Topic `json:"topics" jsonschema:"Available docs: slug, title, and one-line description each"`
}

// GetDocInput selects which document to return. Topic is a plain
// string (not a fixed enum type) since the valid set is discovered, not
// known at compile time — validated against the live Topics() list inside
// the handler, same pattern this codebase already uses for pseudo-enum
// string fields like progress_type/status. The agent is expected to have
// called list_docs first to get a valid slug.
type GetDocInput struct {
    Topic string `json:"topic" jsonschema:"Doc slug to load, from list_docs"`
}

type GetDocOutput struct {
    Content string `json:"content" jsonschema:"Full markdown text of the requested doc"`
}
```

### Repository Interface

None — no database access from this subdomain.

## MCP Tools

### list_docs

No input. Returns the current list of available docs (slug, title, one-line description), read live from `action/docs/content/`. The discovery half of the list-then-load pair — call this first to find the right slug.

### get_doc

Takes a `topic` slug (from `list_docs`). Returns the full markdown text of the matching document verbatim, for the agent to load on demand — not automatically, only when the current task's conventions are genuinely unclear from the always-loaded `instructions` const alone. Unknown `topic` is a tool error.

## HTTP Handlers

### GET /web/docs

Index page listing the available docs (title + one-line description each, from `docs.Topics()`), linking to each doc's page.

### GET /web/docs/{topic}

Renders one document as an HTML page under the shared `webui` layout/nav (new "Docs" nav entry), markdown rendered to HTML via `blackfriday` — read-only, no forms, no query params. A sidebar next to the doc lists every doc from `docs.Topics()` (title, linking to `/web/docs/{slug}`), current one highlighted via `aria-current="page"` (see Best Practices). 404 for an unknown topic.

## E2E Tests

Changes in `tests/docs_test.go` (`TestDocsWeb` table):

- "doc page renders markdown to HTML": also expects `docs-sidebar`, a link to the other doc (`href="/web/docs/activity-rituals"`), and the current doc's link marked active (`href="/web/docs/activity-mechanics" aria-current="page"`)
- "index lists docs with links", "unknown topic is 404": unchanged (index page gets no sidebar)

## `transport/mcp` Instructions Update

Not part of `action/docs` itself, but bundled into this round since it's the file that makes the two new tools discoverable — and since it turned out to need more than a correctness pass. Two changes to `transport/mcp/`:

1. **`mcp.go`**: replace `const instructions = \`...\`` with:
   ```go
   //go:embed instructions.md
   var instructions string
   ```
   (needs `"embed"` added to imports). Everything else in `Server(...)` (the `mcp.ServerOptions{..., Instructions: instructions}` wiring) stays as-is.
2. **`transport/mcp/instructions.md`** — rewritten from scratch (not patched) at its final location, same as the `action/docs/content/` files. The old const was a per-subdomain procedure manual (food/workout step-by-step scripts, exact reply wording, metaphor tables) mixed with claims that no longer matched the code (`finish_activity`, wrong check-in ordering, duplicate section numbering) — that shape doesn't fit a file loaded into every session. Rewritten as a dispatcher: one short section per subdomain (food, workout, progress/activities/achievements, finance, telegram) naming its tools and the first call for the common case, plus a short cross-cutting-rules section (never state a fact without having called the tool this session; delete tools are hard/irreversible, confirm first; free-text fields carry the user's own words) that used to be scattered/implicit or activities-only. The progress/activities/achievements section is now intentionally thin — it points to `list_docs`/`get_doc` instead of repeating `activity-mechanics.md`/`activity-rituals.md`. Food, workout, and finance don't have that fallback yet, so trimming their sections to dispatcher-level is a real, accepted loss of detail until they get their own docs — tracked as `docs/backlog.md`, "25-09-26 — Mechanics/rituals docs for food, workout, finance". See the file directly for exact content.
