package db

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/rs/zerolog"
)

type DBConfig struct {
	DriverName string
	DSN        string
}

type DatabaseConnection struct {
	DB *sql.DB
	DBConfig
}

func ConnectToDB(log *zerolog.Logger, dbConfig DBConfig) *DatabaseConnection {
	if !strings.Contains(dbConfig.DSN, "?") {
		// Add the default connection options if none are given
		switch dbConfig.DriverName {
		case "sqlite3":
			dbConfig.DSN += "?_busy_timeout=5000&cache=shared"
		case "mysql":
			dbConfig.DSN += "?parseTime=true"
		}
	}

	db, err := sql.Open(dbConfig.DriverName, dbConfig.DSN)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not open database")
	}

	// note that we don't do db.SetMaxOpenConns(1), as we don't want to limit
	// read concurrency unnecessarily. sqlite will handle write locking on its
	// own, even across multiple processes accessing the same database file.
	// https://www.sqlite.org/faq.html#q5

	// we also don't enable the write-ahead-log because it does not work over a
	// networked filesystem

	return &DatabaseConnection{
		db,
		dbConfig,
	}
}

// UpdateRow wraps db.Exec and ensures that exactly one row was affected
func UpdateRow(db *sql.DB, query string, args ...interface{}) (err error) {
	res, err := db.Exec(query, args...)
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
