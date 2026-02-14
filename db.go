package dbgo

import (
	"context"
	"errors"
	"sync"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

// ErrInvalidConfig is returned when Config fails validation (e.g. empty PrimaryDSN).
var ErrInvalidConfig = errors.New("dbgo: invalid config: PrimaryDSN is required")

// DBConn wraps a GORM database connection and any error from initialization.
type DBConn struct {
	Instance *gorm.DB
	Error    error
}

// GetConnection establishes or returns the singleton GORM connection for the given Config.
// It is assigned to a package-level variable so it can be overridden in tests (e.g. with a mock);
// production code should use it as-is. Restore the default with UseDefaultConnection() after tests.
var (
	conn          DBConn
	activeConfig  Config
	dbConnOnce    sync.Once
	connMu        sync.RWMutex
	GetConnection = getConnection
)

// GetActiveConfig returns the Config used to establish the current connection.
// Returns a zero-value Config if no connection has been established yet.
func GetActiveConfig() Config {
	connMu.RLock()
	cfg := activeConfig
	connMu.RUnlock()
	return cfg
}

// UseDefaultConnection restores GetConnection to the default implementation.
func UseDefaultConnection() {
	GetConnection = getConnection
}

func applyPoolConfig(db *gorm.DB, config Config) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if sqlDB == nil {
		return nil
	}
	if config.MaxOpenConns != nil {
		sqlDB.SetMaxOpenConns(*config.MaxOpenConns)
	}
	if config.MaxIdleConns != nil {
		sqlDB.SetMaxIdleConns(*config.MaxIdleConns)
	}
	if config.ConnMaxLifetime != nil {
		sqlDB.SetConnMaxLifetime(*config.ConnMaxLifetime)
	}
	if config.ConnMaxIdleTime != nil {
		sqlDB.SetConnMaxIdleTime(*config.ConnMaxIdleTime)
	}
	return nil
}

func getConnection(config Config) *DBConn {
	if err := config.Validate(); err != nil {
		return &DBConn{Error: err}
	}
	dbConnOnce.Do(func() {
		connMu.Lock()
		activeConfig = config
		connMu.Unlock()

		var err error
		cfg := &gorm.Config{
			PrepareStmt: true,
		}

		// Principal or Write/Source
		db, err := gorm.Open(postgres.Open(config.PrimaryDSN), cfg)
		if err != nil {
			connMu.Lock()
			conn.Instance, conn.Error = db, err
			connMu.Unlock()
			return
		}

		if err := applyPoolConfig(db, config); err != nil {
			connMu.Lock()
			conn.Instance, conn.Error = db, err
			connMu.Unlock()
			return
		}

		if len(config.ReplicasDSN) == 0 {
			// Apply Datadog tracing if enabled
			if config.EnableTracing {
				db, err = EnableTracing(db, config)
				if err != nil {
					// Option B: connection remains usable without tracing; caller gets both Instance and Error.
					connMu.Lock()
					conn.Instance, conn.Error = db, err
					connMu.Unlock()
					return
				}
			}
			connMu.Lock()
			conn.Instance, conn.Error = db, err
			connMu.Unlock()
			return
		}

		replicas := make([]gorm.Dialector, len(config.ReplicasDSN))
		for i, r := range config.ReplicasDSN {
			replicas[i] = postgres.Open(r)
		}

		dbResolver := dbresolver.Config{
			// Read Replicas
			Replicas: replicas,
			Policy:   dbresolver.RandomPolicy{},
		}

		err = db.Use(dbresolver.Register(dbResolver))
		if err != nil {
			connMu.Lock()
			conn.Instance, conn.Error = db, err
			connMu.Unlock()
			return
		}

		// Apply Datadog tracing if enabled
		if config.EnableTracing {
			db, err = EnableTracing(db, config)
			if err != nil {
				// Option B: connection remains usable without tracing; caller gets both Instance and Error.
				connMu.Lock()
				conn.Instance, conn.Error = db, err
				connMu.Unlock()
				return
			}
		}

		connMu.Lock()
		conn.Instance, conn.Error = db, err
		connMu.Unlock()
	})
	connMu.RLock()
	result := conn
	connMu.RUnlock()
	return &result
}

// Ping verifies that the database connection is alive, using the DB from ctx (or the default singleton).
// It is intended for health checks (e.g. Kubernetes readiness/liveness probes).
// Returns ErrNoDatabase when no connection is available, or the error from the underlying PingContext.
func Ping(ctx context.Context) error {
	db := GetFromContext(ctx)
	if db == nil {
		return ErrNoDatabase
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if sqlDB == nil {
		return ErrNoDatabase
	}
	return sqlDB.PingContext(ctx)
}

// ResetConnection closes the underlying database connection and resets the singleton,
// allowing a new connection to be established on the next call to GetConnection.
func ResetConnection() {
	connMu.Lock()
	defer connMu.Unlock()
	if conn.Instance != nil {
		func() {
			defer func() { recover() }()
			if sqlDB, err := conn.Instance.DB(); err == nil && sqlDB != nil {
				sqlDB.Close()
			}
		}()
	}
	conn = DBConn{}
	activeConfig = Config{}
	dbConnOnce = sync.Once{}
}
