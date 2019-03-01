package expirer

import (
	"database/sql"
	"time"

	"github.com/kiwiirc/plugin-fileuploader/shardedfilestore"

	"github.com/rs/zerolog/log"
)

type Expirer struct {
	ticker             *time.Ticker
	store              *shardedfilestore.ShardedFileStore
	maxAge             time.Duration
	identifiedMaxAge   time.Duration
	jwtSecretsByIssuer map[string]string
	quitChan           chan struct{} // closes when ticker has been stopped
}

func New(store *shardedfilestore.ShardedFileStore, maxAge, identifiedMaxAge, checkInterval time.Duration, jwtSecretsByIssuer map[string]string) *Expirer {
	expirer := &Expirer{
		time.NewTicker(checkInterval),
		store,
		maxAge,
		identifiedMaxAge,
		jwtSecretsByIssuer,
		make(chan struct{}),
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
	log.Debug().
		Str("event", "gc_tick").
		Msg("Filestore GC tick")

	expiredIds, err := expirer.getExpired()
	if err != nil {
		log.Error().
			Err(err).
			Msg("Failed to enumerate expired uploads")
		return
	}

	for _, id := range expiredIds {
		err = expirer.store.Terminate(id)
		if err != nil {
			log.Error().
				Err(err).
				Msg("Failed to terminate expired upload")
			continue
		}
		log.Info().
			Str("event", "expired").
			Str("id", id).
			Msg("Terminated upload id")
	}
}

func (expirer *Expirer) getExpired() (expiredIds []string, err error) {
	rows, err := expirer.store.DBConn.DB.Query(`
		SELECT id FROM uploads
		WHERE
			CAST(strftime('%s', 'now') AS INTEGER) -- current time
			>=
			created_at + (CASE WHEN jwt_account IS NULL THEN :maxAge ELSE :identifiedMaxAge END) -- expiration time
		AND deleted != 1
		`,
		sql.Named("maxAge", expirer.maxAge.Seconds()),
		sql.Named("identifiedMaxAge", expirer.identifiedMaxAge.Seconds()),
	)

	if rows == nil || err != nil {
		return
	}

	defer rows.Close()

	var id string

	for rows.Next() {
		err = rows.Scan(&id)
		if err != nil {
			return
		}

		expiredIds = append(expiredIds, id)
	}

	return
}
