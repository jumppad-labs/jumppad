# Go Error Handling

## Error Handling Principles

1. **Always handle errors explicitly** - Never ignore errors
2. **Wrap errors with context** - Add information as errors propagate
3. **Use sentinel errors** for expected errors
4. **Use custom error types** for complex error information
5. **Check error types** with `errors.Is()` and `errors.As()`

## Basic Error Handling

### Never Ignore Errors

```go
// ❌ Bad - Ignoring error
data, _ := os.ReadFile("config.json")

// ✅ Good - Handle error explicitly
data, err := os.ReadFile("config.json")
if err != nil {
    return fmt.Errorf("failed to read config: %w", err)
}
```

### Return Errors Immediately

```go
// ✅ Good - Return early
func ProcessUser(id string) (*User, error) {
    user, err := findUser(id)
    if err != nil {
        return nil, fmt.Errorf("failed to find user: %w", err)
    }

    if err := user.Validate(); err != nil {
        return nil, fmt.Errorf("invalid user: %w", err)
    }

    return user, nil
}

// ❌ Bad - Nested error handling
func ProcessUser(id string) (*User, error) {
    user, err := findUser(id)
    if err == nil {
        if err := user.Validate(); err == nil {
            return user, nil
        } else {
            return nil, err
        }
    }
    return nil, err
}
```

## Error Wrapping

### Use %w to Wrap Errors

Use `%w` verb to wrap errors, preserving the error chain:

```go
// ✅ Good - Wrap errors with %w
func (s *UserService) CreateUser(ctx context.Context, user User) error {
    if err := s.validateUser(user); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }

    if err := s.repo.Create(ctx, user); err != nil {
        return fmt.Errorf("failed to create user in repository: %w", err)
    }

    return nil
}

// Consumer can unwrap and check root cause
err := service.CreateUser(ctx, user)
if errors.Is(err, repository.ErrDuplicate) {
    // Handle duplicate user
}
```

### Add Context at Each Layer

Each layer should add relevant context:

```go
// Repository layer
func (r *userRepository) Create(ctx context.Context, user User) error {
    _, err := r.db.ExecContext(ctx, query, user.ID, user.Name)
    if err != nil {
        return fmt.Errorf("database insert failed: %w", err)
    }
    return nil
}

// Service layer
func (s *UserService) CreateUser(ctx context.Context, user User) error {
    if err := s.repo.Create(ctx, user); err != nil {
        return fmt.Errorf("failed to create user %q: %w", user.Name, err)
    }
    return nil
}

// Handler layer
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
    if err := h.service.CreateUser(r.Context(), user); err != nil {
        log.Printf("create user request failed: %v", err)
        // Error chain preserved for logging
        http.Error(w, "failed to create user", http.StatusInternalServerError)
        return
    }
}
```

## Sentinel Errors

### Define Package-Level Sentinel Errors

Use `var` to define sentinel errors for expected error conditions:

```go
// internal/repository/errors.go
package repository

import "errors"

var (
    ErrNotFound      = errors.New("record not found")
    ErrDuplicate     = errors.New("duplicate record")
    ErrInvalidID     = errors.New("invalid ID format")
)

// internal/service/errors.go
package service

import "errors"

var (
    ErrUnauthorized  = errors.New("unauthorized")
    ErrInvalidInput  = errors.New("invalid input")
    ErrQuotaExceeded = errors.New("quota exceeded")
)
```

### Check Sentinel Errors with errors.Is()

```go
user, err := repo.FindByID(ctx, id)
if err != nil {
    if errors.Is(err, repository.ErrNotFound) {
        return nil, fmt.Errorf("user not found: %w", err)
    }
    return nil, fmt.Errorf("failed to find user: %w", err)
}
```

### Sentinel Errors Are Comparable

```go
// ✅ Works - errors.Is handles wrapped errors
if errors.Is(err, repository.ErrNotFound) {
    // Handle not found
}

// ❌ Don't use == for wrapped errors
if err == repository.ErrNotFound {  // Won't work if error is wrapped
    // This only works for unwrapped errors
}
```

## Custom Error Types

### Create Custom Error Types for Rich Error Information

```go
// ValidationError contains details about validation failures
type ValidationError struct {
    Field   string
    Value   any
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation error: %s %s (value: %v)", e.Field, e.Message, e.Value)
}

// Usage
func ValidateAge(age int) error {
    if age < 0 {
        return &ValidationError{
            Field:   "age",
            Value:   age,
            Message: "must be non-negative",
        }
    }
    return nil
}
```

### Check Custom Error Types with errors.As()

```go
err := service.CreateUser(ctx, user)
if err != nil {
    var validationErr *ValidationError
    if errors.As(err, &validationErr) {
        log.Printf("validation failed for field %s: %s", validationErr.Field, validationErr.Message)
        return http.StatusBadRequest
    }
    return http.StatusInternalServerError
}
```

### Multi-Field Validation Errors

```go
type ValidationErrors struct {
    Errors []ValidationError
}

func (e *ValidationErrors) Error() string {
    var messages []string
    for _, err := range e.Errors {
        messages = append(messages, err.Error())
    }
    return strings.Join(messages, "; ")
}

func (e *ValidationErrors) Add(field, message string, value any) {
    e.Errors = append(e.Errors, ValidationError{
        Field:   field,
        Message: message,
        Value:   value,
    })
}

// Usage
func ValidateUser(user User) error {
    errs := &ValidationErrors{}

    if user.Name == "" {
        errs.Add("name", "is required", user.Name)
    }
    if user.Age < 0 {
        errs.Add("age", "must be non-negative", user.Age)
    }
    if !isValidEmail(user.Email) {
        errs.Add("email", "invalid format", user.Email)
    }

    if len(errs.Errors) > 0 {
        return errs
    }
    return nil
}
```

## Error Handling Patterns

### Defer for Cleanup

Use defer to ensure cleanup happens even when errors occur:

```go
func ProcessFile(filename string) error {
    f, err := os.Open(filename)
    if err != nil {
        return fmt.Errorf("failed to open file: %w", err)
    }
    defer f.Close()  // Always closes, even if error occurs

    data, err := io.ReadAll(f)
    if err != nil {
        return fmt.Errorf("failed to read file: %w", err)
    }

    return process(data)
}
```

### Named Return Values for Defer Error Handling

```go
func CreateResource(name string) (resource *Resource, err error) {
    resource = &Resource{Name: name}

    if err = resource.Initialize(); err != nil {
        return nil, fmt.Errorf("failed to initialize: %w", err)
    }

    defer func() {
        if err != nil {
            // Cleanup on error
            resource.Cleanup()
        }
    }()

    if err = resource.Allocate(); err != nil {
        return nil, fmt.Errorf("failed to allocate: %w", err)
    }

    return resource, nil
}
```

### Panic and Recover

**Only use panic for truly exceptional situations.** Prefer returning errors.

```go
// ✅ Acceptable - Programming errors
func MustCompileRegex(pattern string) *regexp.Regexp {
    re, err := regexp.Compile(pattern)
    if err != nil {
        panic(fmt.Sprintf("invalid regex pattern %q: %v", pattern, err))
    }
    return re
}

// Use recover to handle panics at boundaries
func SafeHandler(w http.ResponseWriter, r *http.Request) {
    defer func() {
        if err := recover(); err != nil {
            log.Printf("panic in handler: %v", err)
            http.Error(w, "internal server error", http.StatusInternalServerError)
        }
    }()

    // Handler code that might panic
    handler(w, r)
}

// ❌ Bad - Using panic for control flow
func ValidateAge(age int) {
    if age < 0 {
        panic("age cannot be negative")  // DON'T DO THIS
    }
}
```

## Error Handling in Different Layers

### Handler Layer - HTTP Status Codes

Convert errors to appropriate HTTP status codes:

```go
func (h *UserHandler) GetByID(w http.ResponseWriter, r *http.Request) {
    id := r.URL.Query().Get("id")

    user, err := h.service.FindByID(r.Context(), id)
    if err != nil {
        h.handleError(w, err)
        return
    }

    json.NewEncoder(w).Encode(user)
}

func (h *UserHandler) handleError(w http.ResponseWriter, err error) {
    var validationErr *ValidationError
    if errors.As(err, &validationErr) {
        http.Error(w, validationErr.Error(), http.StatusBadRequest)
        return
    }

    if errors.Is(err, service.ErrNotFound) {
        http.Error(w, "user not found", http.StatusNotFound)
        return
    }

    if errors.Is(err, service.ErrUnauthorized) {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }

    // Log unexpected errors
    log.Printf("internal error: %v", err)
    http.Error(w, "internal server error", http.StatusInternalServerError)
}
```

### Service Layer - Business Logic Errors

```go
func (s *UserService) UpdatePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
    user, err := s.repo.FindByID(ctx, userID)
    if err != nil {
        if errors.Is(err, repository.ErrNotFound) {
            return ErrNotFound
        }
        return fmt.Errorf("failed to find user: %w", err)
    }

    if !user.PasswordMatches(oldPassword) {
        return ErrUnauthorized
    }

    if len(newPassword) < 8 {
        return &ValidationError{
            Field:   "password",
            Message: "must be at least 8 characters",
            Value:   len(newPassword),
        }
    }

    user.SetPassword(newPassword)

    if err := s.repo.Update(ctx, user); err != nil {
        return fmt.Errorf("failed to update password: %w", err)
    }

    return nil
}
```

### Repository Layer - Data Access Errors

```go
func (r *userRepository) FindByID(ctx context.Context, id string) (*User, error) {
    var user User
    err := r.db.QueryRowContext(ctx, "SELECT * FROM users WHERE id = $1", id).
        Scan(&user.ID, &user.Name, &user.Email)

    if err == sql.ErrNoRows {
        return nil, ErrNotFound
    }
    if err != nil {
        return nil, fmt.Errorf("query failed: %w", err)
    }

    return &user, nil
}
```

## Error Logging

### Structured Logging

Use structured logging for better observability:

```go
import "log/slog"

func (s *UserService) CreateUser(ctx context.Context, user User) error {
    if err := s.repo.Create(ctx, user); err != nil {
        slog.Error("failed to create user",
            "error", err,
            "user_id", user.ID,
            "user_name", user.Name,
        )
        return fmt.Errorf("failed to create user: %w", err)
    }

    slog.Info("user created",
        "user_id", user.ID,
        "user_name", user.Name,
    )

    return nil
}
```

### Log at the Right Level

```go
// Handler - Log with request context
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
    if err := h.service.CreateUser(r.Context(), user); err != nil {
        slog.Error("create user request failed",
            "error", err,
            "method", r.Method,
            "path", r.URL.Path,
            "remote_addr", r.RemoteAddr,
        )
        http.Error(w, "failed to create user", http.StatusInternalServerError)
        return
    }
}

// Service - Log business logic errors
func (s *UserService) CreateUser(ctx context.Context, user User) error {
    if err := s.repo.Create(ctx, user); err != nil {
        slog.Error("repository operation failed",
            "error", err,
            "operation", "create",
            "user_id", user.ID,
        )
        return fmt.Errorf("failed to create user: %w", err)
    }
    return nil
}

// Repository - Usually don't log, just return errors
func (r *userRepository) Create(ctx context.Context, user User) error {
    _, err := r.db.ExecContext(ctx, query, user.ID, user.Name)
    if err != nil {
        // Return error, let higher layers decide whether to log
        return fmt.Errorf("database insert failed: %w", err)
    }
    return nil
}
```

## Error Testing

### Testing Expected Errors

```go
func TestUserService_Create_DuplicateUser(t *testing.T) {
    mockRepo := mocks.NewUserRepository(t)
    mockRepo.On("Create", mock.Anything, mock.Anything).
        Return(repository.ErrDuplicate)

    svc := NewUserService(mockRepo)

    err := svc.Create(context.Background(), user)

    require.Error(t, err)
    require.True(t, errors.Is(err, repository.ErrDuplicate))
    mockRepo.AssertExpectations(t)
}
```

### Testing Custom Error Types

```go
func TestValidateUser_InvalidAge(t *testing.T) {
    user := User{Name: "John", Age: -1}

    err := ValidateUser(user)

    require.Error(t, err)

    var validationErr *ValidationError
    require.True(t, errors.As(err, &validationErr))
    require.Equal(t, "age", validationErr.Field)
    require.Contains(t, validationErr.Message, "non-negative")
}
```

## Error Handling Anti-Patterns

### ❌ Don't Ignore Errors

```go
// Bad
data, _ := os.ReadFile("config.json")

// Good
data, err := os.ReadFile("config.json")
if err != nil {
    return fmt.Errorf("failed to read config: %w", err)
}
```

### ❌ Don't Use Panic for Expected Errors

```go
// Bad
func FindUser(id string) *User {
    user, err := db.Find(id)
    if err != nil {
        panic(err)  // Don't panic for expected errors
    }
    return user
}

// Good
func FindUser(id string) (*User, error) {
    user, err := db.Find(id)
    if err != nil {
        return nil, fmt.Errorf("failed to find user: %w", err)
    }
    return user, nil
}
```

### ❌ Don't Lose Error Context

```go
// Bad - Loses error chain
if err := operation(); err != nil {
    return errors.New("operation failed")
}

// Good - Preserves error chain
if err := operation(); err != nil {
    return fmt.Errorf("operation failed: %w", err)
}
```

### ❌ Don't Log and Return

```go
// Bad - Logs at every layer, causing duplicate logs
func (s *Service) Operation() error {
    if err := s.repo.Save(); err != nil {
        log.Printf("save failed: %v", err)  // Logged here
        return err  // And returned
    }
    return nil
}

// Good - Return error, log at top layer only
func (s *Service) Operation() error {
    if err := s.repo.Save(); err != nil {
        return fmt.Errorf("save failed: %w", err)
    }
    return nil
}
```

## Error Handling Checklist

- [ ] All errors are handled explicitly (never ignored)
- [ ] Errors wrapped with `%w` to preserve error chain
- [ ] Context added to errors as they propagate
- [ ] Sentinel errors defined for expected error conditions
- [ ] Custom error types used for complex error information
- [ ] `errors.Is()` used to check sentinel errors
- [ ] `errors.As()` used to check custom error types
- [ ] Defer used for cleanup
- [ ] Errors logged at appropriate layer (usually handler)
- [ ] HTTP status codes match error types
- [ ] Panic only used for programming errors
- [ ] Recover used at boundaries to catch panics
- [ ] Error tests verify both error occurrence and error type
