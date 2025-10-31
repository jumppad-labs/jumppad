# Go Code Organization

## Package Organization Principles

### Single Responsibility

Each package should have a single, well-defined purpose:

```go
// ✅ Good - Clear, focused packages
package user       // User domain logic
package auth       // Authentication
package email      // Email sending
package database   // Database connections

// ❌ Bad - Vague, unfocused packages
package utils      // Too broad
package helpers    // Unclear purpose
package common     // Everything ends up here
```

### Package Size

Keep packages reasonably sized:
- **Small packages** (100-500 lines): Easier to understand and maintain
- **Medium packages** (500-2000 lines): Still manageable
- **Large packages** (2000+ lines): Consider splitting

### Dependencies Flow

Dependencies should flow in one direction:

```
HTTP Handler → Service → Repository → Database
     ↓            ↓           ↓
  (depends)   (depends)   (depends)
```

**Never** create circular dependencies:

```go
// ❌ Bad - Circular dependency
package service
import "yourproject/repository"

package repository
import "yourproject/service"  // Circular!
```

## Interface Design

### Small, Focused Interfaces

Prefer small interfaces with few methods:

```go
// ✅ Good - Small, focused interfaces
type Reader interface {
    Read(p []byte) (n int, err error)
}

type Writer interface {
    Write(p []byte) (n int, err error)
}

type Closer interface {
    Close() error
}

// Compose when needed
type ReadWriteCloser interface {
    Reader
    Writer
    Closer
}

// ❌ Bad - Large, monolithic interface
type Database interface {
    Connect() error
    Close() error
    Query(sql string) (*Result, error)
    Insert(table string, data any) error
    Update(table string, data any) error
    Delete(table string, id string) error
    BeginTransaction() (*Transaction, error)
    // ... 20 more methods
}
```

### Interface Segregation

Create specific interfaces for specific needs:

```go
// ✅ Good - Specific interfaces
type UserFinder interface {
    FindByID(ctx context.Context, id string) (*User, error)
    FindByEmail(ctx context.Context, email string) (*User, error)
}

type UserCreator interface {
    Create(ctx context.Context, user User) error
}

type UserUpdater interface {
    Update(ctx context.Context, user User) error
}

// Service can depend only on what it needs
type UserService struct {
    finder UserFinder
    creator UserCreator
}

// ❌ Bad - One big interface
type UserRepository interface {
    FindByID(ctx context.Context, id string) (*User, error)
    FindByEmail(ctx context.Context, email string) (*User, error)
    FindAll(ctx context.Context) ([]User, error)
    Create(ctx context.Context, user User) error
    Update(ctx context.Context, user User) error
    Delete(ctx context.Context, id string) error
    Count(ctx context.Context) (int, error)
    // Service depends on everything even if it only needs one method
}
```

### Accept Interfaces, Return Structs

Functions should accept interfaces and return concrete types:

```go
// ✅ Good
func NewUserService(repo UserRepository) *UserService {
    return &UserService{repo: repo}
}

// ❌ Bad - Returns interface
func NewUserService(repo UserRepository) UserService {
    return &userService{repo: repo}
}
```

**Why?** Returning structs allows consumers to access all methods without type assertions.

### Define Interfaces in Consumer Package

Define interfaces where they're used, not where they're implemented:

```go
// ✅ Good
// internal/service/user_service.go
package service

// Service defines what it needs
type UserRepository interface {
    FindByID(ctx context.Context, id string) (*User, error)
    Create(ctx context.Context, user User) error
}

type UserService struct {
    repo UserRepository
}

// internal/repository/user_repository.go
package repository

// Repository just implements the interface
type userRepository struct {
    db *sql.DB
}

// Satisfies service.UserRepository interface implicitly
func (r *userRepository) FindByID(ctx context.Context, id string) (*User, error) {
    // implementation
}
```

## Dependency Injection

### Constructor Pattern

Use constructor functions for dependency injection:

```go
// ✅ Good - Clear dependencies
func NewUserService(repo UserRepository, cache Cache, logger Logger) *UserService {
    return &UserService{
        repo:   repo,
        cache:  cache,
        logger: logger,
    }
}

// Usage
repo := repository.NewUserRepository(db)
cache := cache.NewRedisCache(client)
logger := logger.NewStructuredLogger()
service := NewUserService(repo, cache, logger)
```

### Avoid Global State

Don't use package-level variables for dependencies:

```go
// ❌ Bad - Global state
package service

var repo repository.UserRepository

func Init(r repository.UserRepository) {
    repo = r
}

func CreateUser(user User) error {
    return repo.Create(user)
}

// ✅ Good - Injected dependencies
type UserService struct {
    repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) *UserService {
    return &UserService{repo: repo}
}

func (s *UserService) CreateUser(user User) error {
    return s.repo.Create(user)
}
```

### Options Pattern for Optional Dependencies

Use functional options for optional configuration:

```go
type UserService struct {
    repo   UserRepository
    cache  Cache
    logger Logger
}

type Option func(*UserService)

func WithCache(cache Cache) Option {
    return func(s *UserService) {
        s.cache = cache
    }
}

func WithLogger(logger Logger) Option {
    return func(s *UserService) {
        s.logger = logger
    }
}

func NewUserService(repo UserRepository, opts ...Option) *UserService {
    s := &UserService{
        repo:   repo,
        logger: defaultLogger,  // Default
    }

    for _, opt := range opts {
        opt(s)
    }

    return s
}

// Usage
service := NewUserService(
    repo,
    WithCache(cache),
    WithLogger(logger),
)
```

## Layered Architecture

### Three-Layer Pattern

Organize code into three layers:

```
┌─────────────────────────────────┐
│      Presentation Layer         │  HTTP handlers, gRPC handlers
│         (handler/)              │  Thin, delegates to service
└─────────────────────────────────┘
              ↓
┌─────────────────────────────────┐
│       Business Logic Layer      │  Business rules, validation
│         (service/)              │  Orchestrates repositories
└─────────────────────────────────┘
              ↓
┌─────────────────────────────────┐
│       Data Access Layer         │  Database, external APIs
│       (repository/)             │  No business logic
└─────────────────────────────────┘
```

### Handler Layer

Keep handlers thin - only handle HTTP concerns:

```go
// ✅ Good - Thin handler
type UserHandler struct {
    service UserService
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request", http.StatusBadRequest)
        return
    }

    user, err := h.service.CreateUser(r.Context(), req.ToUser())
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(user)
}

// ❌ Bad - Handler contains business logic
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    json.NewDecoder(r.Body).Decode(&req)

    // DON'T put business logic in handler
    if req.Age < 18 {
        http.Error(w, "must be 18+", http.StatusBadRequest)
        return
    }

    if !isValidEmail(req.Email) {
        http.Error(w, "invalid email", http.StatusBadRequest)
        return
    }

    // DON'T access database directly from handler
    db.Exec("INSERT INTO users ...")
}
```

### Service Layer

Business logic and orchestration:

```go
// ✅ Good - Service contains business logic
type UserService struct {
    repo  UserRepository
    email EmailSender
}

func (s *UserService) CreateUser(ctx context.Context, user User) (*User, error) {
    // Validation (business rules)
    if err := s.validateUser(user); err != nil {
        return nil, fmt.Errorf("validation failed: %w", err)
    }

    // Orchestrate multiple operations
    if err := s.repo.Create(ctx, user); err != nil {
        return nil, fmt.Errorf("failed to create user: %w", err)
    }

    // Send welcome email (don't wait for it)
    go s.email.SendWelcome(user.Email)

    return &user, nil
}

func (s *UserService) validateUser(user User) error {
    if user.Age < 18 {
        return errors.New("must be 18 or older")
    }
    if !isValidEmail(user.Email) {
        return errors.New("invalid email")
    }
    return nil
}
```

### Repository Layer

Pure data access, no business logic:

```go
// ✅ Good - Repository only handles data access
type userRepository struct {
    db *sql.DB
}

func (r *userRepository) Create(ctx context.Context, user User) error {
    query := `INSERT INTO users (id, name, email, age) VALUES ($1, $2, $3, $4)`
    _, err := r.db.ExecContext(ctx, query, user.ID, user.Name, user.Email, user.Age)
    if err != nil {
        return fmt.Errorf("failed to insert user: %w", err)
    }
    return nil
}

func (r *userRepository) FindByID(ctx context.Context, id string) (*User, error) {
    var user User
    query := `SELECT id, name, email, age FROM users WHERE id = $1`
    err := r.db.QueryRowContext(ctx, query, id).Scan(&user.ID, &user.Name, &user.Email, &user.Age)
    if err == sql.ErrNoRows {
        return nil, ErrNotFound
    }
    if err != nil {
        return nil, fmt.Errorf("failed to query user: %w", err)
    }
    return &user, nil
}

// ❌ Bad - Repository contains business logic
func (r *userRepository) Create(ctx context.Context, user User) error {
    // DON'T validate in repository
    if user.Age < 18 {
        return errors.New("must be 18+")
    }

    query := `INSERT INTO users ...`
    _, err := r.db.ExecContext(ctx, query, ...)
    return err
}
```

## Error Handling Strategy

### Define Package-Level Sentinel Errors

```go
// internal/service/errors.go
package service

import "errors"

var (
    ErrNotFound      = errors.New("not found")
    ErrUnauthorized  = errors.New("unauthorized")
    ErrInvalidInput  = errors.New("invalid input")
    ErrConflict      = errors.New("already exists")
)
```

### Wrap Errors with Context

```go
// ✅ Good - Wrap errors with context
func (s *UserService) CreateUser(ctx context.Context, user User) error {
    if err := s.validateUser(user); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }

    if err := s.repo.Create(ctx, user); err != nil {
        return fmt.Errorf("failed to create user: %w", err)
    }

    return nil
}

// Consumer can check the root cause
err := service.CreateUser(ctx, user)
if errors.Is(err, repository.ErrDuplicate) {
    // Handle duplicate
}
```

See [Error Handling](error-handling.md) for detailed error patterns.

## Context Usage

### Always Pass Context

Pass `context.Context` as the first parameter:

```go
// ✅ Good
func (s *UserService) CreateUser(ctx context.Context, user User) error {}
func (r *UserRepository) FindByID(ctx context.Context, id string) (*User, error) {}

// ❌ Bad - Missing context
func (s *UserService) CreateUser(user User) error {}
```

### Use Context for Cancellation

```go
func (s *UserService) ProcessUsers(ctx context.Context) error {
    users, err := s.repo.FindAll(ctx)
    if err != nil {
        return err
    }

    for _, user := range users {
        // Check for cancellation
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }

        if err := s.processUser(ctx, user); err != nil {
            return err
        }
    }

    return nil
}
```

### Don't Store Context in Structs

```go
// ❌ Bad - Don't store context
type UserService struct {
    ctx  context.Context  // DON'T DO THIS
    repo UserRepository
}

// ✅ Good - Pass context to methods
type UserService struct {
    repo UserRepository
}

func (s *UserService) CreateUser(ctx context.Context, user User) error {
    // Use ctx parameter
}
```

## Domain Models

### Keep Models in Separate Package

```go
// internal/model/user.go
package model

import "time"

type User struct {
    ID        string
    Name      string
    Email     string
    Age       int
    CreatedAt time.Time
    UpdatedAt time.Time
}

func (u *User) IsAdult() bool {
    return u.Age >= 18
}

func (u *User) Validate() error {
    if u.Name == "" {
        return errors.New("name is required")
    }
    if u.Email == "" {
        return errors.New("email is required")
    }
    return nil
}
```

### Use Value Objects for Complex Types

```go
// internal/model/email.go
package model

import (
    "errors"
    "regexp"
)

type Email string

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func NewEmail(s string) (Email, error) {
    if !emailRegex.MatchString(s) {
        return "", errors.New("invalid email format")
    }
    return Email(s), nil
}

func (e Email) String() string {
    return string(e)
}

func (e Email) Domain() string {
    parts := strings.Split(string(e), "@")
    if len(parts) != 2 {
        return ""
    }
    return parts[1]
}
```

## Composition Over Inheritance

Go doesn't have inheritance. Use composition:

```go
// ✅ Good - Composition
type Logger interface {
    Log(message string)
}

type Service struct {
    logger Logger
}

func (s *Service) DoSomething() {
    s.logger.Log("doing something")
}

// Embedding for shared behavior
type BaseService struct {
    logger Logger
}

func (b *BaseService) Log(message string) {
    b.logger.Log(message)
}

type UserService struct {
    BaseService  // Embedded
    repo UserRepository
}

// UserService has Log method from BaseService
```

## Code Organization Checklist

- [ ] Each package has single, clear purpose
- [ ] No circular dependencies
- [ ] Interfaces are small and focused
- [ ] Interfaces defined in consumer package
- [ ] Dependencies injected via constructors
- [ ] No global state
- [ ] Handlers are thin (only HTTP concerns)
- [ ] Business logic in service layer
- [ ] Data access in repository layer
- [ ] Models in separate package
- [ ] Context passed as first parameter
- [ ] Errors wrapped with context
- [ ] Using composition, not inheritance
