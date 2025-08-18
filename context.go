package dbgo

import (
	"context"

	"github.com/adnvilla/logger-go"
	"gorm.io/gorm"
)

type contextKey struct{}

var dbContextKey = contextKey{}

func GetFromContext(ctx context.Context) *gorm.DB {
	if db, ok := ctx.Value(dbContextKey).(*gorm.DB); ok {
		return db
	}

	if conn.Instance != nil {
		return conn.Instance
	}

	logger.Error(ctx, "No GORM DB instance found in context or default connection.")
	return nil
}

func SetFromContext(ctx context.Context, db *gorm.DB) context.Context {
	return context.WithValue(ctx, dbContextKey, db)
}
