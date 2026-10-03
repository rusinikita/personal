# CLAUDE.md

## Project Overview

This is a Go project that provides a set of personal tools and web applications. The main functionality is located in `main.go`, which sets up a `gin` HTTP server and an `mcp` tool server.

The project is structured with the following layers:
- **Actions:** Contain the business logic for different features, grouped one Go package per subdomain under `action/{subdomain}/` (see `docs/architecture.md`).
- **Util:** Contains shared code and utilities.
- **Docs:** Contains project documentation.
- **Domain:** Contains the domain models.
- **Gateways:** Contains interfaces for external services like databases.
- **Tests:** Contains integration and unit tests.

## Terms

- **Model** - Go struct defining application object data structure and relations with other objects
- **Action** - Controller handling user requests via interface (MCP, HTTP, bot request, or any combination)
- **Gateway** - Wrapper around external API or database
- **Feature document** - Markdown file documenting feature requirements and implementation

## Building and Running

To build and run the project, you can use the following commands from the `Makefile`:

```sh
# To format the code
make format

# To install the development tools
make install-tools

# To build the application for Linux
make deploy

# To run the application in a Docker container
make up

# To stop the application's Docker container
make down
```

## Development Conventions

*   **Code Style:** Follow the standard Go formatting guidelines. Use `gofmt` to format your code before committing.
*   **Testing:** HTTP handlers, bot handlers, and workers MUST be covered by end-to-end tests in root tests package following requirements from docs/architecture.md
*   **Commits:** Follow the [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) specification. NEVER add `Co-Authored-By` or any co-authorship lines to commit messages.
*   **Architecture:** Follow project structure and architecture requirements written in docs/architecture.md

## Common Design Practices

Apply to every subdomain. Feature documents do NOT repeat them — a document only states where it deviates.

**Data**
- **Multi-user:** every table has `user_id`; every read and write is scoped by it. `user_id` comes from the auth context (JWT/session), never from tool or handler input
- **Time:** all timestamps are stored in UTC; conversion to the display timezone (`Asia/Nicosia`, e.g. for day boundaries) happens in the action layer
- **Computed, not stored:** derived values (last used at, counters, records, stats) are computed at read time with a JOIN or aggregate. Store a derived value only when computing it on every read is too expensive, and say who refreshes it
- **Migrations:** `ApplyMigrations` re-runs every `gateways/db/migrations/*.sql` on each startup, so every statement must be idempotent (`IF NOT EXISTS`, guarded `ALTER`). A file that references another subdomain's table gets a `z_` prefix to run after it

**Actions**
- **Validate before writing:** every rule is checked in the tool/handler and returns a clear error message. DB `CHECK` constraints are a backstop, never the error the caller sees
- **One validation and one write path per operation:** a web form and an MCP tool doing the same thing share the same validation function and repository calls
- **Reuse before adding:** use existing repository methods, also for cross-subdomain reads (the repository is shared); extend an existing filter struct rather than add a near-duplicate method
- **Partial updates:** edit tools take optional (pointer) fields; only provided fields change, and a call with no fields is an error
- **Text search:** 1-5 query variants, case-insensitive substring match, ranked by number of variants matched, then newest first. No embeddings or full-text index
- **Configuration:** secrets and deployment settings come from env vars; a missing required one fails the app at startup

**Web pages**
- Built from `action/webui` components and the shared shell, behind `WebMiddleware`; no client-side JS and no CSS beyond what `webui` provides
- **State lives in the URL:** filters, ranges and pages are query params on GET forms (`?page=N`, 1-indexed), no session state
- **Write, then redirect:** a POST form redirects back to a GET page on success and re-renders with an inline error on failure, creating nothing

## Backlog Convention

`docs/backlog.md` holds ideas for future work that haven't started yet.

- Each item is headed by the date it was added (`DD-MM-YY`) followed by a short title, not a sequence number — this way removing a finished item never forces renumbering the rest
- Each item includes a short description, a **Why:** rationale, and, when useful, a **Use cases:** list and a **Depends on:** line referencing other items by date + title
- Per the AI-Driven Development Convention, a backlog item MUST be turned into a feature document (`docs/functions/{subdomain}-spec.md`) and go through Stage 1 approval before implementation starts
- When a backlog item's implementation is complete, remove it from `docs/backlog.md`

## Feature Document Convention

### Feature Document Template

```markdown
# {Subdomain} - Complete Specification

## Overview

// TODO short description of the system and its purpose

## Architecture Diagrams

### Entity Relation Diagram

// TODO mermaid erDiagram

### C4 Context Diagram

// TODO mermaid graph with actors and tool/handler calls

## Database Schema

### SQL DDL

// TODO CREATE TABLE statements with indexes and constraints

## Go Code Structure

### Domain Models

// TODO Go structs for domain objects and filter/search params

### Repository Interface

// TODO Go interface with method signatures (NO internal logic)

## MCP Tools / HTTP Handlers / Bot Commands

### {tool_name or route}

// TODO short description. NOT Input, Output, Logic, Errors - only short desc

## Configuration

// TODO optional: environment variables and constants (defaults, limits, timeouts, timezones). Omit section if none

## E2E Tests

// TODO per test file: one bullet per test function (`TestName`: scenarios it covers) or per group sharing a prefix (`TestName_*`: scenarios)

## Changelog

// TODO one entry per change request, newest first: `- **DD-MM-YY** — what changed (sections touched)`
```

### Overview

Agent should write a short description of the system and its purpose. Ask user for missing details: involved entities, database tables, HTTP routes, bot commands, MCP tools.

### Architecture Diagrams

Agent should produce ER diagram and C4 context diagram. NO sequence diagrams.

### Database Schema

Agent should write SQL DDL with all tables, indexes, and constraints.

### Go Code Structure

Agent should write domain structs and repository interface method signatures (NO internal logic).

### MCP Tools / HTTP Handlers / Bot Commands

Agent should document each tool/handler.

### Configuration

Optional section, omitted when the subdomain has nothing to configure. Agent should list only environment variables (name, purpose, required or not) and constants (defaults, limits, timeouts, timezones), one bullet each. NO enum value lists (they belong to SQL DDL and Domain Models), NO design decisions, NO out-of-scope notes.

### E2E Tests

Agent should name the test files in `tests/` and list one bullet per test function with the scenarios it covers, success and failure cases, in short phrases. Functions sharing a prefix can go in one bullet (`TestCreateActivity_*`: success; empty name; ...). Every HTTP handler, bot handler, MCP tool, and worker of the subdomain MUST be covered by at least one listed test. Stage 2 tests are written from this list.

### Changelog

Last section of the document. Agent should add one entry for every change request to the document, newest first, headed by the date (`DD-MM-YY`). Each entry says in one or two lines what changed and which sections were touched, so a reviewer can find the diff without rereading the whole document. A new document starts with a single "initial version" entry. Existing entries are never rewritten or removed.

## AI-Driven Development Convention

In each development session, the AI agent MUST follow these instructions. NO EXCEPTIONS.

### Stage 1: Planning and Working on Feature Document

**What agent MUST do:**
- Find existing or create NEW feature document in `docs/functions/` folder
- Document name format: `docs/functions/{subdomain}-spec.md` (MUST include .md extension)
- One document per subdomain (each subdomain document covers all related actions and handlers: HTTP, bot, MCP tool, worker)
- Write Overview by asking user for complete information
- Write Architecture Diagrams (ER, C4 context)
- Write Database Schema (SQL DDL), Go Code Structure (domain models, repository interface)
- Write handler/tool sections with only short description
- Write Configuration section (environment variables and constants) if the subdomain has any
- Write E2E Tests section (test functions and scenarios they cover)
- Use SHORT, UNDERSTANDABLE style in all sections
- If feature document already exists - agent MUST EDIT it to add newly appeared requirements
- Add a Changelog entry for every change made to the feature document

**What agent MUST NOT do:**
- NEVER write or edit ANY .go files (including test files)
- NEVER write or edit ANY code files in ANY programming language
- NEVER create any files outside docs/functions/ folder
- NEVER proceed to Stage 2 (E2E Tests and Feature Implementation) without explicit user approval

**Communication rules:**
- Ask user for missing information via TODO comments inside feature document
- Ask user for missing information via chat if needed
- After completing feature document, ask user in chat: "Feature document ready. Please review and provide APPROVAL or change request."
- Agent can iterate multiple times on feature document based on user feedback
- Continue to Stage 2 ONLY after user explicitly says "APPROVED" or "proceed to stage 2" or similar explicit approval

**Stage 1 deliverable:** Complete feature document with all sections filled (Overview, Architecture Diagrams, Database Schema, Go Code Structure, handlers/tools, Configuration if any, E2E Tests, Changelog)

### Stage 2: E2E Tests and Feature Implementation

**What agent MUST do:**
- Write or modify E2E test files (`_test.go` suffix) in `tests/` package, following scenarios from feature document
- Implement the feature according to the feature document plan: edit or create ANY .go files as needed (actions/, domain/, gateways/, common/, tests/)
- Run `make build-app` to check compilation (NEVER use `go build` directly)
- Run `make test` to verify tests pass (NEVER use `go test` directly)
- Fix any build or test failures
- Follow project architecture from docs/architecture.md
- Use existing patterns from codebase

**What agent MUST NOT do:**
- NEVER run `go build` commands directly - always use `make build-app`
- NEVER run `go test` commands directly - always use `make test`

**What agent CAN do:**
- Create new files if absolutely necessary
- Edit existing files
- Refactor code if needed for feature
- Iterate on tests and implementation together until all tests pass

**Stage 2 deliverable:** Working, tested feature implementation with all E2E tests passing
