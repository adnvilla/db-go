# db-go

A lightweight helper on top of [GORM](https://gorm.io/) focused on production-friendly PostgreSQL connections. It standardises connection creation, optional read replicas, Datadog tracing and context-aware helpers so that applications can keep database access consistent across services.

## Features at a Glance

- **Deterministic connection bootstrapping** – `GetConnection` constructs a single shared `*gorm.DB` (with prepared statement caching enabled) and reuses it across the process.
- **Read replicas out of the box** – provide one or more replica DSNs and the library will configure `gorm.io/plugin/dbresolver` with a random read-balancing policy.
- **Context helpers** – store/retrieve the current connection from `context.Context`, log when one is missing, and expose `ResetConnection` for tests.
- **Transaction helper** – `WithTransaction` propagates context, uses write clauses, disables the default GORM transaction nesting and properly commits/rolls back around panics.
- **Datadog APM integration** – opt-in tracing via `dd-trace-go` with knobs for service name, analytics rate and custom error filtering.
- **Examples & tooling** – Docker Compose, Make/Batch scripts and runnable examples that demonstrate tracing and clean architecture usage.

## Installation

```bash
go get github.com/adnvilla/db-go
```

## Basic Usage

```go
import (
    "context"
    dbgo "github.com/adnvilla/db-go"
)

config := dbgo.Config{
    PrimaryDSN: "postgresql://user:password@localhost:5432/mydb?sslmode=disable",
}

// Add replicas if needed
// config.ReplicasDSN = []string{"postgresql://user:password@replica1:5432/mydb?sslmode=disable"}

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

`GetConnection` uses a `sync.Once` guard internally, so repeated calls reuse the same `*gorm.DB`. When writing tests you can call `dbgo.ResetConnection()` to force a new connection on the next `GetConnection` invocation.

### Configuring replicas

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

When replicas are provided, write queries are pinned to the primary while reads are routed randomly through the configured replicas.

## Context helpers

- `SetFromContext(ctx, db)` stores a connection in the context.
- `GetFromContext(ctx)` retrieves it, falling back to the default `GetConnection` result and logging an error if none is available.
- `WithContext(ctx, db)` mirrors `gorm.DB.WithContext` but keeps the helper namespace consistent.

These helpers make it easy to pass the database handle through service layers without creating circular dependencies or implicit globals.

## Transactions

```go
err := dbgo.WithTransaction(ctx, func(txCtx context.Context) error {
    db := dbgo.GetFromContext(txCtx)
    return db.Create(model).Error
})
```

Internally, `WithTransaction`:

1. Reuses the connection from the provided context.
2. Starts a transaction with `SkipDefaultTransaction` and `dbresolver.Write` to ensure primary usage.
3. Commits when the callback returns `nil`, rolls back otherwise, and re-panics to propagate unexpected failures.

## Datadog Tracing

## Datadog Tracing

```go
import (
    "context"
    "github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
    dbgo "github.com/adnvilla/db-go"
)

tracer.Start(tracer.WithService("my-service"))
defer tracer.Stop()

config := dbgo.Config{PrimaryDSN: "postgresql://user:password@localhost:5432/mydb?sslmode=disable"}
config = *dbgo.WithTracing(&config)
config = *dbgo.WithTracingServiceName("db-service")(&config)

// Optional tracing options
config = *dbgo.WithTracingAnalyticsRate(1.0)(&config)
config = *dbgo.WithTracingErrorCheck(func(err error) bool { return err != nil })(&config)

dbConn := dbgo.GetConnection(config)
if dbConn.Error != nil {
    panic(dbConn.Error)
}

span, ctx := tracer.StartSpanFromContext(context.Background(), "db-ops")
defer span.Finish()

db := dbgo.WithContext(ctx, dbConn.Instance)
// ... perform operations with db
```

Tracing is opt-in: call `WithTracing(&config)` before passing the `Config` to `GetConnection`. Additional helpers in `trace.go` include:

- `WithTracingServiceName(name)` – sets the Datadog service used for spans.
- `WithTracingAnalyticsRate(rate)` – controls APM analytics sampling (0.0 – 1.0).
- `WithTracingErrorCheck(func(error) bool)` – custom error filter for span tagging.
- `StartSpan(ctx, name, service)` – convenience helper to create parent spans.

## Docker Setup

The repository includes a Docker Compose configuration for local development with PostgreSQL and a Datadog agent.

```bash
# Start containers
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

### Examples

- `example/usecase` shows how to wire `GetConnection`, repositories and `WithTransaction` in an application/service layout.
- `example/datadog` demonstrates Datadog tracer configuration, including environment-driven DSNs and analytics.

Run either example with `go run ./example/<name>` once PostgreSQL (and optionally the Datadog agent) is available. The `make example` helper spins up Docker Compose (PostgreSQL + Datadog) and executes the tracing example for convenience.

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

## License

This project is licensed under the terms of the license included in this repository.

