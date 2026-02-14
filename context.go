package dbgo

import (
	"context"

	"github.com/adnvilla/logger-go"
	"gorm.io/gorm"
)

type contextKey struct{}

var dbContextKey = contextKey{}

// GetFromContext returns the *gorm.DB from ctx, or the default singleton if not set.
// It can return nil when neither the context nor the default connection has a DB (e.g. before Init or after ResetConnection).
// Callers must check for nil before use; see WithTransaction for the recommended pattern:
//
//	dbInstance := dbgo.GetFromContext(ctx)
//	if dbInstance == nil {
//	    return dbgo.ErrNoDatabase
//	}
func GetFromContext(ctx context.Context) *gorm.DB {
	if db, ok := ctx.Value(dbContextKey).(*gorm.DB); ok {
		return db
	}

	connMu.RLock()
	instance := conn.Instance
	connMu.RUnlock()
	if instance != nil {
		return instance
	}

	logger.Warn(ctx, "No GORM DB instance found in context or default connection.")
	return nil
}

// MustGetFromContext returns the *gorm.DB from ctx, or the default singleton if not set.
// It panics if neither the context nor the default connection has a DB (e.g. before Init or after ResetConnection).
// Use this in layers that assume the context was already initialized with a DB by middleware or a usecase.
func MustGetFromContext(ctx context.Context) *gorm.DB {
	db := GetFromContext(ctx)
	if db == nil {
		panic("dbgo: no database connection available in context or default connection")
	}
	return db
}

func SetFromContext(ctx context.Context, db *gorm.DB) context.Context {
	return context.WithValue(ctx, dbContextKey, db)
}
