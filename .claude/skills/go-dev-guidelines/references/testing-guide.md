# Go Testing Guide

## Test-Driven Development (TDD) Approach

Follow the Red-Green-Refactor cycle for all Go development:

1. **Red** - Write a failing test first
2. **Green** - Write minimal code to make the test pass
3. **Refactor** - Improve code while keeping tests green

### TDD Benefits

- Forces you to think about the API before implementation
- Ensures code is testable by design
- Provides immediate feedback
- Acts as documentation
- Prevents regressions

## Test File Organization

### File Naming

Test files must end with `_test.go`:

```
user_service.go        # Implementation
user_service_test.go   # Tests
```

### Package Naming

Use the same package for unit tests:

```go
// user_service.go
package service

// user_service_test.go
package service  // Same package for unit tests
```

Use `_test` suffix package for integration/black-box tests:

```go
// user_service_integration_test.go
package service_test  // Different package for integration tests

import "yourproject/internal/service"
```

## Using testify/require

**ALWAYS** use `github.com/stretchr/testify/require` for assertions. Never use the standard `testing` package assertions directly.

### Basic Assertions

```go
import (
    "testing"
    "github.com/stretchr/testify/require"
)

func TestUserService_Create(t *testing.T) {
    svc := NewUserService()
    user := User{Name: "John"}

    err := svc.Create(context.Background(), user)

    require.NoError(t, err)
    require.NotNil(t, user.ID)
    require.Equal(t, "John", user.Name)
}
```

### Common Assertions

```go
// Error assertions
require.NoError(t, err)
require.Error(t, err)
require.ErrorIs(t, err, ErrNotFound)
require.ErrorContains(t, err, "not found")

// Nil assertions
require.Nil(t, value)
require.NotNil(t, value)

// Equality assertions
require.Equal(t, expected, actual)
require.NotEqual(t, expected, actual)
require.EqualValues(t, expected, actual)  // Converts types before comparing

// Boolean assertions
require.True(t, condition)
require.False(t, condition)

// Collection assertions
require.Len(t, collection, expectedLen)
require.Empty(t, collection)
require.NotEmpty(t, collection)
require.Contains(t, slice, element)
require.ElementsMatch(t, expected, actual)  // Same elements, any order

// String assertions
require.Contains(t, actualString, substring)
require.NotContains(t, actualString, substring)
```

### Why require vs assert?

Use `require` (not `assert`). `require` stops test execution immediately on failure, while `assert` continues. This prevents cascading failures and confusing error messages.

```go
// ✅ Good - stops on failure
require.NoError(t, err)
require.NotNil(t, user)
require.Equal(t, "John", user.Name)

// ❌ Bad - continues on failure, may panic on next line
assert.NoError(t, err)
assert.NotNil(t, user)
assert.Equal(t, "John", user.Name)  // May panic if user is nil
```

## Test Structure

### Arrange-Act-Assert Pattern

Structure every test with three clear sections:

```go
func TestUserService_Create(t *testing.T) {
    // Arrange - Set up test dependencies and inputs
    mockRepo := mocks.NewUserRepository(t)
    mockRepo.On("Save", mock.Anything).Return(nil)
    svc := NewUserService(mockRepo)
    user := User{Name: "John", Email: "john@example.com"}

    // Act - Execute the code under test
    err := svc.Create(context.Background(), user)

    // Assert - Verify the results
    require.NoError(t, err)
    require.NotEmpty(t, user.ID)
    mockRepo.AssertExpectations(t)
}
```

### Test Naming Convention

Use descriptive names that explain what is being tested:

```go
// Format: Test<Type>_<Method>
func TestUserService_Create(t *testing.T) {}

// Format: Test<Type>_<Method>_<Scenario>
func TestUserService_Create_WithValidInput(t *testing.T) {}
func TestUserService_Create_WithEmptyName(t *testing.T) {}
func TestUserService_Create_WhenRepositoryFails(t *testing.T) {}

// Format: Test<Function>
func TestValidateEmail(t *testing.T) {}
func TestValidateEmail_EmptyString(t *testing.T) {}
```

## Mocking with Mockery

### Generating Mocks

Use mockery to generate mocks from interfaces:

```bash
# Generate mock for a specific interface
mockery --name=UserRepository --dir=internal/repository --output=internal/repository/mocks

# Generate all mocks using config file
mockery --config=.mockery.yaml
```

### Mockery Configuration

Create `.mockery.yaml` in project root:

```yaml
with-expecter: true
dir: "{{.InterfaceDir}}"
mockname: "{{.InterfaceName}}"
outpkg: mocks
packages:
  github.com/yourorg/yourproject/internal/repository:
    interfaces:
      UserRepository:
        config:
          dir: "internal/repository/mocks"
  github.com/yourorg/yourproject/internal/service:
    interfaces:
      UserService:
        config:
          dir: "internal/service/mocks"
```

### Using Mocks in Tests

```go
import (
    "testing"
    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/require"
    "yourproject/internal/repository/mocks"
)

func TestUserService_Create(t *testing.T) {
    // Arrange
    mockRepo := mocks.NewUserRepository(t)

    // Setup mock expectations
    mockRepo.On("Save", mock.Anything, mock.MatchedBy(func(u User) bool {
        return u.Name == "John"
    })).Return(nil)

    svc := NewUserService(mockRepo)
    user := User{Name: "John"}

    // Act
    err := svc.Create(context.Background(), user)

    // Assert
    require.NoError(t, err)
    mockRepo.AssertExpectations(t)  // Verify mock was called correctly
}
```

### Mock Argument Matchers

```go
// Any argument
mockRepo.On("Save", mock.Anything).Return(nil)

// Specific value
mockRepo.On("Save", user).Return(nil)

// Type matcher
mockRepo.On("FindByID", mock.AnythingOfType("string")).Return(&user, nil)

// Custom matcher
mockRepo.On("Save", mock.MatchedBy(func(u User) bool {
    return u.Age >= 18
})).Return(nil)
```

### Mock Return Values

```go
// Single return value
mockRepo.On("Delete", "123").Return(nil)

// Multiple return values
mockRepo.On("FindByID", "123").Return(&user, nil)

// Different returns for different calls
mockRepo.On("FindByID", "123").Return(&user, nil).Once()
mockRepo.On("FindByID", "456").Return(nil, ErrNotFound)

// Return based on input
mockRepo.On("FindByID", mock.Anything).Return(func(id string) (*User, error) {
    if id == "123" {
        return &user, nil
    }
    return nil, ErrNotFound
})
```

## Test Organization Rules

### Rule 1: Never Use Table-Driven Tests

**Avoid** table-driven tests. Write explicit test functions instead.

```go
// ❌ Bad - Table-driven test
func TestValidateEmail(t *testing.T) {
    tests := []struct {
        name    string
        email   string
        wantErr bool
    }{
        {"valid email", "test@example.com", false},
        {"invalid email", "notanemail", true},
        {"empty email", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateEmail(tt.email)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateEmail() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}

// ✅ Good - Explicit test functions
func TestValidateEmail_ValidEmail(t *testing.T) {
    err := ValidateEmail("test@example.com")
    require.NoError(t, err)
}

func TestValidateEmail_InvalidFormat(t *testing.T) {
    err := ValidateEmail("notanemail")
    require.Error(t, err)
    require.Contains(t, err.Error(), "invalid email format")
}

func TestValidateEmail_EmptyString(t *testing.T) {
    err := ValidateEmail("")
    require.Error(t, err)
    require.Contains(t, err.Error(), "email cannot be empty")
}
```

**Why?** Explicit tests are:
- Easier to read and understand
- Easier to debug when they fail
- More maintainable
- Better for TDD workflow

### Rule 2: Never Mix Positive and Negative Tests

Keep positive (success) and negative (error) cases in separate test functions.

```go
// ❌ Bad - Mixed positive and negative
func TestUserService_Create(t *testing.T) {
    // Test success case
    err := svc.Create(ctx, validUser)
    require.NoError(t, err)

    // Test error case - DON'T DO THIS IN SAME FUNCTION
    err = svc.Create(ctx, invalidUser)
    require.Error(t, err)
}

// ✅ Good - Separate functions
func TestUserService_Create(t *testing.T) {
    mockRepo := mocks.NewUserRepository(t)
    mockRepo.On("Save", mock.Anything).Return(nil)
    svc := NewUserService(mockRepo)

    err := svc.Create(context.Background(), validUser)

    require.NoError(t, err)
    mockRepo.AssertExpectations(t)
}

func TestUserService_Create_InvalidInput(t *testing.T) {
    svc := NewUserService(nil)  // No need for mock in validation test

    err := svc.Create(context.Background(), User{})

    require.Error(t, err)
    require.Contains(t, err.Error(), "name is required")
}

func TestUserService_Create_RepositoryError(t *testing.T) {
    mockRepo := mocks.NewUserRepository(t)
    mockRepo.On("Save", mock.Anything).Return(errors.New("db error"))
    svc := NewUserService(mockRepo)

    err := svc.Create(context.Background(), validUser)

    require.Error(t, err)
    require.Contains(t, err.Error(), "db error")
}
```

## Test Coverage Patterns

### Testing HTTP Handlers

```go
func TestUserHandler_Create(t *testing.T) {
    // Arrange
    mockService := mocks.NewUserService(t)
    mockService.On("Create", mock.Anything, mock.Anything).Return(nil)
    handler := NewUserHandler(mockService)

    body := `{"name":"John","email":"john@example.com"}`
    req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
    rec := httptest.NewRecorder()

    // Act
    handler.Create(rec, req)

    // Assert
    require.Equal(t, http.StatusCreated, rec.Code)
    mockService.AssertExpectations(t)
}

func TestUserHandler_Create_InvalidJSON(t *testing.T) {
    handler := NewUserHandler(nil)

    req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader("invalid"))
    rec := httptest.NewRecorder()

    handler.Create(rec, req)

    require.Equal(t, http.StatusBadRequest, rec.Code)
}
```

### Testing with Context

```go
func TestUserService_Create_ContextCanceled(t *testing.T) {
    mockRepo := mocks.NewUserRepository(t)
    svc := NewUserService(mockRepo)

    ctx, cancel := context.WithCancel(context.Background())
    cancel()  // Cancel immediately

    err := svc.Create(ctx, user)

    require.Error(t, err)
    require.ErrorIs(t, err, context.Canceled)
}
```

### Testing Concurrency

```go
func TestCache_ConcurrentAccess(t *testing.T) {
    cache := NewCache()

    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            key := fmt.Sprintf("key-%d", id)
            cache.Set(key, id)
            val, err := cache.Get(key)
            require.NoError(t, err)
            require.Equal(t, id, val)
        }(i)
    }

    wg.Wait()
}
```

## Integration Tests

### File Organization

Place integration tests in separate files or directories:

```
internal/repository/
├── user_repository.go
├── user_repository_test.go           # Unit tests
└── user_repository_integration_test.go  # Integration tests
```

### Build Tags

Use build tags to separate integration tests:

```go
//go:build integration
// +build integration

package repository_test

import "testing"

func TestUserRepository_Integration(t *testing.T) {
    // Integration test with real database
}
```

Run with: `go test -tags=integration ./...`

### Integration Test Structure

```go
func TestUserRepository_Integration(t *testing.T) {
    // Setup
    db := setupTestDatabase(t)
    defer teardownTestDatabase(t, db)

    repo := NewUserRepository(db)
    user := User{Name: "John", Email: "john@example.com"}

    // Execute
    err := repo.Save(context.Background(), user)
    require.NoError(t, err)

    // Verify
    found, err := repo.FindByEmail(context.Background(), "john@example.com")
    require.NoError(t, err)
    require.Equal(t, user.Name, found.Name)
}
```

## Test Helpers

### Setup and Teardown

```go
func setupTestServer(t *testing.T) *Server {
    t.Helper()  // Mark as helper for better error reporting

    server := NewServer()
    // Setup logic
    return server
}

func teardownTestServer(t *testing.T, server *Server) {
    t.Helper()

    if err := server.Close(); err != nil {
        t.Errorf("failed to close server: %v", err)
    }
}

// Usage
func TestServer(t *testing.T) {
    server := setupTestServer(t)
    defer teardownTestServer(t, server)

    // Test logic
}
```

### Test Data Builders

```go
func createTestUser(t *testing.T, name string) User {
    t.Helper()

    return User{
        ID:        uuid.New().String(),
        Name:      name,
        Email:     fmt.Sprintf("%s@example.com", strings.ToLower(name)),
        CreatedAt: time.Now(),
    }
}

// Usage
func TestUserService(t *testing.T) {
    user := createTestUser(t, "John")
    // Use user in test
}
```

## Running Tests

### Basic Commands

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...

# Run tests with detailed coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific test
go test -run TestUserService_Create ./internal/service

# Run tests with race detector
go test -race ./...

# Run integration tests only
go test -tags=integration ./...

# Run tests in parallel
go test -parallel 4 ./...
```

### Makefile Targets

```makefile
.PHONY: test test-unit test-integration test-coverage

test:
	go test -v -race ./...

test-unit:
	go test -v -short ./...

test-integration:
	go test -v -tags=integration ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"
```

## Testing Best Practices

### ✅ Do's

- Write tests first (TDD)
- Use testify/require for all assertions
- Generate mocks with mockery in local `mocks/` folders
- Write explicit test functions, not table-driven tests
- Separate positive and negative test cases
- Use descriptive test names
- Test one thing per test function
- Use test helpers with `t.Helper()`
- Test error cases thoroughly
- Use `t.Cleanup()` for teardown logic

### ❌ Don'ts

- Don't use table-driven tests
- Don't mix positive and negative tests
- Don't use `assert`, always use `require`
- Don't test implementation details
- Don't share state between tests
- Don't ignore test failures
- Don't skip error handling in tests
- Don't make tests depend on each other
- Don't test external services without mocks

## Test Checklist

- [ ] Tests written before implementation (TDD)
- [ ] Using testify/require for assertions
- [ ] Mocks generated with mockery in `mocks/` subfolder
- [ ] No table-driven tests
- [ ] Positive and negative tests separated
- [ ] Each test has clear Arrange-Act-Assert structure
- [ ] Descriptive test names
- [ ] All error cases covered
- [ ] Integration tests separated with build tags
- [ ] Tests run with `-race` flag
- [ ] Code coverage meets team standards
