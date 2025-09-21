# GEMINI.md

## Project Overview

This is a Go project that provides a set of personal tools, including Telegram bots and web applications. The main functionality is located in `main.go`, which sets up a `gin` HTTP server and an `mcp` tool server. The project includes a `say_hi` tool as an example.

The project is structured with the following layers:
- **Actions:** Contain the business logic for different features.
- **Common:** Contains shared code and utilities.
- **Docs:** Contains project documentation.
- **Domain:** Contains the domain models.
- **Gateways:** Contains interfaces for external services like databases.
- **Tests:** Contains integration and unit tests.

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
*   **Testing:** HTTP handlers, bot handlers and workers MUST be covered by end-to-end tests in root tests package following requirements from docs/architecture.md
*   **Commits:** Follow the [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) specification.
*   **Architecture** Follow project structure and architecture requirements written in docs/architecture.md

## AI driven development convention

In each development session AI agent MUST follow these instructions. NO EXCEPTIONS.

Terms:
- Model - go struct defining application object data structure and relations with other objects
- Action - controller handling user request via interface (it could be MCP, HTTP, bot request in any combination)
- Gateway - wrapper around external API or database

### Stage 0: Feature document

Agent should find or create new feature document in docs folder.
One document per each action. Action can contain different handlers.

Document name: {name}_action

Feature document template
```markdown
# {Feature name}

## Requirements

// TODO requirements specification

## Implementation

### Domain structure

// TODO domain structure (structs and fields) code block

### Database

// TODO repository interface methods specification code block
// TODO database tables and columns as mermaid ER diagram code block

### External API

// TODO external api if present

// TODO header and description per each interface
### handler/command/tool/worker

// TODO internal logic short description
// TODO input and output data code blocks

```

### Stage 1: Requirements gathering

Agent should ask for task details if not everything provided. Such as envolved entities, database tables, http requests and bot commands.
Agent should ask about data flow if not everything provided. Such as input data formats, database data formats, state changes and data mutations.

Ask Nikita for APPROVAL or change request. Continue to writing to document and next step ONLY AFTER APPROAL. NO EXCEPTIONS.

Agent should write collected data using SHORT, UNDERSTANDABLE style.
IF feature document already exist - agent MUST edit it with newly appeared requirements.

### Stage 2: E2E tests

Agent should provide idea for creation or modification of existed end-to-end tests in tests package for this feature.

If there is no repository method or handler which required for test - leave todo note.
Example:

```go
// Call repository to get nutrition log
```

Ask Nikita for APPROVAL or change request. Continue to writing to document and next step ONLY AFTER APPROVAL. NO EXCEPTIONS.

### Stage 3: Implementation decision

Agent should provide implementation plan.

Implementation plan consists of:
- domain structure (structs and fields)
- repository interface methods specification. NO INTERNAL LOGIC
- database tables and columns as mermaid ER diagram
- handler/worker input and output data short description
- handler/worker internal logic short description

Ask Nikita for APPROVAL or change request. Continue to writing to document and next step ONLY AFTER APPROVAL. NO EXCEPTIONS.

Agent should write collected data using SHORT, UNDERSTANDABLE style.
IF feature document already exist - agent MUST edit it with newly appeared requirements.

### Stage 4:

Implementation. Only after Stage 2 approval agent can write and edit any project files to make feature work.
