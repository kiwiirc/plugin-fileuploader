package shardedfilestore

import (
	"fmt"

	migrate "github.com/rubenv/sql-migrate"
)

func (store *ShardedFileStore) initDB() {
	migrations := &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			{
				Id: "1",
				Up: []string{
					`
					CREATE TABLE uploads(
						id VARCHAR(36) PRIMARY KEY,
						uploader_ip BLOB,
						sha256sum BLOB,
						created_at INTEGER(8)
					);`,
				},
				Down: []string{"DROP TABLE uploads;"},
			},
			{
				Id: "2",
				Up: []string{
					`
					ALTER TABLE uploads
						ADD deleted INTEGER(1) DEFAULT 0 NOT NULL
					;`,
				},
			},
			{
				Id: "3",
				Up: []string{
					`
					CREATE TABLE new_uploads(
						id VARCHAR(36) PRIMARY KEY,
						uploader_ip VARCHAR(45),
						sha256sum BLOB,
						created_at INTEGER(8),
						deleted INTEGER(1) DEFAULT 0 NOT NULL
					);`,
					`
					INSERT INTO new_uploads(id, sha256sum, created_at, deleted)
						SELECT id, sha256sum, created_at, deleted
						FROM uploads
					;`,
					`DROP TABLE uploads;`,
					`ALTER TABLE new_uploads RENAME TO uploads;`,
				},
			},
			{
				Id: "4",
				Up: []string{
					`
					CREATE TABLE new_uploads(
						id VARCHAR(36) PRIMARY KEY,
						uploader_ip VARCHAR(45),
						sha256sum BLOB,
						created_at INTEGER(8),
						deleted INTEGER(1) DEFAULT 0 NOT NULL,
						jwt_account TEXT,
						jwt_issuer TEXT
					);`,
					`
					INSERT INTO new_uploads(id, uploader_ip, sha256sum, created_at, deleted)
						SELECT id, uploader_ip, sha256sum, created_at, deleted
						FROM uploads
					;`,
					`DROP TABLE uploads;`,
					`ALTER TABLE new_uploads RENAME TO uploads;`,
				},
			},
			{
				Id: "5",
				Up: []string{
					`
					CREATE TABLE new_uploads(
						id VARCHAR(36) PRIMARY KEY,
						uploader_ip VARCHAR(45),
						sha256sum BLOB,
						created_at INTEGER(8),
						expires_at INTEGER(8),
						deleted INTEGER(1) DEFAULT 0 NOT NULL,
						jwt_account TEXT DEFAULT '' NOT NULL,
						jwt_issuer TEXT DEFAULT '' NOT NULL
					);`,
					`
					INSERT INTO new_uploads(id, uploader_ip, sha256sum, created_at, deleted, jwt_account, jwt_issuer, expires_at)
						SELECT id, uploader_ip, sha256sum, created_at, deleted,
						CASE WHEN jwt_account IS NULL THEN '' ELSE jwt_account END,
						CASE WHEN jwt_issuer IS NULL THEN '' ELSE jwt_issuer END,
						CASE WHEN jwt_account IS NOT NULL THEN created_at + ` + fmt.Sprintf("%.0f", store.ExpireIdentifiedTime.Seconds()) + `
						ELSE created_at + ` + fmt.Sprintf("%.0f", store.ExpireTime.Seconds()) + ` END
					 	as expires_at
						FROM uploads
					;`,
					`DROP TABLE uploads;`,
					`ALTER TABLE new_uploads RENAME TO uploads;`,
				},
			},
		},
	}

	n, err := migrate.Exec(store.DBConn.DB.DB, store.DBConn.DriverName, migrations, migrate.Up)
	if err != nil {
		store.log.Fatal().Err(err).Msg("Failed to apply migrations")
	}

	if n > 0 {
		store.log.Info().
			Str("event", "schema_migrations").
			Int("count", n).Msg("Applied schema migrations")
	}
}
