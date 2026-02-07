package dbgo

import (
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

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
	origConn := conn
	defer func() {
		conn = origConn
		ResetConnection()
	}()

	// Mark the Once as used
	dbConnOnce.Do(func() {
		conn = DBConn{Instance: &gorm.DB{}, Error: nil}
	})
	assert.NotNil(t, conn.Instance)

	ResetConnection()

	// After reset, sync.Once should allow re-execution
	executed := false
	dbConnOnce.Do(func() {
		executed = true
	})
	assert.True(t, executed, "sync.Once should execute again after ResetConnection")
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
	origConn := conn
	origGetConn := GetConnection
	defer func() {
		conn = origConn
		GetConnection = origGetConn
		ResetConnection()
	}()

	ResetConnection()
	conn = DBConn{}

	callCount := 0
	var mu sync.Mutex
	GetConnection = func(cfg Config) *DBConn {
		dbConnOnce.Do(func() {
			mu.Lock()
			callCount++
			mu.Unlock()
			conn = DBConn{Instance: &gorm.DB{}, Error: nil}
		})
		return &conn
	}

	// Call twice - the inner function should only execute once
	GetConnection(Config{})
	GetConnection(Config{})

	assert.Equal(t, 1, callCount, "sync.Once should only execute the init function once")
}
