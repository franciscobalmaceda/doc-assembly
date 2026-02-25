# Go Best Practices for Microservices

Reference guide for Go development, based on official sources and community research (2025/2026).

## Table of Contents

1. [Function Size](#1-function-size)
2. [Architecture](#2-architecture)
3. [Naming Conventions](#3-naming-conventions)
4. [Error Handling](#4-error-handling)
5. [Guard Clauses](#5-guard-clauses)
6. [API Design](#6-api-design)
7. [Concurrency](#7-concurrency)
8. [Anti-patterns](#8-anti-patterns)
9. [Testing](#9-testing)
10. [Modern Go (1.22+)](#10-modern-go-122)
11. [Code Organization](#11-code-organization)
12. [Tools](#12-tools)

---

## 1. Function Size

### Recommendations

| Guideline | Recommended Size |
|-----------|------------------|
| Ideal (Go-specific) | **5-8 lines** |
| Limit before refactoring | **50-70 lines** |
| Clean Code general | **Under 20 lines** |
| Absolute maximum | **Never 100+ lines** |

### Principles

- **One function = one responsibility**
- **Easier testing**: It's easier to test 4 lines than 40
- **Readability over duplication**: Short functions = readable code
- **Separation of intention and implementation**: Name = "what", body = "how"

---

## 2. Architecture

### Hexagonal Architecture (Ports & Adapters)

**Recommended for microservices** - Implemented in doc-engine.

```
+--------------------------------------------------+
|                    Adapters                       |
|  +---------+                      +---------+    |
|  |  HTTP   |<---- Ports --------->|   DB    |    |
|  |  gRPC   |    (interfaces)      |  Cache  |    |
|  +---------+                      +---------+    |
|              +------------------+                 |
|              |   Core Domain   |                 |
|              |   (Entities)    |                 |
|              |   (Use Cases)   |                 |
|              +------------------+                 |
+--------------------------------------------------+
```

**When to apply:**
- Projects with multiple adapters (HTTP, gRPC, CLI)
- High testability required
- Long-lived projects
- Medium/large teams

### Clean Architecture Layers

- **Domain (Entities)**: Pure business rules
- **Use Cases/Services**: Business logic
- **Interfaces**: API, CLI, gRPC
- **Infrastructure**: DB, external APIs, frameworks

---

## 3. Naming Conventions

### Examples

```go
// BAD: Type redundancy
var usersMap map[string]User
func ParseYAMLConfig(input string) (*Config, error)

// GOOD: Descriptive names without redundancy
var users map[string]User
func Parse(input string) (*Config, error)
```

### Rules

| Rule | Description |
|------|-------------|
| Length proportional to scope | `i` for loops, long names for wide scope |
| Descriptive packages | Reflect what they provide, not generic content |
| Avoid | `util`, `helper`, `common`, `misc`, `tools` |
| Consistency | Same name for the same concept |

---

## 4. Error Handling

### Correct Patterns

```go
// Sentinel error values
var ErrNotFound = errors.New("not found")
var ErrDuplicate = errors.New("duplicate entry")

// Wrapping with context
return fmt.Errorf("failed to create user %s: %w", userID, err)

// Verification with errors.Is
if errors.Is(err, ErrNotFound) {
    // handle not found
}
```

### Incorrect Patterns

```go
// BAD: String matching
if err.Error() == "not found" {
    // fragile and error-prone
}

// BAD: Log + return (handles twice)
log.Error("failed", err)
return err
```

### Principles

- Use `%w` internally to preserve error chains
- Use `%v` at system boundaries (RPC, storage)
- Handle errors ONCE (not log + return)
- Specific and actionable messages

---

## 5. Guard Clauses

### Line of Sight Coding

```go
// BAD: Deep nesting
func process(data *Data) error {
    if data != nil {
        if data.Valid {
            if data.Ready {
                // actual logic
            }
        }
    }
    return nil
}

// GOOD: Guard clauses, success path flows down
func process(data *Data) error {
    if data == nil {
        return ErrNilData
    }
    if !data.Valid {
        return ErrInvalidData
    }
    if !data.Ready {
        return ErrNotReady
    }

    // actual logic - success path
    return nil
}
```

---

## 6. API Design

### Clear Parameters

```go
// BAD: Ambiguous parameters of the same type
func CopyFile(dest, source string) error

// GOOD: Types that prevent misuse
type Source struct{ Path string }
func (s Source) CopyTo(dest string) error
```

### Option Structs

```go
// BAD: Too many parameters
func CreateUser(name, email, phone, addr1, addr2, city, country string) error

// GOOD: Option struct
type CreateUserOpts struct {
    Name    string
    Email   string
    Phone   string
    Address Address
}
func CreateUser(opts CreateUserOpts) (*User, error)
```

### Useful Zero Value

```go
// Usable without explicit initialization
type Buffer struct {
    data []byte
}

func (b *Buffer) Write(p []byte) (int, error) {
    b.data = append(b.data, p...)  // works with nil slice
    return len(p), nil
}

var buf Buffer  // usable immediately
buf.Write([]byte("hello"))
```

---

## 7. Concurrency

### Worker Pool

```go
func workerPool(jobs <-chan Job, results chan<- Result, workers int) {
    var wg sync.WaitGroup
    for i := 0; i < workers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for job := range jobs {
                results <- process(job)
            }
        }()
    }
    wg.Wait()
    close(results)
}
```

### Fan-Out/Fan-In

```go
// Fan-out: distribute work
for _, item := range items {
    go func(i Item) {
        resultCh <- process(i)
    }(item)
}

// Fan-in: consolidate results
func merge(channels ...<-chan Result) <-chan Result {
    out := make(chan Result)
    var wg sync.WaitGroup
    for _, ch := range channels {
        wg.Add(1)
        go func(c <-chan Result) {
            defer wg.Done()
            for r := range c {
                out <- r
            }
        }(ch)
    }
    go func() {
        wg.Wait()
        close(out)
    }()
    return out
}
```

### Golden Rules

| Rule | Description |
|------|-------------|
| Know when it ends | Use channels for completion signal |
| Context for cancellation | Every goroutine must be cancellable |
| Close from sender | The sender knows when it's done |
| Limit concurrency | Worker pools to avoid exhaustion |
| Leave to caller | Functions should not start goroutines |

### Concurrency Anti-patterns

```go
// BAD: time.Sleep for synchronization
go doWork()
time.Sleep(time.Second)  // race condition

// GOOD: Explicit synchronization
done := make(chan struct{})
go func() {
    doWork()
    close(done)
}()
<-done

// BAD: Hanging goroutines
go func() {
    for {
        // no way to exit
    }
}()

// GOOD: Cancellation via context
go func(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        default:
            // work
        }
    }
}(ctx)
```

---

## 8. Anti-patterns

### 8.1 Generic Package Names

```go
// AVOID
package util
package common
package helper
package misc
package tools

// PREFER
package auth
package payment
package notification
```

### 8.2 Context Value Pollution

```go
// BAD: Using context to pass data
ctx = context.WithValue(ctx, "userID", userID)
ctx = context.WithValue(ctx, "tenantID", tenantID)

// GOOD: Explicit parameters
func Process(ctx context.Context, userID, tenantID string) error
```

### 8.3 Constructor Doing Too Much

```go
// BAD: Constructor that connects
func NewClient(addr string) (*Client, error) {
    c := &Client{addr: addr}
    return c, c.connect()  // coupling
}

// GOOD: Separate creation from connection
func NewClient(addr string) *Client {
    return &Client{addr: addr}
}
func (c *Client) Connect() error { ... }
```

### 8.4 Excessive Pointers

```go
// BAD: Pointers to types that don't need them
func process(s *string, n *int, m *map[string]int)

// GOOD: Direct values
func process(s string, n int, m map[string]int)
```

### 8.5 Tight Coupling

```go
// BAD: One struct for API, storage, and logic
type User struct {
    ID        int       `json:"id" db:"id"`
    Name      string    `json:"name" db:"name"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// GOOD: Separate models per layer
// entity/user.go
type User struct { ID int; Name string; CreatedAt time.Time }

// dto/user.go
type UserResponse struct { ID int `json:"id"`; Name string `json:"name"` }
```

---

## 9. Testing

### Table-Driven Tests

```go
func TestAdd(t *testing.T) {
    tests := []struct {
        name     string
        a, b     int
        expected int
    }{
        {"positive", 1, 2, 3},
        {"negative", -1, -2, -3},
        {"zero", 0, 0, 0},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := Add(tt.a, tt.b)
            if got != tt.expected {
                t.Errorf("Add(%d, %d) = %d; want %d", tt.a, tt.b, got, tt.expected)
            }
        })
    }
}
```

### Recommended Practices

- Parallel tests with `t.Parallel()`
- Clean mocks via interfaces
- Separate unit tests from integration tests
- Use `t.Helper()` in auxiliary functions
- Test coverage for exported functions

---

## 10. Modern Go (1.22+)

### Loop Variable Fix (Go 1.22)

```go
// Before 1.22: Common bug
for _, v := range values {
    go func() {
        fmt.Println(v)  // all print the last value
    }()
}

// Go 1.22+: Automatically fixed
for _, v := range values {
    go func() {
        fmt.Println(v)  // each goroutine has its own copy
    }()
}
```

### Range over Integers (Go 1.22)

```go
// Before
for i := 0; i < 10; i++ { ... }

// Go 1.22+
for i := range 10 { ... }
```

### Generics Best Practices

```go
// GOOD: Use when it reduces significant duplication
func Map[T, U any](slice []T, fn func(T) U) []U {
    result := make([]U, len(slice))
    for i, v := range slice {
        result[i] = fn(v)
    }
    return result
}

// AVOID: Over-engineering
// Don't use generics when concrete types are sufficient
```

### New Packages (Go 1.21+)

| Package | Purpose |
|---------|---------|
| `log/slog` | Structured logging |
| `slices` | Generic slice operations |
| `maps` | Generic map operations |
| `cmp` | Comparison of ordered values |

---

## 11. Code Organization

### File Structure

```
// BAD: Giant files
service.go  (5000+ lines)

// GOOD: Files by responsibility
user_service.go
user_repository.go
user_handler.go
```

**Suggested limits:**
- Maximum ~500-1000 lines per file
- One `doc.go` for package documentation
- Tests in separate files (`*_test.go`)

### Minimal Main

```go
func main() {
    cfg := config.Load()
    db := database.Connect(cfg.DB)
    defer db.Close()

    app := application.New(db, cfg)
    if err := app.Run(); err != nil {
        log.Fatal(err)
    }
}
```

---

## 12. Tools

| Tool | Purpose |
|------|---------|
| `golangci-lint` | All-in-one linter (100+ linters) |
| `go fmt` / `goimports` | Standard formatting |
| `go vet` | Static analysis |
| `-race` | Race condition detector |
| `wire` | Compile-time DI |
| `goleak` | Goroutine leak detection |

---

## Summary: The 10 Commandments of Go

1. **Small functions** (5-20 lines ideally)
2. **One responsibility per function/type**
3. **Guard clauses over deep nesting**
4. **Explicit error handling, once**
5. **Clear names over comments**
6. **Small and focused interfaces**
7. **Useful zero value**
8. **Controlled concurrency (worker pools, context)**
9. **Layer separation (hexagonal/clean)**
10. **Stdlib first, minimal dependencies**

---

## Sources

- [Google Go Style Best Practices](https://google.github.io/styleguide/go/best-practices.html)
- [Practical Go - Dave Cheney](https://dave.cheney.net/practical-go/presentations/qcon-china.html)
- [Clean Go Article](https://github.com/Pungyeon/clean-go-article)
- [Go Ecosystem 2025 - JetBrains](https://blog.jetbrains.com/go/2025/11/10/go-language-trends-ecosystem-2025/)
- [Common Anti-patterns in Go - DeepSource](https://deepsource.com/blog/common-antipatterns-in-go)
- [Anti-Patterns in Go Web Applications - Three Dots Labs](https://threedots.tech/post/common-anti-patterns-in-go-web-applications/)
- [Go Concurrency Patterns 2025](https://dev.to/aleksei_aleinikov/go-concurrency-2025-goroutines-channels-clean-patterns-3d2c)
- [Architectural Patterns in Go](https://norbix.dev/posts/architectural-patterns/)
- [Error Handling Best Practices - Datadog](https://www.datadoghq.com/blog/go-error-handling/)
