# db-go

A lightweight wrapper for GORM that provides connection pooling, replicas support, and Datadog tracing.

## Features

- Connection pooling with GORM
- Support for read replicas
- Datadog APM integration
- Context propagation for distributed tracing

## Installation

```bash
go get github.com/adnvilla/db-go
```

````markdown
# db-go

A lightweight wrapper for GORM that provides connection pooling, replicas support, and Datadog tracing.

## Features

- Connection pooling with GORM
- Support for read replicas
- Datadog APM integration
- Context propagation for distributed tracing

## Installation

```bash
go get github.com/adnvilla/db-go
```

## Docker Setup

The project includes a Docker Compose setup for local development with PostgreSQL and Datadog agent:

```bash
# Start the containers
docker compose up -d
```

## Docker Commands

### For Linux/Mac (Makefile)

A Makefile is included for convenient Docker operations:

```bash
# Start containers
make up

# Stop containers
make down

# Restart containers
make restart

# View logs
make logs

# Check container status
make ps

# Connect to PostgreSQL shell
make pg-shell

# Run the datadog example
make example
```

### For Windows (Batch File)

A Windows batch file is included for the same operations:

```cmd
# Start containers
db-cmds up

# Stop containers
db-cmds down

# Restart containers
db-cmds restart

# View logs
db-cmds logs

# Check container status
db-cmds ps

# Connect to PostgreSQL shell
db-cmds pg-shell

# Run the datadog example
db-cmds example
```

## Basic Usage

## Datadog Tracing Support

Enable Datadog APM tracing for your database operations:

```go
package main

import (
    "context"
    "github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
    "github.com/adnvilla/db-go"
)

func main() {
    // Start the Datadog tracer
    tracer.Start(
        tracer.WithService("my-service"),
        tracer.WithEnv("production"),
    )
    defer tracer.Stop()
    
    // Configure the database with tracing enabled
    config := dbgo.Config{
        PrimaryDSN: "postgresql://user:password@localhost:5432/mydb?sslmode=disable",
    }
    
    // Enable Datadog tracing with options
    config = *dbgo.WithTracing(&config)
    config = *dbgo.WithTracingServiceName("db-service")(&config)
    config = *dbgo.WithTracingAnalyticsRate(1.0)(&config)
    
    // Get database connection
    dbConn := dbgo.GetConnection(config)
    if dbConn.Error != nil {
        panic(dbConn.Error)
    }
    
    // Create a parent span
    span, ctx := tracer.StartSpanFromContext(context.Background(), "database-operations")
    defer span.Finish()
    
    // Use the context with the span for database operations
    db := dbgo.WithContext(ctx, dbConn.Instance)
    
    // Your database operations will now be traced
    // ...
}
```

```

## License

This project is licensed under the terms of the license included in the repository.
```