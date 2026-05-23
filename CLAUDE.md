# CLAUDE.md - Project Instructions for Claude Code

## Project Overview

**db-go** is a Go library that wraps GORM for PostgreSQL with singleton connection management, read replica support, connection pool tuning, context propagation, transaction safety, and optional Datadog APM tracing.

- **Module**: `github.com/adnvilla/db-go`
- **Package**: `dbgo`
- **Go version**: 1.24.0
- **Current version**: 2.2.0

## Architecture

The library is ~509 LOC (source only) organized in 5 files at the root, plus corresponding `*_test.go` files:

| File | Responsibility |
|------|---------------|
| `config.go` | `Config` struct with DSN, pool, and tracing fields; `Validate()` method |
| `db.go` | Singleton `*gorm.DB` via `sync.Once`; `GetConnection` variable; `GetActiveConfig`, `UseDefaultConnection`, `Ping`, `ResetConnection` |
| `context.go` | `GetFromContext`, `MustGetFromContext`, `SetFromContext` using typed context key |
| `transaction.go` | `WithTransaction` with nested TX detection, Datadog span creation, panic recovery, and `dbresolver.Write` clause; `ErrNoDatabase` |
| `trace.go` | Datadog tracing: `EnableTracing`, `WithTracing`, `WithTracingServiceName`, `WithTracingAnalyticsRate`, `WithTracingErrorCheck`, `WithContext`, `StartSpan`; constants `SpanNameTransaction`, `DefaultTracingServiceName` |

## Public API

### Config

```go
type Config struct {
    PrimaryDSN           string
    ReplicasDSN          []string
    MaxOpenConns         *int
    MaxIdleConns         *int
    ConnMaxLifetime      *time.Duration
    ConnMaxIdleTime      *time.Duration
    EnableTracing        bool
    TracingServiceName   string
    TracingAnalyticsRate *float64           // pointer — nil uses tracer default
    TracingErrorCheck    func(error) bool
}
func (c Config) Validate() error            // returns ErrInvalidConfig if PrimaryDSN is empty
```

### Connection management (db.go)

```go
var GetConnection = func(config Config) *DBConn  // overridable in tests

type DBConn struct {
    Instance *gorm.DB
    Error    error
}

func GetActiveConfig() Config        // returns the Config used to open the current connection
func UseDefaultConnection()          // restores GetConnection to the real implementation
func Ping(ctx context.Context) error // health check; uses DB from ctx or singleton
func ResetConnection()               // closes DB, resets singleton — required between tests

var ErrInvalidConfig = errors.New("dbgo: invalid config: PrimaryDSN is required")
```

### Context helpers (context.go)

```go
func GetFromContext(ctx context.Context) *gorm.DB      // returns nil + warns when not found
func MustGetFromContext(ctx context.Context) *gorm.DB  // panics when not found
func SetFromContext(ctx context.Context, db *gorm.DB) context.Context
```

### Transactions (transaction.go)

```go
type UnitOfWork func(ctx context.Context) error

func WithTransaction(ctx context.Context, fn UnitOfWork) error
// - Detects active TX (via ConnPool type assertion) and reuses it instead of nesting
// - Forces writes to primary via dbresolver.Write clause
// - Creates a "db.transaction" Datadog span when tracing is enabled
// - Rolls back on error or panic; re-throws panics after rollback

var ErrNoDatabase = errors.New("dbgo: no database connection available")
```

### Tracing helpers (trace.go)

```go
const SpanNameTransaction      = "db.transaction"
const DefaultTracingServiceName = "db-go"

func WithTracing(cfg *Config) *Config                                   // sets EnableTracing = true
func WithTracingServiceName(name string) func(*Config) *Config          // functional option
func WithTracingAnalyticsRate(rate float64) func(*Config) *Config       // functional option
func WithTracingErrorCheck(fn func(error) bool) func(*Config) *Config   // functional option

func EnableTracing(db *gorm.DB, cfg Config) (*gorm.DB, error)  // internal; called by getConnection
func WithContext(ctx context.Context, db *gorm.DB) (context.Context, *gorm.DB)  // combines db.WithContext + SetFromContext
func StartSpan(ctx context.Context, name, service string) (context.Context, *tracer.Span)
```

## Build & Development Commands

```bash
make build          # go mod tidy + go build ./...
make test           # go test ./... -count=1
make test-v         # go test ./... -v -count=1
make vet            # go vet ./...
make fmt            # check formatting (exits non-zero if unformatted)
make fmt-fix        # apply gofmt
make lint           # vet + fmt check combined

make up             # docker compose up -d (PostgreSQL + Datadog agent)
make down           # docker compose down
make restart        # down + up
make logs           # docker compose logs -f
make ps             # docker compose ps
make pg-shell       # psql into postgres container

make example        # run example/datadog
make example-usecase # run example/usecase
```

## Code Conventions

### Style
- Standard `gofmt` formatting — CI enforces this
- PascalCase for exported symbols (`GetConnection`, `WithTransaction`)
- camelCase for unexported package-level vars (`dbContextKey`, `dbConnOnce`)
- Typed context keys to avoid collisions (`type contextKey struct{}`)
- Go doc comments required on all exported symbols

### Patterns
- **Singleton**: `sync.Once` (`dbConnOnce`) guarded by `sync.RWMutex` (`connMu`) for race safety
- **Overridable function variable**: `GetConnection` is a `var` so tests can replace it with a mock; restore with `UseDefaultConnection()`
- **Functional options**: tracing config helpers return `func(*Config) *Config`
- **Result wrapper**: `DBConn{Instance, Error}` pairs connection + error
- **Context propagation**: store/retrieve `*gorm.DB` via typed context key
- **Defer-based safety**: transaction rollback on panic with re-throw

### Error Handling
- Return errors, never panic (except re-throwing recovered panics in `WithTransaction`)
- Use `ErrInvalidConfig` and `ErrNoDatabase` sentinel errors; check with `errors.Is`
- Log with `github.com/adnvilla/logger-go`, not `fmt` or `log`
- When tracing is enabled, `WithTransaction` tags the span with `error=true` and `error.message`

## Dependencies (direct)

| Package | Purpose |
|---------|---------|
| `gorm.io/gorm` | ORM |
| `gorm.io/driver/postgres` | PostgreSQL driver |
| `gorm.io/plugin/dbresolver` | Read replica routing |
| `github.com/DataDog/dd-trace-go/v2` | Datadog APM tracer |
| `github.com/DataDog/dd-trace-go/contrib/gorm.io/gorm.v1/v2` | GORM tracing plugin |
| `github.com/adnvilla/logger-go` | Structured logging |
| `github.com/joho/godotenv` | `.env` loading (examples only) |
| `github.com/stretchr/testify` | Test assertions |
| `github.com/DATA-DOG/go-sqlmock` | SQL mock for unit tests |

## Testing

Unit tests live in `*_test.go` files in the root package (`package dbgo`), giving access to unexported state.

**Key testing patterns:**
- `saveAndRestoreConn(t)` — helper that snapshots `conn` and registers `t.Cleanup` to restore it; always call this in tests that touch global state
- `newMockDB(t)` — creates a `*gorm.DB` backed by `go-sqlmock`; registers cleanup
- Override `GetConnection` with a test double for isolation; restore with `defer func() { GetConnection = origGetConn }()`
- Access `connMu`, `conn`, `activeConfig`, `dbConnOnce` directly in tests (same package)
- Call `ResetConnection()` between tests that exercise `getConnection` end-to-end

Run tests: `make test` or `go test ./... -count=1`

## Git & Release Workflow

- **Main branch**: `master`
- **Commit style**: [Conventional Commits](https://www.conventionalcommits.org/)
  - `feat:` → minor release
  - `fix:`, `perf:`, `refactor:` → patch release
  - `docs:`, `chore:`, `style:`, `test:` → no release
  - `BREAKING CHANGE:` in commit body → major release
- **Automated release**: semantic-release (`.releaserc.json`) via GitHub Actions, triggered after the Go CI workflow succeeds on `master`
- **CI**: Reusable workflow from `adnvilla/gha-toolkit@v1.1.1` — runs `go build` and `go test`
- Do **not** manually edit `CHANGELOG.md` — it is managed by semantic-release

## Important Notes

- The connection is a **singleton** (`sync.Once`). A given process can only call `getConnection` successfully once per lifecycle; subsequent calls return the cached `conn`. Call `ResetConnection()` to allow re-initialization.
- `GetFromContext` falls back to the singleton if the context carries no DB, and logs a warning when neither is available. `MustGetFromContext` panics instead of returning nil.
- `WithTransaction` detects an existing transaction by type-asserting `db.Statement.ConnPool` against `gorm.TxCommitter`. Nested calls reuse the outer TX.
- Pool settings (`MaxOpenConns`, `MaxIdleConns`, `ConnMaxLifetime`, `ConnMaxIdleTime`) are applied via the underlying `*sql.DB` after the GORM connection opens.
- When tracing setup fails, the connection is still returned as usable (`DBConn.Instance` is set) alongside the tracing error.
- Never commit `.env` files — they contain credentials.
- Examples in `example/` are standalone programs (`package main`) and are not part of the library.
