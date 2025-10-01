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

Agent should find or create a new feature document in the docs folder.
One document per action. Each action can contain different handlers.

Document name format: `{name}_action`

Write requirements, implementation plan and discuss it with the user to make it optimal. Agent MUST NOT EDIT OR WRITE code files. Agent can write or update ONLY the feature document.
Ask USER for APPROVAL or change request. Continue to next step ONLY AFTER EXPLICIT APPROVAL. NO EXCEPTIONS.

Agent should write collected data using SHORT, UNDERSTANDABLE style.
IF feature document already exists - agent MUST edit it with newly appeared requirements.

### Stage 2: E2E Tests Implementation

Write tests. Agent MUST NOT EDIT OR WRITE regular files, ONLY TESTS. Do not run tests, comment code if it does not compile.

Ask USER for APPROVAL or change request. Continue to next step ONLY AFTER EXPLICIT APPROVAL. NO EXCEPTIONS.

### Stage 3: Feature Implementation

Only after Stage 3 approval can the agent write and edit any project files to make the feature work.
