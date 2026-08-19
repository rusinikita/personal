New code MUST follow existing patterns.

## Layers

1. Workers, http handlers, bot commands handlers, MCP tool handlers
2. Domain structures in domain package
3. Repository as database client with convenient interface in repository package
4. Tests in tests package

## Actions

One Go package per subdomain under `action/{subdomain}/`, matching `docs/functions/{subdomain}-spec.md`. Do not create a new package per individual action — add a new file to the existing subdomain package.

Current packages: `action/food`, `action/workout`, `action/money`, `action/progress`, `action/auth`.

## Repository

Single structure for all database interaction. Actual repository interface in repository/interface.go.

Each repository method in separate file.
Reusable mapper functions in mapper_{entity_name}.go files.

## Tests

HTTP handlers, bot handlers and workers MUST be covered by end-to-end tests in root tests package.

HTTP handlers tests must use httptest to run test server and call function.
Bot handlers tests must use telegraf test functions to send test message.
Tests must use ready-made test suite.
Tests must use table test structure.
Tests MUST NOT CALL database using SQL directly. Only using repository. NO EXCEPTIONS.
Tests can call repository methods ONLY for actual data state assertions and start state creation.