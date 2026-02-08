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

func SetFromContext(ctx context.Context, db *gorm.DB) context.Context {
	return context.WithValue(ctx, dbContextKey, db)
}
