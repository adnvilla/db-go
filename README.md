# db-go

A lightweight wrapper around GORM that adds connection pooling, read replicas and optional Datadog tracing.

## Features

- Connection pooling with GORM
- Optional read replica support using `gorm.io/plugin/dbresolver`
- Datadog APM integration via `dd-trace-go`
- Helpers for storing a `*gorm.DB` in `context.Context`
- Transaction helper with context propagation

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

