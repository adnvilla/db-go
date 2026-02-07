package dbgo

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func newMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	assert.NoError(t, err)

	db, err := gorm.Open(postgres.New(postgres.Config{
		Conn: mockDB,
	}), &gorm.Config{})
	assert.NoError(t, err)

	t.Cleanup(func() {
		mockDB.Close()
	})

	return db, mock
}

func TestUnitOfWork_Type(t *testing.T) {
	var fn UnitOfWork = func(ctx context.Context) error {
		return nil
	}
	assert.NotNil(t, fn)
	assert.NoError(t, fn(context.Background()))
}

func TestWithTransaction_Success(t *testing.T) {
	saveAndRestoreConn(t)

	db, mock := newMockDB(t)
	connMu.Lock()
	conn = DBConn{Instance: db}
	connMu.Unlock()

	mock.ExpectBegin()
	mock.ExpectCommit()

	ctx := context.Background()
	err := WithTransaction(ctx, func(ctx context.Context) error {
		return nil
	})

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWithTransaction_FnReturnsError(t *testing.T) {
	saveAndRestoreConn(t)

	db, mock := newMockDB(t)
	connMu.Lock()
	conn = DBConn{Instance: db}
	connMu.Unlock()

	mock.ExpectBegin()
	mock.ExpectRollback()

	ctx := context.Background()
	fnErr := errors.New("business logic error")
	err := WithTransaction(ctx, func(ctx context.Context) error {
		return fnErr
	})

	assert.ErrorIs(t, err, fnErr)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWithTransaction_Panic(t *testing.T) {
	saveAndRestoreConn(t)

	db, mock := newMockDB(t)
	connMu.Lock()
	conn = DBConn{Instance: db}
	connMu.Unlock()

	mock.ExpectBegin()
	mock.ExpectRollback()

	ctx := context.Background()
	assert.PanicsWithValue(t, "something went wrong", func() {
		_ = WithTransaction(ctx, func(ctx context.Context) error {
			panic("something went wrong")
		})
	})

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWithTransaction_NilDB_ReturnsError(t *testing.T) {
	saveAndRestoreConn(t)

	connMu.Lock()
	conn = DBConn{}
	connMu.Unlock()

	ctx := context.Background()
	err := WithTransaction(ctx, func(ctx context.Context) error {
		return nil
	})

	assert.ErrorIs(t, err, ErrNoDatabase)
}

func TestWithTransaction_NestedReusesTransaction(t *testing.T) {
	saveAndRestoreConn(t)

	db, mock := newMockDB(t)
	connMu.Lock()
	conn = DBConn{Instance: db}
	connMu.Unlock()

	// Only one BEGIN/COMMIT pair — the nested call should reuse the TX
	mock.ExpectBegin()
	mock.ExpectCommit()

	ctx := context.Background()
	err := WithTransaction(ctx, func(ctx context.Context) error {
		// This inner call should detect the active TX and not begin a new one
		return WithTransaction(ctx, func(ctx context.Context) error {
			return nil
		})
	})

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWithTransaction_NestedPropagatesError(t *testing.T) {
	saveAndRestoreConn(t)

	db, mock := newMockDB(t)
	connMu.Lock()
	conn = DBConn{Instance: db}
	connMu.Unlock()

	mock.ExpectBegin()
	mock.ExpectRollback()

	innerErr := errors.New("inner error")
	ctx := context.Background()
	err := WithTransaction(ctx, func(ctx context.Context) error {
		return WithTransaction(ctx, func(ctx context.Context) error {
			return innerErr
		})
	})

	assert.ErrorIs(t, err, innerErr)
	assert.NoError(t, mock.ExpectationsWereMet())
}
