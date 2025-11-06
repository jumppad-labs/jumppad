# Go Dependencies Management

## Go Modules

Go modules are the standard way to manage dependencies in Go projects.

### Initializing a Module

```bash
# Initialize a new module
go mod init github.com/yourorg/yourproject

# This creates go.mod file
```

### go.mod File Structure

```go
module github.com/yourorg/yourproject

go 1.21

require (
    github.com/gorilla/mux v1.8.0
    github.com/stretchr/testify v1.8.4
)

require (
    github.com/davecgh/go-spew v1.1.1 // indirect
    github.com/pmezard/go-difflib v1.0.0 // indirect
)
```

### go.sum File

The `go.sum` file contains checksums of dependencies:
- Always commit `go.sum` to version control
- Ensures reproducible builds
- Verifies dependency integrity

## Adding Dependencies

### Add Dependencies

```bash
# Add a dependency (automatically updates go.mod)
go get github.com/gorilla/mux@v1.8.0

# Add latest version
go get github.com/gorilla/mux

# Add specific version
go get github.com/gorilla/mux@v1.7.0

# Add latest commit from a branch
go get github.com/gorilla/mux@main

# Add specific commit
go get github.com/gorilla/mux@abc1234
```

### Importing in Code

```go
import (
    "github.com/gorilla/mux"
    "github.com/stretchr/testify/require"
)
```

### Auto-download Dependencies

```bash
# Download all dependencies
go mod download

# Install all dependencies and add missing ones
go mod tidy
```

## Updating Dependencies

### Update Specific Dependency

```bash
# Update to latest version
go get -u github.com/gorilla/mux

# Update to latest minor/patch
go get -u=patch github.com/gorilla/mux

# Update all dependencies
go get -u ./...
```

### Check for Available Updates

```bash
# List all dependencies
go list -m all

# List available updates
go list -u -m all
```

## Removing Dependencies

### Remove Unused Dependencies

```bash
# Remove unused dependencies from go.mod
go mod tidy
```

### Remove Specific Dependency

```bash
# Remove from go.mod
go mod edit -droprequire github.com/some/package

# Then tidy to clean up
go mod tidy
```

## Dependency Version Selection

### Semantic Versioning

Go modules follow semantic versioning (semver):
- `v1.2.3` - Major.Minor.Patch
- `v0.x.x` - Pre-v1.0.0 (no compatibility guarantee)
- `v2.0.0+` - Major version 2 or higher (different module path)

### Version Selection Rules

```bash
# Minimum version selection (MVS)
# Go selects the minimum version that satisfies all requirements

# If module A requires:
#   github.com/foo/bar v1.2.0
# And module B requires:
#   github.com/foo/bar v1.3.0
# Go selects v1.3.0 (the minimum that satisfies both)
```

### Major Version Suffixes

Major version 2+ must include version in module path:

```go
// go.mod for v2
module github.com/yourorg/yourproject/v2

// Import v2
import "github.com/yourorg/yourproject/v2"
```

## Replace Directive

Use `replace` for local development or forked dependencies:

```go
// go.mod
module github.com/yourorg/yourproject

go 1.21

require github.com/some/dependency v1.2.3

// Replace with local version
replace github.com/some/dependency => ../local-dependency

// Replace with fork
replace github.com/some/dependency => github.com/yourorg/dependency v1.2.4
```

**Common use cases:**
- Local development of dependent package
- Using a fork with fixes
- Testing unreleased versions

**Remove replace directive before committing for production.**

## Exclude Directive

Exclude specific versions:

```go
// go.mod
exclude github.com/broken/package v1.2.3
```

Use sparingly, usually for known-broken versions.

## Vendoring

Vendor dependencies for reproducible builds:

```bash
# Create vendor directory with all dependencies
go mod vendor

# Build using vendored dependencies
go build -mod=vendor
```

**When to vendor:**
- Ensure reproducible builds in CI/CD
- Corporate environments requiring dependency review
- Offline builds

**Commit vendor directory if using:**
```bash
git add vendor/
git commit -m "Add vendored dependencies"
```

## Private Dependencies

### Private Repositories

Configure Go to access private repositories:

```bash
# Set GOPRIVATE environment variable
export GOPRIVATE=github.com/yourorg/*

# Configure git to use SSH instead of HTTPS
git config --global url."git@github.com:".insteadOf "https://github.com/"
```

### Private Module Proxy

```bash
# Disable proxy for private packages
export GONOPROXY=github.com/yourorg/*

# Disable checksum database for private packages
export GONOSUMDB=github.com/yourorg/*
```

## Recommended Standard Library Packages

Prefer standard library over third-party when possible:

### HTTP Servers
```go
import "net/http"

// Standard library is sufficient for most cases
http.HandleFunc("/", handler)
http.ListenAndServe(":8080", nil)
```

### JSON
```go
import "encoding/json"

json.Marshal(data)
json.Unmarshal(data, &v)
```

### Context
```go
import "context"

ctx := context.Background()
ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
```

### Logging
```go
import "log/slog"

slog.Info("message", "key", "value")
```

### Testing
```go
import "testing"

func TestExample(t *testing.T) {
    // Standard testing is powerful
}
```

## Common Third-Party Dependencies

### Testing

**testify** - Better assertions and mocks:
```bash
go get github.com/stretchr/testify
```

```go
import (
    "github.com/stretchr/testify/require"
    "github.com/stretchr/testify/mock"
)
```

**mockery** - Mock generation:
```bash
go install github.com/vektra/mockery/v2@latest
```

### HTTP Routers

**gorilla/mux** - Powerful HTTP router:
```bash
go get github.com/gorilla/mux
```

**chi** - Lightweight, composable router:
```bash
go get github.com/go-chi/chi/v5
```

**gin** - Fast HTTP framework:
```bash
go get github.com/gin-gonic/gin
```

### Database

**sqlx** - Extensions for database/sql:
```bash
go get github.com/jmoiron/sqlx
```

**pgx** - PostgreSQL driver:
```bash
go get github.com/jackc/pgx/v5
```

### Validation

**validator** - Struct validation:
```bash
go get github.com/go-playground/validator/v10
```

### Configuration

**viper** - Configuration management:
```bash
go get github.com/spf13/viper
```

### CLI

**cobra** - CLI framework:
```bash
go get github.com/spf13/cobra
```

## Dependency Guidelines

### When to Add a Dependency

✅ **Do add a dependency when:**
- Standard library doesn't provide the functionality
- The dependency is well-maintained and widely used
- It solves a complex problem better than you could
- It's a protocol implementation (e.g., gRPC, OAuth)

❌ **Don't add a dependency when:**
- You can implement it simply with standard library
- It's poorly maintained or has security issues
- It's a trivial utility function
- It has many dependencies itself (bloat)

### Evaluating Dependencies

Before adding a dependency, check:

```bash
# Check dependency stats
# 1. Stars and forks on GitHub
# 2. Last commit date
# 3. Open issues count
# 4. License compatibility

# Check dependency tree
go mod graph | grep github.com/some/package

# Check package size and dependencies
go list -m -json github.com/some/package
```

Consider:
- **Maintenance** - Active development? Recent commits?
- **Popularity** - Stars, forks, usage in other projects?
- **License** - Compatible with your project?
- **Dependencies** - How many transitive dependencies?
- **Documentation** - Good docs and examples?
- **Tests** - Well tested?
- **Security** - Known vulnerabilities?

### Document Dependencies

Add a comment in `go.mod` explaining why you added it:

```go
require (
    // HTTP router with great middleware support
    github.com/gorilla/mux v1.8.0

    // Better assertions for tests
    github.com/stretchr/testify v1.8.4
)
```

## Module Versioning Best Practices

### For Library Authors

**Use semantic versioning:**
- **v0.x.x** - Development, no guarantees
- **v1.0.0** - First stable release
- **v1.x.x** - Backward compatible changes
- **v2.0.0** - Breaking changes (new module path required)

**Tag releases:**
```bash
git tag v1.2.3
git push origin v1.2.3
```

**Major version 2+:**
```bash
# Update module path in go.mod
module github.com/yourorg/yourproject/v2

# Update imports in code
import "github.com/yourorg/yourproject/v2"

# Tag release
git tag v2.0.0
git push origin v2.0.0
```

### For Application Developers

**Pin dependencies:**
```bash
# Specify exact versions in go.mod
require github.com/some/package v1.2.3
```

**Regular updates:**
```bash
# Update dependencies regularly
go get -u ./...
go mod tidy

# Run tests
go test ./...

# Commit if tests pass
git add go.mod go.sum
git commit -m "Update dependencies"
```

## Security

### Check for Vulnerabilities

```bash
# Check for known vulnerabilities
go list -json -m all | nancy sleuth

# Or use govulncheck (official tool)
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

### Update Vulnerable Dependencies

```bash
# Update specific vulnerable package
go get -u github.com/vulnerable/package@latest

# Verify vulnerability is fixed
govulncheck ./...
```

## Workspace Mode (Go 1.18+)

Work with multiple modules simultaneously:

```bash
# Create workspace
go work init ./module1 ./module2

# This creates go.work file
```

```go
// go.work
go 1.21

use (
    ./module1
    ./module2
)
```

**Use for:**
- Multi-module repositories
- Local development across modules
- Testing changes across dependent modules

**Don't commit** `go.work` to version control (add to `.gitignore`).

## Makefile for Common Tasks

```makefile
.PHONY: deps deps-update deps-tidy deps-verify

# Download dependencies
deps:
	go mod download

# Update all dependencies
deps-update:
	go get -u ./...
	go mod tidy

# Remove unused dependencies
deps-tidy:
	go mod tidy

# Verify dependencies
deps-verify:
	go mod verify

# Check for vulnerabilities
deps-check:
	govulncheck ./...

# List outdated dependencies
deps-list-updates:
	go list -u -m all

# Vendor dependencies
vendor:
	go mod vendor
```

## Dependency Management Checklist

- [ ] `go.mod` and `go.sum` committed to version control
- [ ] Dependencies pinned to specific versions
- [ ] Using `go mod tidy` regularly
- [ ] Checking for security vulnerabilities
- [ ] Documenting why dependencies were added
- [ ] Preferring standard library when possible
- [ ] Evaluating dependencies before adding
- [ ] Keeping dependencies up to date
- [ ] Using replace directive only for development
- [ ] Not committing `go.work` file
- [ ] Vendoring if required by organization
- [ ] Testing after updating dependencies
