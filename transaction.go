package dbgo

import (
	"context"
	"errors"

	logger "github.com/adnvilla/logger-go"
	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

// ErrNoDatabase is returned when no database connection is available.
var ErrNoDatabase = errors.New("dbgo: no database connection available")

// UnitOfWork represents a function that executes within a transaction context.
type UnitOfWork func(ctx context.Context) error

func isTransaction(db *gorm.DB) bool {
	_, ok := db.Statement.ConnPool.(gorm.TxCommitter)
	return ok
}

// WithTransaction executes the given UnitOfWork within a database transaction.
// If the context already contains an active transaction, it reuses it instead of nesting.
// On panic, the transaction is rolled back and the panic is re-thrown.
// When tracing is enabled, a "db.transaction" span is automatically created.
func WithTransaction(ctx context.Context, fn UnitOfWork) (err error) {
	dbInstance := GetFromContext(ctx)
	if dbInstance == nil {
		return ErrNoDatabase
	}

	if isTransaction(dbInstance) {
		return fn(ctx)
	}

	cfg := GetActiveConfig()
	if cfg.EnableTracing {
		var span *tracer.Span
		opts := []tracer.StartSpanOption{}
		if cfg.TracingServiceName != "" {
			opts = append(opts, tracer.ServiceName(cfg.TracingServiceName))
		}
		span, ctx = tracer.StartSpanFromContext(ctx, "db.transaction", opts...)
		defer func() {
			if err != nil {
				span.SetTag("error", true)
				span.SetTag("error.message", err.Error())
			}
			span.Finish()
		}()
	}

	db := dbInstance.
		Session(&gorm.Session{Context: ctx}).
		Clauses(dbresolver.Write).
		Begin()
	if db.Error != nil {
		return db.Error
	}

	defer func() {
		if p := recover(); p != nil {
			if rbErr := db.Rollback().Error; rbErr != nil {
				logger.Error(ctx, "failed to rollback transaction: %v", rbErr)
			}
			panic(p) // re-throw panic
		} else if err != nil {
			if rbErr := db.Rollback().Error; rbErr != nil {
				logger.Error(ctx, "failed to rollback transaction: %v", rbErr)
			}
		} else {
			err = db.Commit().Error
		}
	}()

	err = fn(SetFromContext(ctx, db))
	return err
}
