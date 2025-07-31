# ServiceNow Toolkit Development Guide

## Build/Test/Lint Commands
- **Build**: `make build` or `go build -o bin/servicenowtoolkit ./cmd/servicenowtoolkit`
- **Test all**: `make test` or `go test ./... -v`
- **Test single**: `go test ./tests/unit/client_test.go -v` or `go test ./pkg/servicenow/core -v`
- **Lint**: `make lint` or `golangci-lint run`
- **Clean**: `make clean`

## Code Style Guidelines
- **Imports**: Group stdlib, external, internal packages with blank lines between groups
- **Naming**: Use camelCase for unexported, PascalCase for exported. Interfaces end with -er when possible
- **Error handling**: Always wrap errors with context using `fmt.Errorf("operation failed: %w", err)`
- **Types**: Use struct embedding for composition, prefer interfaces for dependencies
- **Comments**: Use godoc format. No inline comments unless explaining complex logic
- **Testing**: Unit tests in `tests/unit/`, integration in `tests/integration/`. Use table-driven tests
- **Structs**: Group related fields, put exported fields first
- **Constants**: Use typed constants with descriptive names, group in const blocks
- **Functions**: Keep functions small, single responsibility. Return errors as last parameter
- **Packages**: Keep packages focused, avoid circular dependencies. Use internal/ for private code

## Project Structure
- `pkg/servicenow/`: Main SDK packages (core, table, catalog, etc.)
- `cmd/servicenowtoolkit/`: CLI application
- `internal/`: Private utilities (ratelimit, retry, tui, config)
- `tests/`: Separate unit and integration test directories
- `examples/`: Usage examples for different auth methods and operations