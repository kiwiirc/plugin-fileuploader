package expirer

import (
	"time"

	"github.com/kiwiirc/plugin-fileuploader/shardedfilestore"
	"github.com/rs/zerolog"
)

type Expirer struct {
	ticker             *time.Ticker
	store              *shardedfilestore.ShardedFileStore
	maxAge             time.Duration
	identifiedMaxAge   time.Duration
	deletedMaxAge      time.Duration
	jwtSecretsByIssuer map[string]string
	quitChan           chan struct{} // closes when ticker has been stopped
	log                *zerolog.Logger
}

func New(store *shardedfilestore.ShardedFileStore, maxAge, identifiedMaxAge, deletedMaxAge, checkInterval time.Duration, jwtSecretsByIssuer map[string]string, log *zerolog.Logger) *Expirer {
	expirer := &Expirer{
		ticker:             time.NewTicker(checkInterval),
		store:              store,
		maxAge:             maxAge,
		identifiedMaxAge:   identifiedMaxAge,
		deletedMaxAge:      deletedMaxAge,
		jwtSecretsByIssuer: jwtSecretsByIssuer,
		quitChan:           make(chan struct{}),
		log:                log,
	}

	go func() {
		for {
			select {

			// tick
			case t := <-expirer.ticker.C:
				expirer.gc(t)

			// ticker stopped, exit the goroutine
			case _, ok := <-expirer.quitChan:
				if !ok {
					return
				}

			}
		}
	}()

	return expirer
}

// Stop turns off an Expirer. No more Filestore garbage collection cycles will start.
func (expirer *Expirer) Stop() {
	expirer.ticker.Stop()
	close(expirer.quitChan)
}

func (expirer *Expirer) gc(t time.Time) {
	expirer.log.Debug().
		Str("event", "gc_tick").
		Msg("Filestore GC tick")

	expiredIds, err := expirer.getExpired()
	if err != nil {
		expirer.log.Error().
			Err(err).
			Msg("Failed to enumerate expired uploads")
		return
	}

	for _, id := range expiredIds {
		err = expirer.store.Terminate(id)
		if err != nil {
			expirer.log.Error().
				Err(err).
				Msg("Failed to terminate expired upload")
			continue
		}
		expirer.log.Info().
			Str("event", "expired").
			Str("id", id).
			Msg("Terminated upload id")
	}

	err = expirer.deleteExpired()
	if err != nil {
		expirer.log.Error().
			Err(err).
			Msg("Failed to purge deleted uploads from database")
	}
}

func (expirer *Expirer) getExpired() (expiredIds []string, err error) {
	switch expirer.store.DBConn.DBConfig.DriverName {
	case "sqlite3":
		err = expirer.store.DBConn.DB.Select(&expiredIds, `
			SELECT id FROM uploads
			WHERE
				CAST(strftime('%s', 'now') AS INTEGER) -- current time
				>=
				created_at + (CASE WHEN jwt_account IS NULL THEN $1 ELSE $2 END) -- expiration time
			AND deleted != 1
			`,
			expirer.maxAge.Seconds(),
			expirer.identifiedMaxAge.Seconds(),
		)
	case "mysql":
		err = expirer.store.DBConn.DB.Select(&expiredIds, `
			SELECT id FROM uploads
			WHERE
				UNIX_TIMESTAMP() -- current time
				>=
				created_at + (CASE WHEN jwt_account IS NULL THEN ? ELSE ? END) -- expiration time
			AND deleted != 1
			`,
			expirer.maxAge.Seconds(),
			expirer.identifiedMaxAge.Seconds(),
		)
	default:
		panic("Unhandled database driver")
	}

	return
}

func (expirer *Expirer) deleteExpired() (err error) {
	switch expirer.store.DBConn.DBConfig.DriverName {
	case "sqlite3":
		_, err = expirer.store.DBConn.DB.Exec(`
			DELETE FROM uploads
			WHERE
				CAST(strftime('%s', 'now') AS INTEGER) -- current time
				>=
				created_at + (CASE WHEN jwt_account IS NULL THEN $1 ELSE $2 END) + $3 -- expiration time
			AND deleted == 1
			`,
			expirer.maxAge.Seconds(),
			expirer.identifiedMaxAge.Seconds(),
			expirer.deletedMaxAge.Seconds(),
		)
	case "mysql":
		_, err = expirer.store.DBConn.DB.Exec(`
			DELETE FROM uploads
			WHERE
				UNIX_TIMESTAMP() -- current time
				>=
				created_at + (CASE WHEN jwt_account IS NULL THEN ? ELSE ? END) + ? -- expiration time
			AND deleted == 1
			`,
			expirer.maxAge.Seconds(),
			expirer.identifiedMaxAge.Seconds(),
			expirer.deletedMaxAge.Seconds(),
		)
	default:
		panic("Unhandled database driver")
	}

	return
}
