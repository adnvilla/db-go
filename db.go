package dbgo

import (
	"sync"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

type DBConn struct {
	Instance *gorm.DB
	Error    error
}

var (
	conn          DBConn
	dbConnOnce    sync.Once
	GetConnection = getConnection
)

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
			conn.Instance, conn.Error = db, err
			return
		}

		if len(config.ReplicasDSN) == 0 {
			conn.Instance, conn.Error = db, err
			return
		}

		replicas := make([]gorm.Dialector, len(config.ReplicasDSN))
		for i, r := range config.ReplicasDSN {
			replicas[i] = postgres.Open(r)
		}

		dbRresolver := dbresolver.Config{
			// Principal or Write/Source
			Sources: []gorm.Dialector{postgres.Open(config.PrimaryDSN)},
			// Read Replicas
			Replicas: replicas,
			Policy:   dbresolver.RandomPolicy{},
		}

		err = db.Use(dbresolver.Register(dbRresolver))

		conn.Instance, conn.Error = db, err
	})
	return &conn
}

func ResetConnection() {
	dbConnOnce = sync.Once{}
}
