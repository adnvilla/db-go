package dbgo

import (
	"sync"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

// DBConn wraps a GORM database connection and any error from initialization.
type DBConn struct {
	Instance *gorm.DB
	Error    error
}

var (
	conn          DBConn
	dbConnOnce    sync.Once
	connMu        sync.RWMutex
	GetConnection = getConnection
)

// UseDefaultConnection restores GetConnection to the default implementation.
func UseDefaultConnection() {
	GetConnection = getConnection
}

func getConnection(config Config) *DBConn {
	dbConnOnce.Do(func() {
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

		if len(config.ReplicasDSN) == 0 {
			// Apply Datadog tracing if enabled
			if config.EnableTracing {
				db, err = EnableTracing(db, config)
				if err != nil {
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

		dbRresolver := dbresolver.Config{
			// Read Replicas
			Replicas: replicas,
			Policy:   dbresolver.RandomPolicy{},
		}

		err = db.Use(dbresolver.Register(dbRresolver))
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
	dbConnOnce = sync.Once{}
}
