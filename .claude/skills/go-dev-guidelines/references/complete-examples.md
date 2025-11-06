# Complete Go Examples

This document provides complete, working examples following all the Go development guidelines.

## Example 1: REST API with Layered Architecture

A complete example showing handler → service → repository pattern with TDD.

### Domain Model

```go
// internal/model/user.go
package model

import (
    "errors"
    "time"
)

type User struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    Email     string    `json:"email"`
    Age       int       `json:"age"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

func (u *User) Validate() error {
    if u.Name == "" {
        return errors.New("name is required")
    }
    if u.Email == "" {
        return errors.New("email is required")
    }
    if u.Age < 0 {
        return errors.New("age must be non-negative")
    }
    return nil
}

func (u *User) IsAdult() bool {
    return u.Age >= 18
}
```

### Repository Interface and Implementation

```go
// internal/repository/user_repository.go
package repository

import (
    "context"
    "database/sql"
    "errors"
    "fmt"

    "yourproject/internal/model"
)

var (
    ErrNotFound  = errors.New("user not found")
    ErrDuplicate = errors.New("user already exists")
)

type UserRepository interface {
    Create(ctx context.Context, user *model.User) error
    FindByID(ctx context.Context, id string) (*model.User, error)
    FindByEmail(ctx context.Context, email string) (*model.User, error)
    Update(ctx context.Context, user *model.User) error
    Delete(ctx context.Context, id string) error
}

type userRepository struct {
    db *sql.DB
}

func NewUserRepository(db *sql.DB) *userRepository {
    return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *model.User) error {
    query := `
        INSERT INTO users (id, name, email, age, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6)
    `

    _, err := r.db.ExecContext(ctx, query,
        user.ID, user.Name, user.Email, user.Age, user.CreatedAt, user.UpdatedAt)

    if err != nil {
        // Check for unique constraint violation
        if isDuplicateError(err) {
            return ErrDuplicate
        }
        return fmt.Errorf("failed to insert user: %w", err)
    }

    return nil
}

func (r *userRepository) FindByID(ctx context.Context, id string) (*model.User, error) {
    query := `
        SELECT id, name, email, age, created_at, updated_at
        FROM users
        WHERE id = $1
    `

    var user model.User
    err := r.db.QueryRowContext(ctx, query, id).Scan(
        &user.ID, &user.Name, &user.Email, &user.Age, &user.CreatedAt, &user.UpdatedAt)

    if err == sql.ErrNoRows {
        return nil, ErrNotFound
    }
    if err != nil {
        return nil, fmt.Errorf("failed to query user: %w", err)
    }

    return &user, nil
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
    query := `
        SELECT id, name, email, age, created_at, updated_at
        FROM users
        WHERE email = $1
    `

    var user model.User
    err := r.db.QueryRowContext(ctx, query, email).Scan(
        &user.ID, &user.Name, &user.Email, &user.Age, &user.CreatedAt, &user.UpdatedAt)

    if err == sql.ErrNoRows {
        return nil, ErrNotFound
    }
    if err != nil {
        return nil, fmt.Errorf("failed to query user: %w", err)
    }

    return &user, nil
}

func (r *userRepository) Update(ctx context.Context, user *model.User) error {
    query := `
        UPDATE users
        SET name = $2, email = $3, age = $4, updated_at = $5
        WHERE id = $1
    `

    result, err := r.db.ExecContext(ctx, query,
        user.ID, user.Name, user.Email, user.Age, user.UpdatedAt)

    if err != nil {
        return fmt.Errorf("failed to update user: %w", err)
    }

    rows, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %w", err)
    }

    if rows == 0 {
        return ErrNotFound
    }

    return nil
}

func (r *userRepository) Delete(ctx context.Context, id string) error {
    query := `DELETE FROM users WHERE id = $1`

    result, err := r.db.ExecContext(ctx, query, id)
    if err != nil {
        return fmt.Errorf("failed to delete user: %w", err)
    }

    rows, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %w", err)
    }

    if rows == 0 {
        return ErrNotFound
    }

    return nil
}

func isDuplicateError(err error) bool {
    // PostgreSQL unique violation error code
    // This is database-specific
    return err != nil && err.Error() == "pq: duplicate key value violates unique constraint"
}
```

### Repository Tests

```go
// internal/repository/user_repository_test.go
package repository

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/require"

    "yourproject/internal/model"
)

func TestUserRepository_Create(t *testing.T) {
    db := setupTestDB(t)
    defer teardownTestDB(t, db)

    repo := NewUserRepository(db)
    user := &model.User{
        ID:        "user-123",
        Name:      "John Doe",
        Email:     "john@example.com",
        Age:       30,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }

    err := repo.Create(context.Background(), user)

    require.NoError(t, err)
}

func TestUserRepository_Create_DuplicateEmail(t *testing.T) {
    db := setupTestDB(t)
    defer teardownTestDB(t, db)

    repo := NewUserRepository(db)
    user := &model.User{
        ID:        "user-123",
        Name:      "John Doe",
        Email:     "john@example.com",
        Age:       30,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }

    err := repo.Create(context.Background(), user)
    require.NoError(t, err)

    user2 := &model.User{
        ID:        "user-456",
        Name:      "Jane Doe",
        Email:     "john@example.com",  // Same email
        Age:       25,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }

    err = repo.Create(context.Background(), user2)

    require.Error(t, err)
    require.ErrorIs(t, err, ErrDuplicate)
}

func TestUserRepository_FindByID(t *testing.T) {
    db := setupTestDB(t)
    defer teardownTestDB(t, db)

    repo := NewUserRepository(db)
    user := &model.User{
        ID:        "user-123",
        Name:      "John Doe",
        Email:     "john@example.com",
        Age:       30,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }

    err := repo.Create(context.Background(), user)
    require.NoError(t, err)

    found, err := repo.FindByID(context.Background(), "user-123")

    require.NoError(t, err)
    require.NotNil(t, found)
    require.Equal(t, user.ID, found.ID)
    require.Equal(t, user.Name, found.Name)
}

func TestUserRepository_FindByID_NotFound(t *testing.T) {
    db := setupTestDB(t)
    defer teardownTestDB(t, db)

    repo := NewUserRepository(db)

    found, err := repo.FindByID(context.Background(), "nonexistent")

    require.Error(t, err)
    require.ErrorIs(t, err, ErrNotFound)
    require.Nil(t, found)
}
```

### Service Layer

```go
// internal/service/user_service.go
package service

import (
    "context"
    "errors"
    "fmt"
    "time"

    "github.com/google/uuid"

    "yourproject/internal/model"
    "yourproject/internal/repository"
)

var (
    ErrNotFound     = errors.New("user not found")
    ErrInvalidInput = errors.New("invalid input")
    ErrDuplicate    = errors.New("user already exists")
)

type UserRepository interface {
    Create(ctx context.Context, user *model.User) error
    FindByID(ctx context.Context, id string) (*model.User, error)
    FindByEmail(ctx context.Context, email string) (*model.User, error)
    Update(ctx context.Context, user *model.User) error
    Delete(ctx context.Context, id string) error
}

type UserService struct {
    repo UserRepository
}

func NewUserService(repo UserRepository) *UserService {
    return &UserService{repo: repo}
}

func (s *UserService) CreateUser(ctx context.Context, name, email string, age int) (*model.User, error) {
    user := &model.User{
        ID:        uuid.New().String(),
        Name:      name,
        Email:     email,
        Age:       age,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }

    if err := user.Validate(); err != nil {
        return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
    }

    if err := s.repo.Create(ctx, user); err != nil {
        if errors.Is(err, repository.ErrDuplicate) {
            return nil, ErrDuplicate
        }
        return nil, fmt.Errorf("failed to create user: %w", err)
    }

    return user, nil
}

func (s *UserService) GetUserByID(ctx context.Context, id string) (*model.User, error) {
    user, err := s.repo.FindByID(ctx, id)
    if err != nil {
        if errors.Is(err, repository.ErrNotFound) {
            return nil, ErrNotFound
        }
        return nil, fmt.Errorf("failed to find user: %w", err)
    }

    return user, nil
}

func (s *UserService) UpdateUser(ctx context.Context, id, name, email string, age int) (*model.User, error) {
    user, err := s.repo.FindByID(ctx, id)
    if err != nil {
        if errors.Is(err, repository.ErrNotFound) {
            return nil, ErrNotFound
        }
        return nil, fmt.Errorf("failed to find user: %w", err)
    }

    user.Name = name
    user.Email = email
    user.Age = age
    user.UpdatedAt = time.Now()

    if err := user.Validate(); err != nil {
        return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
    }

    if err := s.repo.Update(ctx, user); err != nil {
        return nil, fmt.Errorf("failed to update user: %w", err)
    }

    return user, nil
}

func (s *UserService) DeleteUser(ctx context.Context, id string) error {
    if err := s.repo.Delete(ctx, id); err != nil {
        if errors.Is(err, repository.ErrNotFound) {
            return ErrNotFound
        }
        return fmt.Errorf("failed to delete user: %w", err)
    }

    return nil
}
```

### Service Tests with Mocks

```go
// internal/service/user_service_test.go
package service

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/require"

    "yourproject/internal/model"
    "yourproject/internal/repository"
    "yourproject/internal/service/mocks"
)

func TestUserService_CreateUser(t *testing.T) {
    mockRepo := mocks.NewUserRepository(t)
    mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(u *model.User) bool {
        return u.Name == "John Doe" && u.Email == "john@example.com"
    })).Return(nil)

    svc := NewUserService(mockRepo)

    user, err := svc.CreateUser(context.Background(), "John Doe", "john@example.com", 30)

    require.NoError(t, err)
    require.NotNil(t, user)
    require.Equal(t, "John Doe", user.Name)
    require.Equal(t, "john@example.com", user.Email)
    require.NotEmpty(t, user.ID)
    mockRepo.AssertExpectations(t)
}

func TestUserService_CreateUser_InvalidInput(t *testing.T) {
    svc := NewUserService(nil)  // No repo needed for validation

    user, err := svc.CreateUser(context.Background(), "", "john@example.com", 30)

    require.Error(t, err)
    require.ErrorIs(t, err, ErrInvalidInput)
    require.Nil(t, user)
}

func TestUserService_CreateUser_DuplicateEmail(t *testing.T) {
    mockRepo := mocks.NewUserRepository(t)
    mockRepo.On("Create", mock.Anything, mock.Anything).Return(repository.ErrDuplicate)

    svc := NewUserService(mockRepo)

    user, err := svc.CreateUser(context.Background(), "John Doe", "john@example.com", 30)

    require.Error(t, err)
    require.ErrorIs(t, err, ErrDuplicate)
    require.Nil(t, user)
    mockRepo.AssertExpectations(t)
}

func TestUserService_GetUserByID(t *testing.T) {
    expectedUser := &model.User{
        ID:        "user-123",
        Name:      "John Doe",
        Email:     "john@example.com",
        Age:       30,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }

    mockRepo := mocks.NewUserRepository(t)
    mockRepo.On("FindByID", mock.Anything, "user-123").Return(expectedUser, nil)

    svc := NewUserService(mockRepo)

    user, err := svc.GetUserByID(context.Background(), "user-123")

    require.NoError(t, err)
    require.Equal(t, expectedUser, user)
    mockRepo.AssertExpectations(t)
}

func TestUserService_GetUserByID_NotFound(t *testing.T) {
    mockRepo := mocks.NewUserRepository(t)
    mockRepo.On("FindByID", mock.Anything, "nonexistent").Return(nil, repository.ErrNotFound)

    svc := NewUserService(mockRepo)

    user, err := svc.GetUserByID(context.Background(), "nonexistent")

    require.Error(t, err)
    require.ErrorIs(t, err, ErrNotFound)
    require.Nil(t, user)
    mockRepo.AssertExpectations(t)
}
```

### HTTP Handler

```go
// internal/handler/user_handler.go
package handler

import (
    "encoding/json"
    "errors"
    "log/slog"
    "net/http"

    "github.com/gorilla/mux"

    "yourproject/internal/service"
)

type UserService interface {
    CreateUser(ctx context.Context, name, email string, age int) (*model.User, error)
    GetUserByID(ctx context.Context, id string) (*model.User, error)
    UpdateUser(ctx context.Context, id, name, email string, age int) (*model.User, error)
    DeleteUser(ctx context.Context, id string) error
}

type UserHandler struct {
    service UserService
}

func NewUserHandler(service UserService) *UserHandler {
    return &UserHandler{service: service}
}

type CreateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
    Age   int    `json:"age"`
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request body", http.StatusBadRequest)
        return
    }

    user, err := h.service.CreateUser(r.Context(), req.Name, req.Email, req.Age)
    if err != nil {
        h.handleError(w, r, err)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(user)
}

func (h *UserHandler) GetByID(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]

    user, err := h.service.GetUserByID(r.Context(), id)
    if err != nil {
        h.handleError(w, r, err)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(user)
}

func (h *UserHandler) handleError(w http.ResponseWriter, r *http.Request, err error) {
    if errors.Is(err, service.ErrNotFound) {
        http.Error(w, "user not found", http.StatusNotFound)
        return
    }

    if errors.Is(err, service.ErrInvalidInput) {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    if errors.Is(err, service.ErrDuplicate) {
        http.Error(w, "user already exists", http.StatusConflict)
        return
    }

    slog.Error("internal error", "error", err, "path", r.URL.Path)
    http.Error(w, "internal server error", http.StatusInternalServerError)
}
```

### Handler Tests

```go
// internal/handler/user_handler_test.go
package handler

import (
    "bytes"
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"

    "github.com/gorilla/mux"
    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/require"

    "yourproject/internal/handler/mocks"
    "yourproject/internal/model"
    "yourproject/internal/service"
)

func TestUserHandler_Create(t *testing.T) {
    expectedUser := &model.User{
        ID:        "user-123",
        Name:      "John Doe",
        Email:     "john@example.com",
        Age:       30,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }

    mockService := mocks.NewUserService(t)
    mockService.On("CreateUser", mock.Anything, "John Doe", "john@example.com", 30).
        Return(expectedUser, nil)

    handler := NewUserHandler(mockService)

    body := `{"name":"John Doe","email":"john@example.com","age":30}`
    req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader([]byte(body)))
    rec := httptest.NewRecorder()

    handler.Create(rec, req)

    require.Equal(t, http.StatusCreated, rec.Code)

    var user model.User
    err := json.NewDecoder(rec.Body).Decode(&user)
    require.NoError(t, err)
    require.Equal(t, expectedUser.ID, user.ID)
    require.Equal(t, expectedUser.Name, user.Name)
    mockService.AssertExpectations(t)
}

func TestUserHandler_Create_InvalidJSON(t *testing.T) {
    handler := NewUserHandler(nil)

    req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader([]byte("invalid")))
    rec := httptest.NewRecorder()

    handler.Create(rec, req)

    require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUserHandler_Create_InvalidInput(t *testing.T) {
    mockService := mocks.NewUserService(t)
    mockService.On("CreateUser", mock.Anything, "", "john@example.com", 30).
        Return(nil, service.ErrInvalidInput)

    handler := NewUserHandler(mockService)

    body := `{"name":"","email":"john@example.com","age":30}`
    req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader([]byte(body)))
    rec := httptest.NewRecorder()

    handler.Create(rec, req)

    require.Equal(t, http.StatusBadRequest, rec.Code)
    mockService.AssertExpectations(t)
}

func TestUserHandler_GetByID(t *testing.T) {
    expectedUser := &model.User{
        ID:        "user-123",
        Name:      "John Doe",
        Email:     "john@example.com",
        Age:       30,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }

    mockService := mocks.NewUserService(t)
    mockService.On("GetUserByID", mock.Anything, "user-123").Return(expectedUser, nil)

    handler := NewUserHandler(mockService)

    req := httptest.NewRequest(http.MethodGet, "/users/user-123", nil)
    req = mux.SetURLVars(req, map[string]string{"id": "user-123"})
    rec := httptest.NewRecorder()

    handler.GetByID(rec, req)

    require.Equal(t, http.StatusOK, rec.Code)

    var user model.User
    err := json.NewDecoder(rec.Body).Decode(&user)
    require.NoError(t, err)
    require.Equal(t, expectedUser.ID, user.ID)
    mockService.AssertExpectations(t)
}

func TestUserHandler_GetByID_NotFound(t *testing.T) {
    mockService := mocks.NewUserService(t)
    mockService.On("GetUserByID", mock.Anything, "nonexistent").Return(nil, service.ErrNotFound)

    handler := NewUserHandler(mockService)

    req := httptest.NewRequest(http.MethodGet, "/users/nonexistent", nil)
    req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
    rec := httptest.NewRecorder()

    handler.GetByID(rec, req)

    require.Equal(t, http.StatusNotFound, rec.Code)
    mockService.AssertExpectations(t)
}
```

### Main Application

```go
// cmd/server/main.go
package main

import (
    "context"
    "database/sql"
    "log"
    "log/slog"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/gorilla/mux"
    _ "github.com/lib/pq"

    "yourproject/internal/handler"
    "yourproject/internal/repository"
    "yourproject/internal/service"
)

func main() {
    // Setup database
    db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Dependency injection
    userRepo := repository.NewUserRepository(db)
    userService := service.NewUserService(userRepo)
    userHandler := handler.NewUserHandler(userService)

    // Setup router
    r := mux.NewRouter()
    r.HandleFunc("/users", userHandler.Create).Methods(http.MethodPost)
    r.HandleFunc("/users/{id}", userHandler.GetByID).Methods(http.MethodGet)

    // Setup server
    srv := &http.Server{
        Addr:         ":8080",
        Handler:      r,
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    // Start server
    go func() {
        slog.Info("server starting", "addr", srv.Addr)
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatal(err)
        }
    }()

    // Graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    slog.Info("server shutting down")

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        log.Fatal(err)
    }

    slog.Info("server stopped")
}
```

This complete example demonstrates:
- ✅ Layered architecture (handler → service → repository)
- ✅ Dependency injection
- ✅ Interface-based design
- ✅ Comprehensive tests with testify/require and mockery
- ✅ Separate positive and negative tests
- ✅ No table-driven tests
- ✅ Proper error handling with wrapping
- ✅ Context usage throughout
- ✅ Graceful shutdown
- ✅ Idiomatic Go naming and structure
