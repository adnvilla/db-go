# db-go

A lightweight helper on top of [GORM](https://gorm.io/) focused on production-friendly PostgreSQL connections. It standardises connection creation, optional read replicas, Datadog tracing and context-aware helpers so that applications can keep database access consistent across services.

## Features at a Glance

- **Singleton connection** – `GetConnection` constructs a single shared `*gorm.DB` (with prepared statement caching) via `sync.Once` and reuses it across the process. Thread-safe with `sync.RWMutex` protection.
- **Read replicas out of the box** – provide one or more replica DSNs and the library configures `gorm.io/plugin/dbresolver` with a random read-balancing policy. Writes are always pinned to the primary.
- **Context helpers** – store/retrieve the current connection from `context.Context`, with automatic fallback to the singleton and error logging when none is available. `MustGetFromContext` panics when no DB is available for layers that assume the context was already initialized.
- **Transaction helper** – `WithTransaction` propagates context, forces writes to the primary, handles commit/rollback with panic recovery, reuses active transactions for nested calls, and logs rollback errors. Repositories and usecases share the same transaction via context without passing `*gorm.DB` through every layer (**transaction-in-context** pattern).
- **Datadog APM integration** – opt-in tracing via `dd-trace-go` with knobs for service name, analytics rate and custom error filtering. Transactions automatically create `"db.transaction"` spans when tracing is enabled.
- **Connection pool tuning** – optional `MaxOpenConns`, `MaxIdleConns`, and `ConnMaxLifetime` in `Config` for production tuning of the underlying `*sql.DB` pool.
- **Health check** – `Ping(ctx)` verifies the connection is alive (e.g. Kubernetes readiness/liveness probes), using the DB from context or the singleton.
- **Active config introspection** – `GetActiveConfig` returns the `Config` used to establish the current connection, enabling runtime introspection.
- **Clean resource management** – `ResetConnection` closes the underlying `*sql.DB` before resetting the singleton, preventing connection leaks.
- **Examples & tooling** – Docker Compose, Make/Batch scripts and runnable examples that demonstrate tracing and clean architecture usage.

## Installation

```bash
go get github.com/adnvilla/db-go
```

## Quick Start

```go
import (
    "context"
    dbgo "github.com/adnvilla/db-go"
)

config := dbgo.Config{
    PrimaryDSN: "postgresql://user:password@localhost:5432/mydb?sslmode=disable",
}

dbConn := dbgo.GetConnection(config)
if dbConn.Error != nil {
    panic(dbConn.Error)
}

ctx := dbgo.SetFromContext(context.Background(), dbConn.Instance)

err := dbgo.WithTransaction(ctx, func(txCtx context.Context) error {
    db := dbgo.GetFromContext(txCtx)
    // use db for operations
    return nil
})
if err != nil {
    panic(err)
}
```

## API Reference

### Connection Management

#### `GetConnection(config Config) *DBConn`

Creates or returns the singleton database connection. Uses `sync.Once` internally — repeated calls reuse the same `*gorm.DB`.

```go
dbConn := dbgo.GetConnection(config)
if dbConn.Error != nil {
    log.Fatal(dbConn.Error)
}
```

#### `ResetConnection()`

Closes the underlying `*sql.DB` connection and resets the singleton, allowing a new connection on the next `GetConnection` call. Useful in tests.

```go
dbgo.ResetConnection()
```

#### `GetActiveConfig() Config`

Returns the `Config` used to establish the current connection. Returns a zero-value `Config` if no connection has been established yet.

```go
cfg := dbgo.GetActiveConfig()
fmt.Println(cfg.PrimaryDSN)
fmt.Println(cfg.EnableTracing)
```

#### `UseDefaultConnection()`

Restores `GetConnection` to the default implementation after it has been overridden (e.g., in tests).

#### `DBConn`

Wraps a GORM database connection and any initialization error.

```go
type DBConn struct {
    Instance *gorm.DB
    Error    error
}
```

### Read Replicas

```go
config := dbgo.Config{
    PrimaryDSN:  "postgresql://.../primary",
    ReplicasDSN: []string{
        "postgresql://.../replica1",
        "postgresql://.../replica2",
    },
}

dbConn := dbgo.GetConnection(config)
```

When replicas are provided, write queries are pinned to the primary while reads are routed randomly through the configured replicas via `dbresolver`.

### Context Helpers

#### `SetFromContext(ctx, db) context.Context`

Stores a `*gorm.DB` in the context for later retrieval.

#### `GetFromContext(ctx) *gorm.DB`

Retrieves the DB from context. Falls back to the singleton connection if none is found. Logs an error and returns `nil` if no connection is available at all.

#### `MustGetFromContext(ctx) *gorm.DB`

Like `GetFromContext`, but panics if no DB is available. Use in layers that assume the context was already initialized with a DB by middleware or a usecase (e.g. repositories called inside `WithTransaction`).

#### `WithContext(ctx, db) (context.Context, *gorm.DB)`

Combines `db.WithContext(ctx)` and `SetFromContext` in a single call. Returns both the enriched context (with the DB stored in it) and the context-aware `*gorm.DB`.

```go
ctx, db := dbgo.WithContext(ctx, dbConn.Instance)
// db has the context set for GORM operations
// ctx has the db stored for retrieval via GetFromContext
```

### Transactions

#### `WithTransaction(ctx, fn UnitOfWork) error`

Executes the given function within a database transaction.

```go
err := dbgo.WithTransaction(ctx, func(txCtx context.Context) error {
    db := dbgo.GetFromContext(txCtx)
    return db.Create(&model).Error
})
```

Behavior:

- **Write routing** – applies `dbresolver.Write` clause to ensure the primary is used.
- **Nested transaction reuse** – if the context already contains an active transaction, it reuses it instead of starting a new one.
- **Nil safety** – returns `dbgo.ErrNoDatabase` if no database connection is available.
- **Panic recovery** – rolls back on panic and re-throws.
- **Rollback logging** – logs rollback errors via `logger.Error` instead of silently discarding them.
- **Auto-tracing** – when Datadog tracing is enabled, automatically creates a `"db.transaction"` span with error tagging on failure.

#### `Ping(ctx) error`

Verifies the database connection is alive using the DB from context (or the default singleton). Intended for health checks (e.g. Kubernetes readiness/liveness). Returns `ErrNoDatabase` when no connection is available, or the error from the underlying `PingContext`.

#### `ErrNoDatabase`

Sentinel error returned by `WithTransaction` and `Ping` when no database connection is available.

```go
if errors.Is(err, dbgo.ErrNoDatabase) {
    // handle missing connection
}
```

#### `UnitOfWork`

Function type for transaction callbacks.

```go
type UnitOfWork func(ctx context.Context) error
```

### Datadog Tracing

Tracing is opt-in. Enable it before passing the `Config` to `GetConnection`:

```go
import (
    "context"
    "github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
    dbgo "github.com/adnvilla/db-go"
)

tracer.Start(tracer.WithService("my-service"))
defer tracer.Stop()

config := dbgo.Config{PrimaryDSN: "postgresql://..."}
config = *dbgo.WithTracing(&config)
config = *dbgo.WithTracingServiceName("db-service")(&config)
config = *dbgo.WithTracingAnalyticsRate(1.0)(&config)
config = *dbgo.WithTracingErrorCheck(func(err error) bool { return err != nil })(&config)

dbConn := dbgo.GetConnection(config)
if dbConn.Error != nil {
    panic(dbConn.Error)
}

// Create a span and use WithContext to propagate it
span, ctx := tracer.StartSpanFromContext(context.Background(), "db-ops")
defer span.Finish()

ctx, db := dbgo.WithContext(ctx, dbConn.Instance)
// All queries through db will appear under the span

// Transactions auto-create a "db.transaction" span when tracing is enabled
err := dbgo.WithTransaction(ctx, func(txCtx context.Context) error {
    db := dbgo.GetFromContext(txCtx)
    return db.Create(&model).Error
})
```

#### Tracing Configuration Functions

| Function | Description |
|----------|-------------|
| `WithTracing(cfg)` | Enables tracing on the config |
| `WithTracingServiceName(name)` | Sets the Datadog service name for spans |
| `WithTracingAnalyticsRate(rate)` | Controls APM analytics sampling (0.0 – 1.0). Uses `*float64` to distinguish unset from zero |
| `WithTracingErrorCheck(fn)` | Custom error filter for span tagging |
| `EnableTracing(db, cfg)` | Applies tracing plugin to a `*gorm.DB` (called internally) |
| `StartSpan(ctx, name, service)` | Convenience helper to create parent spans |

### Configuration

```go
type Config struct {
    PrimaryDSN           string
    ReplicasDSN          []string
    MaxOpenConns         *int              // nil = driver default. Max open connections in the pool.
    MaxIdleConns         *int              // nil = driver default. Max idle connections.
    ConnMaxLifetime      *time.Duration    // nil = driver default. Max time a connection may be reused.
    EnableTracing        bool
    TracingServiceName   string
    TracingAnalyticsRate *float64           // nil = unset, use pointer to distinguish from 0.0
    TracingErrorCheck    func(error) bool
}
```

## Docker Setup

The repository includes a Docker Compose configuration for local development with PostgreSQL and a Datadog agent.

```bash
docker compose up -d
```

### Makefile commands (Linux/Mac)

```bash
make up        # Start containers
make down      # Stop containers
make restart   # Restart containers
make logs      # View container logs
make ps        # Show container status
make pg-shell  # Open a psql shell
make example   # Run the Datadog example
```

### Windows commands

```cmd
db-cmds up
db-cmds down
db-cmds restart
db-cmds logs
db-cmds ps
db-cmds pg-shell
db-cmds example
```

### Examples

- `example/usecase` shows how to wire `GetConnection`, repositories and `WithTransaction` in an application/service layout.
- `example/datadog` demonstrates Datadog tracer configuration, including environment-driven DSNs and analytics.

Run either example with `go run ./example/<name>` once PostgreSQL (and optionally the Datadog agent) is available.

## Testing

```bash
go test -v -count=1 ./...       # Run all tests
go test -race -count=1 ./...    # Run with race detector
go vet ./...                    # Static analysis
```

Use `ResetConnection()` between tests to clear the singleton state.

## License

This project is licensed under the terms of the license included in this repository.
