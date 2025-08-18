package dbgo

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

type UnitOfWork func(ctx context.Context) error

func WithTransaction(ctx context.Context, fn UnitOfWork) (err error) {
	// https://gorm.io/docs/transactions.html#Disable-Default-Transaction
	db := GetFromContext(ctx).
		Session(&gorm.Session{Context: ctx, SkipDefaultTransaction: true}).
		Clauses(dbresolver.Write).
		Begin()
	if db.Error != nil {
		return db.Error
	}

	defer func() {
		if p := recover(); p != nil {
			db.Rollback()
			panic(p) // re-throw panic
		} else if err != nil {
			db.Rollback()
		} else {
			err = db.Commit().Error
		}
	}()

	err = fn(SetFromContext(ctx, db))
	return err
}
