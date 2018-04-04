package shardedfilestore

import (
	"fmt"
	"log"

	"github.com/rubenv/sql-migrate"
)

func (store *ShardedFileStore) initDB() {
	migrations := &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			&migrate.Migration{
				Id: "1",
				Up: []string{`CREATE TABLE uploads(
					id TEXT PRIMARY KEY,
					uploader_ip BLOB,
					sha256sum BLOB,
					created_at INTEGER(8)
				);`},
				Down: []string{"DROP TABLE uploads;"},
			},
		},
	}

	n, err := migrate.Exec(store.Db, "sqlite3", migrations, migrate.Up)
	if err != nil {
		log.Fatal(err)
	}

	if n > 0 {
		fmt.Printf("Applied %d schema migrations\n", n)
	}
}
