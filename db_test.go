package dbgo

import (
	"errors"
	"sync"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func saveAndRestoreConn(t *testing.T) {
	t.Helper()
	connMu.RLock()
	origConn := conn
	connMu.RUnlock()
	t.Cleanup(func() {
		connMu.Lock()
		conn = origConn
		connMu.Unlock()
		ResetConnection()
	})
}

func TestGetConnection_MockReturnsDBConn(t *testing.T) {
	origGetConn := GetConnection
	defer func() { GetConnection = origGetConn }()

	expectedDB := &gorm.DB{}
	GetConnection = func(cfg Config) *DBConn {
		return &DBConn{Instance: expectedDB, Error: nil}
	}

	result := GetConnection(Config{})
	assert.NotNil(t, result)
	assert.Equal(t, expectedDB, result.Instance)
	assert.NoError(t, result.Error)
}

func TestGetConnection_MockReturnsError(t *testing.T) {
	origGetConn := GetConnection
	defer func() { GetConnection = origGetConn }()

	expectedErr := errors.New("connection failed")
	GetConnection = func(cfg Config) *DBConn {
		return &DBConn{Instance: nil, Error: expectedErr}
	}

	result := GetConnection(Config{})
	assert.NotNil(t, result)
	assert.Nil(t, result.Instance)
	assert.ErrorIs(t, result.Error, expectedErr)
}

func TestResetConnection_ClearsSyncOnce(t *testing.T) {
	saveAndRestoreConn(t)

	// Mark the Once as used
	dbConnOnce.Do(func() {
		connMu.Lock()
		conn = DBConn{Instance: &gorm.DB{}, Error: nil}
		connMu.Unlock()
	})
	connMu.RLock()
	assert.NotNil(t, conn.Instance)
	connMu.RUnlock()

	ResetConnection()

	// After reset, sync.Once should allow re-execution
	executed := false
	dbConnOnce.Do(func() {
		executed = true
	})
	assert.True(t, executed, "sync.Once should execute again after ResetConnection")
}

func TestResetConnection_ClosesUnderlyingDB(t *testing.T) {
	saveAndRestoreConn(t)

	mockDB, mock, err := sqlmock.New()
	assert.NoError(t, err)

	db, err := gorm.Open(postgres.New(postgres.Config{
		Conn: mockDB,
	}), &gorm.Config{})
	assert.NoError(t, err)

	connMu.Lock()
	conn = DBConn{Instance: db, Error: nil}
	connMu.Unlock()

	mock.ExpectClose()
	ResetConnection()

	assert.NoError(t, mock.ExpectationsWereMet())

	connMu.RLock()
	assert.Nil(t, conn.Instance)
	assert.NoError(t, conn.Error)
	connMu.RUnlock()
}

func TestUseDefaultConnection_RestoresDefault(t *testing.T) {
	origGetConn := GetConnection
	defer func() { GetConnection = origGetConn }()

	// Override GetConnection
	GetConnection = func(cfg Config) *DBConn {
		return &DBConn{Instance: nil, Error: errors.New("mock")}
	}

	// Restore default
	UseDefaultConnection()

	// GetConnection should now point to the original getConnection
	// We can't easily call it without a real DB, but we can verify the function was reassigned
	// by checking it's no longer our mock
	result := GetConnection(Config{PrimaryDSN: "invalid"})
	// The real getConnection will fail with a connection error, not our mock error
	if result.Error != nil {
		assert.NotEqual(t, "mock", result.Error.Error())
	}

	// Cleanup: reset the singleton that getConnection may have set
	ResetConnection()
}

func TestDBConn_Struct(t *testing.T) {
	t.Run("with nil values", func(t *testing.T) {
		dc := DBConn{}
		assert.Nil(t, dc.Instance)
		assert.NoError(t, dc.Error)
	})

	t.Run("with values", func(t *testing.T) {
		db := &gorm.DB{}
		err := errors.New("test error")
		dc := DBConn{Instance: db, Error: err}
		assert.Equal(t, db, dc.Instance)
		assert.Equal(t, err, dc.Error)
	})
}

func TestGetConnection_Singleton(t *testing.T) {
	saveAndRestoreConn(t)
	origGetConn := GetConnection
	defer func() { GetConnection = origGetConn }()

	ResetConnection()
	connMu.Lock()
	conn = DBConn{}
	connMu.Unlock()

	callCount := 0
	var mu sync.Mutex
	GetConnection = func(cfg Config) *DBConn {
		dbConnOnce.Do(func() {
			mu.Lock()
			callCount++
			mu.Unlock()
			connMu.Lock()
			conn = DBConn{Instance: &gorm.DB{}, Error: nil}
			connMu.Unlock()
		})
		connMu.RLock()
		result := conn
		connMu.RUnlock()
		return &result
	}

	// Call twice - the inner function should only execute once
	GetConnection(Config{})
	GetConnection(Config{})

	assert.Equal(t, 1, callCount, "sync.Once should only execute the init function once")
}
