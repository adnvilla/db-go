# Go Conventions for db-go

## Code Style
- Use `gofmt` formatting (no custom style overrides)
- Keep functions short and focused; this is a ~285 LOC library
- Use early returns to reduce nesting
- Exported functions must have Go doc comments

## Naming
- Package name: `dbgo` (no underscores, no hyphens)
- Exported: PascalCase (`GetConnection`, `WithTracing`)
- Unexported package vars: camelCase (`dbConnOnce`, `conn`)
- Context keys: unexported typed struct (`type contextKey struct{}`)

## Patterns
- Functional options return `func(*Config) *Config`
- Singleton connections use `sync.Once`; reset with `ResetConnection()`
- Wrap `*gorm.DB` + `error` in `DBConn` struct
- Propagate DB through `context.Context` using typed keys
- Transactions use defer for rollback/commit with panic recovery

## Dependencies
- Do NOT add new direct dependencies without justification
- Prefer stdlib where possible
- All GORM plugins go through `db.Use()`

## Error Handling
- Return errors, don't panic (except re-throwing recovered panics)
- Use GORM's `db.Error` pattern for DB operations
- Log with `github.com/adnvilla/logger-go`, not `fmt` or `log`
