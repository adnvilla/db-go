# CLAUDE.md - Project Instructions for Claude Code

## Project Overview

**db-go** is a Go library that wraps GORM for PostgreSQL with singleton connection management, read replica support, context propagation, transaction safety, and optional Datadog APM tracing.

- **Module**: `github.com/adnvilla/db-go`
- **Package**: `dbgo`
- **Go version**: 1.24.0
- **Current version**: 1.0.1

## Architecture

The library is ~285 LOC organized in 5 files at the root:

| File | Responsibility |
|------|---------------|
| `config.go` | `Config` struct with DSN, replica, and tracing fields |
| `db.go` | Singleton `*gorm.DB` via `sync.Once`, replica setup with `dbresolver` |
| `context.go` | `GetFromContext`/`SetFromContext` helpers using typed context key |
| `transaction.go` | `WithTransaction` with panic recovery, rollback, and write clause |
| `trace.go` | Datadog tracing integration via functional options pattern |

## Build & Development Commands

```bash
make up          # Start Docker Compose (PostgreSQL + Datadog agent)
make down        # Stop containers
make restart     # Restart containers
make logs        # View container logs
make ps          # Container status
make pg-shell    # Open psql shell
make example     # Run example/datadog
```

## Code Conventions

### Style
- Standard `gofmt` formatting
- PascalCase for exported symbols (`GetConnection`, `WithTransaction`)
- Lowercase for unexported package-level vars (`dbContextKey`, `dbConnOnce`)
- Typed context keys to avoid collisions (`type contextKey struct{}`)

### Patterns Used
- **Singleton**: `sync.Once` for connection initialization
- **Functional options**: `WithTracingServiceName(name) func(*Config) *Config`
- **Result wrapper**: `DBConn{Instance, Error}` pairs connection + error
- **Context propagation**: Store/retrieve `*gorm.DB` from `context.Context`
- **Defer-based safety**: Transaction rollback on panic with re-throw

### Error Handling
- Early return pattern
- Errors propagated via return values
- GORM's `db.Error` field pattern

## Dependencies (direct)

- `gorm.io/gorm` - ORM
- `gorm.io/driver/postgres` - PostgreSQL driver
- `gorm.io/plugin/dbresolver` - Read replica support
- `github.com/DataDog/dd-trace-go/v2` - Datadog tracing
- `github.com/adnvilla/logger-go` - Logging
- `github.com/joho/godotenv` - .env files (used in examples)

## Git & Release Workflow

- **Main branch**: `master`
- **Commit style**: [Conventional Commits](https://www.conventionalcommits.org/)
  - `feat:` -> minor release
  - `fix:`, `perf:`, `refactor:` -> patch release
  - `docs:`, `chore:`, `style:`, `test:` -> no release
  - `BREAKING CHANGE` -> major release
- **Automated release**: semantic-release via GitHub Actions
- **CI**: Reusable workflow from `adnvilla/gha-toolkit` (Go build + test)

## Testing

- No unit tests yet - validation is via examples and Docker integration
- When adding tests, use `*_test.go` files in the root package
- Use `ResetConnection()` between tests to clear the singleton

## Important Notes

- The connection is a **singleton** (`sync.Once`). Call `ResetConnection()` to reset it.
- `GetFromContext` falls back to the default singleton if context has no DB.
- `WithTransaction` forces writes to the primary via `dbresolver.Write` clause.
- Never commit `.env` files - they contain credentials.
- Examples live in `example/` and are not part of the library package.
