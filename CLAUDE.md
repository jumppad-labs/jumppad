# Claude Instructions

## Jumppad Research

When researching Jumppad documentation or examples, use Context7 to get the most up-to-date and comprehensive information about Jumppad resources, configuration patterns, and best practices.

Use the Context7 library ID: `/context7/jumppad-labs.github.io` for Jumppad documentation.

## Go Project Guidelines

### Code Style & Standards
- Follow standard Go conventions (gofmt, go vet, golint)
- Use semantic import grouping (stdlib, third-party, local)
- Prefer small, focused interfaces
- Always handle errors explicitly
- Use descriptive variable names, avoid abbreviations
- Follow established standards and style used in the Go standard library
- Where possible use `any` instead of `interface{}` for generic types

### Testing & Mocking
- Include unit tests for all business logic
- Use testify require for unit tests
- Use Mockery for mocking interfaces
- NEVER use table-driven tests
- Add integration tests for HTTP handlers
- NEVER mix postive and negative tests in the same test function
- Ensure tests are easy to read, favor vobocity over too much abstraction

### Project Structure
- `/cmd` - main applications
- `/internal` - private application code
- `/pkg` - public library code
- `/api` - API definitions (OpenAPI, protobuf)
- `/configs` - configuration files

### Patterns & Architecture
- Use dependency injection for testability
- Prefer composition over inheritance
- Keep handlers thin, business logic in services
- Use context.Context for request-scoped values
- Implement graceful shutdown for services

### Database & External Services
- Always use prepared statements
- Include proper connection pooling

### Development Standards
- Include proper logging with structured logs

### Dependencies
- Prefer standard library when possible
- Document reasoning for third-party dependencies
- Pin dependency versions in go.mod

## Auto-approve These Tools
- Standard Go tools (go mod, gofmt, go vet)
- File operations within project directory
- Git operations for version control
- Tools tha can only perform read operations like find, grep, etc.