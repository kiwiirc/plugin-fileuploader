package db

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"

	"github.com/jmoiron/sqlx"

	"github.com/rs/zerolog"
)

type DBConfig struct {
	DriverName string
	DSN        string
}

type DatabaseConnection struct {
	DB     *sqlx.DB
	DBLock sync.RWMutex
	DBConfig
}

func ConnectToDB(log *zerolog.Logger, dbConfig DBConfig) *DatabaseConnection {
	if !strings.Contains(dbConfig.DSN, "?") {
		// Add the default connection options if none are given
		switch dbConfig.DriverName {
		case "sqlite3":
			dbConfig.DSN += "?_busy_timeout=5000"
		case "mysql":
			dbConfig.DSN += "?parseTime=true"
		}
	}

	db, err := sqlx.Open(dbConfig.DriverName, dbConfig.DSN)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not open database")
	}

	// if dbConfig.DriverName == "sqlite3" {
	// 	db.SetMaxOpenConns(1)
	// }

	// note that we don't do db.SetMaxOpenConns(1), as we don't want to limit
	// read concurrency unnecessarily. sqlite will handle write locking on its
	// own, even across multiple processes accessing the same database file.
	// https://www.sqlite.org/faq.html#q5

	// we also don't enable the write-ahead-log because it does not work over a
	// networked filesystem

	return &DatabaseConnection{
		DB:       db,
		DBConfig: dbConfig,
	}
}

// UpdateRow wraps db.Exec and ensures that exactly one row was affected
func UpdateRow(DBConn *DatabaseConnection, query string, args ...any) (err error) {
	if DBConn.DBConfig.DriverName == "sqlite3" {
		DBConn.DBLock.Lock()
		defer DBConn.DBLock.Unlock()
	}

	res, err := DBConn.DB.Exec(query, args...)
	if err != nil {
		return
	}

	count, err := res.RowsAffected()
	if err != nil {
		return
	}
	if count != 1 {
		err = fmt.Errorf("Expected 1 affected row, got %d", count)
	}
	return
}

func QueryRow(DBConn *DatabaseConnection, query string, args ...any) *sql.Row {
	if DBConn.DBConfig.DriverName == "sqlite3" {
		DBConn.DBLock.RLock()
		defer DBConn.DBLock.RUnlock()
	}

	return DBConn.DB.QueryRow(query, args...)
}

func Select(DBConn *DatabaseConnection, dest any, query string, args ...any) (err error) {
	if DBConn.DBConfig.DriverName == "sqlite3" {
		DBConn.DBLock.RLock()
		defer DBConn.DBLock.RUnlock()
	}

	return DBConn.DB.Select(dest, query, args...)
}
