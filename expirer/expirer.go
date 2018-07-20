package expirer

import (
	"time"

	"github.com/kiwiirc/plugin-fileuploader/shardedfilestore"

	"github.com/rs/zerolog/log"
)

type Expirer struct {
	ticker   *time.Ticker
	store    *shardedfilestore.ShardedFileStore
	maxAge   time.Duration
	quitChan chan struct{} // closes when ticker has been stopped
}

func New(store *shardedfilestore.ShardedFileStore, maxAge, checkInterval time.Duration) *Expirer {
	expirer := &Expirer{
		time.NewTicker(checkInterval),
		store,
		maxAge,
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
		WHERE created_at < ?
		AND deleted != 1
	`, time.Now().Add(-expirer.maxAge).Unix())

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
