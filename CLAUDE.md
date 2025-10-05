# CLAUDE.md

## Project Overview

This is a Go project that provides a set of personal tools, including Telegram bots and web applications. The main functionality is located in `main.go`, which sets up a `gin` HTTP server and an `mcp` tool server. The project includes a `say_hi` tool as an example.

The project is structured with the following layers:
- **Actions:** Contain the business logic for different features.
- **Common:** Contains shared code and utilities.
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
*   **Commits:** Follow the [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) specification.
*   **Architecture:** Follow project structure and architecture requirements written in docs/architecture.md

## Feature Document Convention

### Feature Document Template

```markdown
# {Feature name}

## Requirements

// TODO requirements specification

## E2E Tests

// TODO list of tests

## Implementation

### Domain structure

// TODO domain structure (structs and fields) code block

### Database

// TODO repository interface methods specification code block
// TODO database tables and columns as mermaid ER diagram code block

### External API

// TODO external API if present

### Handler/Command/Tool/Worker

// TODO header and description for each interface

// TODO internal logic short description
// TODO input and output data code blocks

```

### Requirements

Agent should ask for task details if not everything is provided: involved entities, database tables, HTTP requests, and bot commands.
Agent should ask about data flow if not everything is provided: input data formats, database data formats, state changes, and data mutations.

### Tests Idea

Agent should write ideas for creating or modifying existing end-to-end tests in the tests package for this feature.

If there is no repository method or handler required for the test - leave a TODO note.
Example:

```go
// Call repository to get nutrition log
```

### Implementation Decision

Agent should provide an implementation plan.

Implementation plan consists of:
- Domain structure (structs and fields)
- Repository interface methods specification (NO INTERNAL LOGIC)
- Database tables and columns as Mermaid ER diagram
- Handler/worker input and output data short description
- Handler/worker internal logic short description

## AI-Driven Development Convention

In each development session, the AI agent MUST follow these instructions. NO EXCEPTIONS.

### Stage 1: Planning and Working on Feature Document

**What agent MUST do:**
- Find existing or create NEW feature document in `docs/` folder
- Document name format: `docs/{name}_action.md` (MUST include .md extension)
- One document per action (each action can contain different handlers: HTTP, bot, MCP tool, worker)
- Write requirements section by asking user for complete information
- Write E2E tests section with test scenarios
- Write implementation section with domain structures, database schemas, handler specifications
- Use SHORT, UNDERSTANDABLE style in all sections
- If feature document already exists - agent MUST EDIT it to add newly appeared requirements

**What agent MUST NOT do:**
- NEVER write or edit ANY .go files (including test files)
- NEVER write or edit ANY code files in ANY programming language
- NEVER create any files outside docs/ folder
- NEVER proceed to Stage 2 without explicit user approval

**Communication rules:**
- Ask user for missing information via TODO comments inside feature document
- Ask user for missing information via chat if needed
- After completing feature document, ask user in chat: "Feature document ready. Please review and provide APPROVAL or change request."
- Agent can iterate multiple times on feature document based on user feedback
- Continue to Stage 2 ONLY after user explicitly says "APPROVED" or "proceed to stage 2" or similar explicit approval

**Stage 1 deliverable:** Complete feature document with all sections filled (Requirements, E2E Tests, Implementation)

### Stage 2: E2E Tests Implementation

**What agent MUST do:**
- Write or modify ONLY test files in `tests/` package
- Test files MUST have `_test.go` suffix
- Follow E2E test scenarios from feature document
- If test requires non-existent repository method or handler, leave TODO comment like: `// TODO: implement Repository.GetNutritionLog method`
- If test code doesn't compile due to missing implementation, COMMENT OUT the test code and leave explanation comment
- NEVER fix compilation errors by creating stubs in non-test files

**What agent MUST NOT do:**
- NEVER edit or create ANY .go files outside tests/ folder
- NEVER edit or create implementation files (actions/, domain/, gateways/, common/)
- NEVER create stub implementations to make tests compile
- NEVER run the tests (compilation check is optional but not required)
- NEVER proceed to Stage 3 without explicit user approval

**Communication rules:**
- After completing test implementation, ask user in chat: "E2E tests written. Please review and provide APPROVAL or change request."
- Continue to Stage 3 ONLY after user explicitly says "APPROVED" or "proceed to stage 3" or similar explicit approval

**Stage 2 deliverable:** E2E test files in tests/ package (may have commented code or TODO comments for missing implementations)

### Stage 3: Feature Implementation

**What agent MUST do:**
- Implement feature according to feature document plan
- Edit or create ANY .go files as needed to complete the feature
- Uncomment test code from Stage 2
- Implement missing methods referenced in tests
- Run `make build-app` to check compilation (NEVER use `go build` directly)
- Run `make tests` to verify tests pass (NEVER use `go test` directly)
- Fix any build or test failures
- Follow project architecture from docs/architecture.md
- Use existing patterns from codebase

**What agent MUST NOT do:**
- NEVER run `go build` commands directly - always use `make build-app`
- NEVER run `go test` commands directly - always use `make tests`

**What agent CAN do:**
- Create new files if absolutely necessary
- Edit existing files
- Refactor code if needed for feature
- Iterate on implementation until all tests pass

**Stage 3 deliverable:** Working, tested feature implementation with all E2E tests passing
