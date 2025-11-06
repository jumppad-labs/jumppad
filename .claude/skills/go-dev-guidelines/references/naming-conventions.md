# Go Naming Conventions

## General Principles

Go emphasizes clarity and simplicity in naming. Follow these core principles:

1. **Be consistent** - Follow Go community conventions
2. **Be clear** - Names should reveal intent
3. **Be concise** - Shorter is better when meaning is clear
4. **Avoid stuttering** - Don't repeat package name in type names
5. **Use MixedCaps** - Not underscores (except in test names and file names)

## Package Names

### Rules
- **Lowercase only** - No uppercase, underscores, or mixedCaps
- **Short** - Prefer single word when possible
- **Plural not needed** - Use `user` not `users`, `http` not `https`
- **Descriptive** - Should describe what the package provides

### ✅ Good Package Names
```go
package http
package user
package auth
package handler
package service
package repository
```

### ❌ Bad Package Names
```go
package HTTPHandler    // Don't use uppercase
package user_service   // Don't use underscores
package utils          // Too generic
package helpers        // Too generic
package common         // Too vague
```

### Import Aliasing
Only alias imports when necessary to avoid conflicts:

```go
// Only when needed
import (
    "crypto/rand"
    mathrand "math/rand"  // Alias to avoid conflict
)
```

## File Names

### Rules
- **Lowercase** with underscores separating words
- **Match content** - Name should reflect primary type or function
- **Test suffix** - Test files end with `_test.go`

### ✅ Good File Names
```go
user_service.go
user_service_test.go
http_handler.go
auth_middleware.go
database.go
```

### ❌ Bad File Names
```go
UserService.go         // Don't use uppercase
user-service.go        // Use underscores, not hyphens
userservice.go         // Hard to read without separator
```

## Type Names

### Structs

Use **PascalCase** (MixedCaps). Start with uppercase for exported, lowercase for unexported.

```go
// Exported types
type UserService struct {}
type HTTPClient struct {}
type Config struct {}

// Unexported types
type internalCache struct {}
type requestContext struct {}
```

### Avoid Stuttering

Don't include package name in type names:

```go
// ✅ Good
package user
type Service struct {}        // Use as user.Service
type Repository struct {}     // Use as user.Repository

// ❌ Bad
package user
type UserService struct {}    // Stutters: user.UserService
type UserRepository struct {} // Stutters: user.UserRepository
```

### Acronyms and Initialisms

Keep acronyms uppercase in names:

```go
// ✅ Good
type HTTPServer struct {}
type URLParser struct {}
type IDGenerator struct {}
type APIClient struct {}

// ❌ Bad
type HttpServer struct {}
type UrlParser struct {}
type IdGenerator struct {}
type ApiClient struct {}
```

**Exception:** When acronym is at the start of an unexported name, use lowercase:

```go
type httpClient struct {}  // ✅ Good - unexported
type HTTPClient struct {}  // ✅ Good - exported
```

## Interface Names

### Single-Method Interfaces

Use `-er` suffix for single-method interfaces:

```go
type Reader interface {
    Read(p []byte) (n int, err error)
}

type Writer interface {
    Write(p []byte) (n int, err error)
}

type Closer interface {
    Close() error
}

type Stringer interface {
    String() string
}
```

### Multi-Method Interfaces

Use descriptive names without `-er` suffix:

```go
type UserRepository interface {
    Create(user User) error
    FindByID(id string) (*User, error)
    Update(user User) error
    Delete(id string) error
}

type Cache interface {
    Get(key string) (any, error)
    Set(key string, value any) error
    Delete(key string) error
}
```

### Interface Location

Define interfaces in the consumer package, not the implementation package:

```go
// ✅ Good
// internal/service/service.go - defines interface it needs
type UserRepository interface {
    FindByID(id string) (*User, error)
}

type UserService struct {
    repo UserRepository
}

// internal/repository/user_repository.go - implements it
type userRepository struct {}

func (r *userRepository) FindByID(id string) (*User, error) {
    // implementation
}

// ❌ Bad
// internal/repository/repository.go - defines interface with implementation
type UserRepository interface {
    FindByID(id string) (*User, error)
}
```

## Function and Method Names

### Rules
- **PascalCase** for exported functions/methods
- **camelCase** for unexported functions/methods
- **Start with verb** when performing an action
- **Be descriptive** but concise

### ✅ Good Function Names

```go
// Exported functions
func NewUserService() *UserService {}
func CreateUser(name string) error {}
func GetUserByID(id string) (*User, error) {}
func UpdatePassword(userID, password string) error {}
func IsValidEmail(email string) bool {}

// Unexported functions
func validateInput(input string) error {}
func parseRequest(r *http.Request) (*Request, error) {}
func buildQuery(filters map[string]any) string {}
```

### ❌ Bad Function Names

```go
func user_service() {}           // Don't use underscores
func get_user_by_id() {}         // Don't use underscores
func GetUserById() {}            // Use ID not Id
func GetUserByIdFromDatabase() {} // Too verbose
func Process() {}                // Too vague
func DoStuff() {}                // Meaningless
```

## Variable Names

### General Variables

Use short names for local variables with limited scope:

```go
// ✅ Good - short scope
func ProcessUser(user User) error {
    u := user.Normalize()
    if err := u.Validate(); err != nil {
        return err
    }
    return nil
}

// ✅ Good - longer name for wider scope
type UserService struct {
    repository UserRepository
    cache      Cache
    logger     Logger
}
```

### Common Short Names

Use conventional short names:

```go
i, j, k       // Loop indices
n, m          // Counts or dimensions
c, ch         // Channels
r, w          // Readers and Writers
req, resp     // Request and Response
ctx           // Context
err           // Errors
b             // Byte slice
s             // String
```

### Receiver Names

Use short, consistent receiver names (1-2 characters):

```go
// ✅ Good
func (u *UserService) Create(user User) error {}
func (u *UserService) FindByID(id string) (*User, error) {}

// ✅ Also good for longer names
func (us *UserService) Create(user User) error {}

// ❌ Bad
func (userService *UserService) Create(user User) error {}
func (this *UserService) Create(user User) error {}
func (self *UserService) Create(user User) error {}
```

**Rules for receivers:**
- Use same name for all methods on a type
- Typically 1-2 character abbreviation of type name
- Avoid generic names like `this`, `self`, `me`

### Constants

Use PascalCase for exported constants, camelCase for unexported:

```go
// Exported constants
const (
    MaxConnections = 100
    DefaultTimeout = 30 * time.Second
    StatusActive   = "active"
)

// Unexported constants
const (
    maxRetries     = 3
    defaultBufSize = 4096
)
```

### Boolean Variables

Prefix boolean variables with `is`, `has`, `can`, `should`:

```go
// ✅ Good
isValid := true
hasPermission := user.IsAdmin()
canWrite := file.IsWritable()
shouldRetry := err != nil

// ❌ Bad
valid := true
permission := user.IsAdmin()
```

## Error Variables

### Error Variables

Use `Err` prefix for sentinel errors:

```go
var (
    ErrNotFound      = errors.New("not found")
    ErrUnauthorized  = errors.New("unauthorized")
    ErrInvalidInput  = errors.New("invalid input")
)
```

### Error Types

Use `Error` suffix for custom error types:

```go
type ValidationError struct {
    Field string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

type DatabaseError struct {
    Op    string
    Err   error
}

func (e *DatabaseError) Error() string {
    return fmt.Sprintf("database %s: %v", e.Op, e.Err)
}
```

## Test Names

### Test Functions

Use `Test` prefix followed by the function/method being tested:

```go
func TestUserService_Create(t *testing.T) {}
func TestUserService_Create_InvalidInput(t *testing.T) {}
func TestValidateEmail(t *testing.T) {}
func TestValidateEmail_EmptyString(t *testing.T) {}
```

### Benchmark Functions

Use `Benchmark` prefix:

```go
func BenchmarkUserService_Create(b *testing.B) {}
func BenchmarkValidateEmail(b *testing.B) {}
```

### Example Functions

Use `Example` prefix:

```go
func ExampleUserService_Create() {}
func ExampleValidateEmail() {}
```

### Test Helper Functions

Helper functions can use underscores for readability:

```go
func setup_test_database(t *testing.T) *sql.DB {}
func teardown_test_database(t *testing.T, db *sql.DB) {}
func create_test_user(name string) User {}
```

## Package-Level Variables

Use descriptive names for package-level variables:

```go
// ✅ Good
var (
    DefaultConfig = Config{
        Timeout: 30 * time.Second,
        MaxRetries: 3,
    }

    ErrNotFound = errors.New("not found")
)

// ❌ Bad
var (
    config = Config{}  // Too generic
    Cfg = Config{}     // Abbreviation at package level
)
```

## Type Aliases

Use descriptive names for type aliases:

```go
// ✅ Good
type UserID string
type OrderID int64
type Timestamp int64

// ❌ Bad
type UID string        // Not clear
type ID any           // Too generic
```

## Method Sets and Getter/Setter Naming

### Getters

**Don't use** `Get` prefix for getters in Go:

```go
// ✅ Good
func (u *User) Name() string { return u.name }
func (u *User) Email() string { return u.email }
func (u *User) Age() int { return u.age }

// ❌ Bad
func (u *User) GetName() string { return u.name }
func (u *User) GetEmail() string { return u.email }
```

### Setters

**Do use** `Set` prefix for setters:

```go
// ✅ Good
func (u *User) SetName(name string) { u.name = name }
func (u *User) SetEmail(email string) { u.email = email }
```

### Boolean Getters

Use `Is`, `Has`, `Can` prefixes for boolean getters:

```go
// ✅ Good
func (u *User) IsActive() bool { return u.active }
func (u *User) HasPermission(p Permission) bool {}
func (u *User) CanWrite() bool {}
```

## Naming Anti-Patterns

### ❌ Avoid Generic Names

```go
// Bad
var data any
var info map[string]string
var obj Object
var mgr Manager
var tmp string

// Good
var users []User
var config map[string]string
var request Request
var service UserService
var normalized string
```

### ❌ Avoid Redundant Names

```go
// Bad
type UserStruct struct {}      // Redundant "Struct"
type IUserRepository interface {} // Don't use "I" prefix
var userVariable User          // Redundant "Variable"

// Good
type User struct {}
type UserRepository interface {}
var user User
```

### ❌ Avoid Meaningless Names

```go
// Bad
func DoStuff() {}
func Process() {}
func Handle() {}
func Manager() {}

// Good
func ProcessOrder() {}
func HandleRequest() {}
func ManageUsers() {}
```

## Summary Checklist

- [ ] Package names are lowercase, single word when possible
- [ ] File names use lowercase with underscores
- [ ] Types use PascalCase
- [ ] Interfaces use `-er` suffix for single-method interfaces
- [ ] Functions/methods use PascalCase (exported) or camelCase (unexported)
- [ ] Variables use short names in limited scope
- [ ] Constants use PascalCase
- [ ] Error variables have `Err` prefix
- [ ] Error types have `Error` suffix
- [ ] Test functions have `Test` prefix
- [ ] No stuttering (package.PackageType)
- [ ] Acronyms stay uppercase (HTTP, URL, ID)
- [ ] Getter methods don't use `Get` prefix
- [ ] Boolean methods use `Is`, `Has`, `Can` prefixes
