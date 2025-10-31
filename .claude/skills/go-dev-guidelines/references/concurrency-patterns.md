# Go Concurrency Patterns

## Concurrency Principles

1. **Don't communicate by sharing memory; share memory by communicating**
2. **Goroutines are cheap, but not free** - Use them appropriately
3. **Always handle goroutine lifecycle** - Know when goroutines exit
4. **Use context for cancellation** - Propagate cancellation through call stack
5. **Avoid goroutine leaks** - Ensure all goroutines terminate

## Goroutines

### Launching Goroutines

```go
// ✅ Good - Simple goroutine launch
go func() {
    result := doWork()
    log.Printf("work completed: %v", result)
}()

// ✅ Good - Goroutine with parameters
go func(id string, data Data) {
    process(id, data)
}(userID, userData)

// ❌ Bad - Goroutine captures loop variable incorrectly
for _, user := range users {
    go func() {
        process(user)  // Bug: all goroutines may see the same user
    }()
}

// ✅ Good - Pass loop variable as parameter
for _, user := range users {
    go func(u User) {
        process(u)
    }(user)
}
```

### Goroutine Lifecycle Management

Always ensure goroutines can exit:

```go
// ✅ Good - Goroutine with context for cancellation
func (s *Service) Start(ctx context.Context) error {
    go func() {
        ticker := time.NewTicker(time.Minute)
        defer ticker.Stop()

        for {
            select {
            case <-ctx.Done():
                log.Println("worker stopping")
                return
            case <-ticker.C:
                s.doPeriodicWork()
            }
        }
    }()

    return nil
}

// ❌ Bad - Goroutine with no way to stop
func (s *Service) Start() {
    go func() {
        for {
            time.Sleep(time.Minute)
            s.doPeriodicWork()  // Runs forever, no way to stop
        }
    }()
}
```

## Channels

### Channel Basics

```go
// Unbuffered channel (synchronous)
ch := make(chan int)

// Buffered channel (asynchronous up to capacity)
ch := make(chan int, 100)

// Send
ch <- value

// Receive
value := <-ch

// Receive with ok check (detects closed channel)
value, ok := <-ch
if !ok {
    // Channel closed
}

// Close channel (only sender should close)
close(ch)
```

### Channel Direction

Specify channel direction in function signatures:

```go
// ✅ Good - Explicit channel directions
func producer(ch chan<- int) {  // Send-only
    for i := 0; i < 10; i++ {
        ch <- i
    }
    close(ch)
}

func consumer(ch <-chan int) {  // Receive-only
    for value := range ch {
        process(value)
    }
}

// Usage
ch := make(chan int)
go producer(ch)
consumer(ch)
```

### Closing Channels

**Only the sender should close a channel:**

```go
// ✅ Good - Sender closes
func producer(ch chan<- int) {
    defer close(ch)  // Sender closes

    for i := 0; i < 10; i++ {
        ch <- i
    }
}

func consumer(ch <-chan int) {
    for value := range ch {  // Range automatically handles close
        process(value)
    }
}

// ❌ Bad - Never close in receiver
func consumer(ch <-chan int) {
    for value := range ch {
        process(value)
    }
    close(ch)  // DON'T DO THIS - receiver shouldn't close
}
```

## Select Statement

### Basic Select

```go
select {
case value := <-ch1:
    process(value)
case ch2 <- result:
    log.Println("sent result")
case <-time.After(time.Second):
    log.Println("timeout")
default:
    log.Println("no channel ready")
}
```

### Select with Context

Always include context cancellation in select:

```go
// ✅ Good - Select with context
func (s *Service) processLoop(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case data := <-s.dataChan:
            if err := s.process(data); err != nil {
                return err
            }
        case <-s.stopChan:
            return nil
        }
    }
}
```

### Timeout Pattern

```go
func fetchWithTimeout(ctx context.Context, url string) ([]byte, error) {
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    resultChan := make(chan []byte, 1)
    errChan := make(chan error, 1)

    go func() {
        data, err := fetch(url)
        if err != nil {
            errChan <- err
            return
        }
        resultChan <- data
    }()

    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    case err := <-errChan:
        return nil, err
    case data := <-resultChan:
        return data, nil
    }
}
```

## Context Usage

### Context Rules

1. Pass context as **first parameter** named `ctx`
2. **Never store** context in a struct
3. Pass context explicitly through call chain
4. Use context for cancellation, deadlines, and request-scoped values

```go
// ✅ Good - Context as first parameter
func (s *Service) ProcessUser(ctx context.Context, userID string) error {
    user, err := s.repo.FindByID(ctx, userID)
    if err != nil {
        return err
    }
    return s.process(ctx, user)
}

// ❌ Bad - Context stored in struct
type Service struct {
    ctx  context.Context  // DON'T DO THIS
    repo Repository
}
```

### Context Creation

```go
// Background context (top-level)
ctx := context.Background()

// With cancellation
ctx, cancel := context.WithCancel(parentCtx)
defer cancel()

// With timeout
ctx, cancel := context.WithTimeout(parentCtx, 30*time.Second)
defer cancel()

// With deadline
deadline := time.Now().Add(time.Minute)
ctx, cancel := context.WithDeadline(parentCtx, deadline)
defer cancel()

// With value (use sparingly)
ctx = context.WithValue(parentCtx, keyRequestID, "12345")
```

### Context Cancellation

```go
func (s *Service) ProcessItems(ctx context.Context, items []Item) error {
    for _, item := range items {
        // Check for cancellation
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }

        if err := s.processItem(ctx, item); err != nil {
            return err
        }
    }
    return nil
}

// Alternative: Pass ctx to operations that support it
func (s *Service) ProcessItems(ctx context.Context, items []Item) error {
    for _, item := range items {
        // processItem internally checks ctx.Done()
        if err := s.processItem(ctx, item); err != nil {
            if errors.Is(err, context.Canceled) {
                return err
            }
            return err
        }
    }
    return nil
}
```

### Context Values (Use Sparingly)

```go
type contextKey string

const (
    contextKeyRequestID contextKey = "request_id"
    contextKeyUserID    contextKey = "user_id"
)

// Set value
func WithRequestID(ctx context.Context, requestID string) context.Context {
    return context.WithValue(ctx, contextKeyRequestID, requestID)
}

// Get value
func GetRequestID(ctx context.Context) string {
    if requestID, ok := ctx.Value(contextKeyRequestID).(string); ok {
        return requestID
    }
    return ""
}

// Usage
ctx = WithRequestID(ctx, "req-123")
requestID := GetRequestID(ctx)
```

## Common Concurrency Patterns

### Worker Pool

```go
func (s *Service) ProcessWithWorkerPool(ctx context.Context, items []Item) error {
    numWorkers := 10
    itemChan := make(chan Item, len(items))
    errChan := make(chan error, 1)

    var wg sync.WaitGroup

    // Start workers
    for i := 0; i < numWorkers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for item := range itemChan {
                if err := s.processItem(ctx, item); err != nil {
                    select {
                    case errChan <- err:
                    default:
                    }
                    return
                }
            }
        }()
    }

    // Send work
    for _, item := range items {
        select {
        case <-ctx.Done():
            close(itemChan)
            return ctx.Err()
        case itemChan <- item:
        }
    }
    close(itemChan)

    // Wait for completion
    wg.Wait()

    // Check for errors
    select {
    case err := <-errChan:
        return err
    default:
        return nil
    }
}
```

### Fan-Out, Fan-In

```go
func fanOutFanIn(ctx context.Context, items []Item) ([]Result, error) {
    resultChan := make(chan Result, len(items))
    errChan := make(chan error, 1)

    var wg sync.WaitGroup

    // Fan-out: Launch goroutine for each item
    for _, item := range items {
        wg.Add(1)
        go func(i Item) {
            defer wg.Done()

            result, err := process(ctx, i)
            if err != nil {
                select {
                case errChan <- err:
                default:
                }
                return
            }

            select {
            case <-ctx.Done():
            case resultChan <- result:
            }
        }(item)
    }

    // Wait for all goroutines
    go func() {
        wg.Wait()
        close(resultChan)
    }()

    // Fan-in: Collect results
    var results []Result
    for result := range resultChan {
        results = append(results, result)
    }

    // Check for errors
    select {
    case err := <-errChan:
        return nil, err
    default:
        return results, nil
    }
}
```

### Pipeline

```go
func pipeline(ctx context.Context, items []Item) <-chan Result {
    // Stage 1: Generate items
    itemChan := generate(ctx, items)

    // Stage 2: Process items
    processedChan := process(ctx, itemChan)

    // Stage 3: Aggregate results
    resultChan := aggregate(ctx, processedChan)

    return resultChan
}

func generate(ctx context.Context, items []Item) <-chan Item {
    out := make(chan Item)
    go func() {
        defer close(out)
        for _, item := range items {
            select {
            case <-ctx.Done():
                return
            case out <- item:
            }
        }
    }()
    return out
}

func process(ctx context.Context, in <-chan Item) <-chan ProcessedItem {
    out := make(chan ProcessedItem)
    go func() {
        defer close(out)
        for item := range in {
            processed := doProcess(item)
            select {
            case <-ctx.Done():
                return
            case out <- processed:
            }
        }
    }()
    return out
}
```

### Rate Limiting

```go
// Token bucket rate limiter
type RateLimiter struct {
    tokens chan struct{}
}

func NewRateLimiter(requestsPerSecond int) *RateLimiter {
    limiter := &RateLimiter{
        tokens: make(chan struct{}, requestsPerSecond),
    }

    // Refill tokens
    go func() {
        ticker := time.NewTicker(time.Second / time.Duration(requestsPerSecond))
        defer ticker.Stop()

        for range ticker.C {
            select {
            case limiter.tokens <- struct{}{}:
            default:
            }
        }
    }()

    return limiter
}

func (r *RateLimiter) Wait(ctx context.Context) error {
    select {
    case <-ctx.Done():
        return ctx.Err()
    case <-r.tokens:
        return nil
    }
}

// Usage
limiter := NewRateLimiter(10)  // 10 requests per second

for _, item := range items {
    if err := limiter.Wait(ctx); err != nil {
        return err
    }
    process(item)
}
```

### Retry with Backoff

```go
func retryWithBackoff(ctx context.Context, operation func() error) error {
    maxRetries := 5
    baseDelay := time.Second

    for attempt := 0; attempt < maxRetries; attempt++ {
        err := operation()
        if err == nil {
            return nil
        }

        if attempt == maxRetries-1 {
            return fmt.Errorf("max retries exceeded: %w", err)
        }

        delay := baseDelay * time.Duration(1<<uint(attempt))  // Exponential backoff

        log.Printf("attempt %d failed: %v, retrying in %v", attempt+1, err, delay)

        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(delay):
        }
    }

    return nil
}
```

## Synchronization Primitives

### sync.Mutex

```go
type Counter struct {
    mu    sync.Mutex
    count int
}

func (c *Counter) Increment() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.count++
}

func (c *Counter) Value() int {
    c.mu.Lock()
    defer c.mu.Unlock()
    return c.count
}
```

### sync.RWMutex

```go
type Cache struct {
    mu   sync.RWMutex
    data map[string]any
}

func (c *Cache) Get(key string) (any, bool) {
    c.mu.RLock()  // Read lock
    defer c.mu.RUnlock()
    value, ok := c.data[key]
    return value, ok
}

func (c *Cache) Set(key string, value any) {
    c.mu.Lock()  // Write lock
    defer c.mu.Unlock()
    c.data[key] = value
}
```

### sync.WaitGroup

```go
func processAll(items []Item) {
    var wg sync.WaitGroup

    for _, item := range items {
        wg.Add(1)
        go func(i Item) {
            defer wg.Done()
            process(i)
        }(item)
    }

    wg.Wait()  // Wait for all goroutines
}
```

### sync.Once

```go
type Service struct {
    initOnce sync.Once
    client   *http.Client
}

func (s *Service) getClient() *http.Client {
    s.initOnce.Do(func() {
        s.client = &http.Client{
            Timeout: 30 * time.Second,
        }
    })
    return s.client
}
```

### sync.Map

Use for concurrent map access without custom locking:

```go
var cache sync.Map

// Store
cache.Store("key", "value")

// Load
if value, ok := cache.Load("key"); ok {
    log.Println(value)
}

// LoadOrStore
actual, loaded := cache.LoadOrStore("key", "value")

// Delete
cache.Delete("key")

// Range
cache.Range(func(key, value any) bool {
    log.Printf("%v: %v", key, value)
    return true  // Continue iteration
})
```

## Concurrency Anti-Patterns

### ❌ Goroutine Leak

```go
// Bad - Goroutine leaks if channel is never read
func leak() {
    ch := make(chan int)
    go func() {
        ch <- 42  // Blocks forever if no one reads
    }()
}

// Good - Use buffered channel or select with context
func noLeak(ctx context.Context) {
    ch := make(chan int, 1)  // Buffered
    go func() {
        select {
        case ch <- 42:
        case <-ctx.Done():
        }
    }()
}
```

### ❌ Race Condition

```go
// Bad - Race condition
type Counter struct {
    count int
}

func (c *Counter) Increment() {
    c.count++  // Not thread-safe!
}

// Good - Use mutex
type Counter struct {
    mu    sync.Mutex
    count int
}

func (c *Counter) Increment() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.count++
}
```

### ❌ Holding Lock Too Long

```go
// Bad - Lock held during slow operation
func (c *Cache) Get(key string) (any, error) {
    c.mu.Lock()
    defer c.mu.Unlock()

    if value, ok := c.data[key]; ok {
        return value, nil
    }

    // Slow operation while holding lock!
    value, err := c.fetchFromDatabase(key)
    if err != nil {
        return nil, err
    }

    c.data[key] = value
    return value, nil
}

// Good - Minimize lock duration
func (c *Cache) Get(key string) (any, error) {
    c.mu.RLock()
    value, ok := c.data[key]
    c.mu.RUnlock()

    if ok {
        return value, nil
    }

    // Slow operation without lock
    value, err := c.fetchFromDatabase(key)
    if err != nil {
        return nil, err
    }

    c.mu.Lock()
    c.data[key] = value
    c.mu.Unlock()

    return value, nil
}
```

## Testing Concurrent Code

### Testing with Race Detector

```bash
go test -race ./...
```

### Testing Goroutines

```go
func TestConcurrentAccess(t *testing.T) {
    cache := NewCache()
    var wg sync.WaitGroup

    // Launch multiple goroutines
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()

            key := fmt.Sprintf("key-%d", id)
            cache.Set(key, id)

            value, ok := cache.Get(key)
            require.True(t, ok)
            require.Equal(t, id, value)
        }(i)
    }

    wg.Wait()
}
```

### Testing with Context Cancellation

```go
func TestService_ProcessWithCancellation(t *testing.T) {
    service := NewService()
    ctx, cancel := context.WithCancel(context.Background())

    // Start processing in goroutine
    errChan := make(chan error, 1)
    go func() {
        errChan <- service.Process(ctx)
    }()

    // Cancel after short delay
    time.Sleep(100 * time.Millisecond)
    cancel()

    // Verify cancellation
    err := <-errChan
    require.Error(t, err)
    require.True(t, errors.Is(err, context.Canceled))
}
```

## Concurrency Checklist

- [ ] Context passed as first parameter to all functions
- [ ] Context used for cancellation and timeouts
- [ ] All goroutines have a way to exit
- [ ] Channel directions specified in function signatures
- [ ] Only sender closes channels
- [ ] Select statements include context cancellation
- [ ] Proper synchronization (mutex, channels) for shared data
- [ ] No goroutine leaks
- [ ] WaitGroup used to wait for goroutines
- [ ] Lock duration minimized
- [ ] Tests run with `-race` flag
- [ ] Buffered channels sized appropriately
- [ ] Error handling in goroutines
