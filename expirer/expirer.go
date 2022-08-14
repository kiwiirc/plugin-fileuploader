package expirer

import (
	"time"

	"github.com/kiwiirc/plugin-fileuploader/shardedfilestore"
	"github.com/rs/zerolog"
)

type Expirer struct {
	ticker   *time.Ticker
	store    *shardedfilestore.ShardedFileStore
	quitChan chan struct{} // closes when ticker has been stopped
	log      *zerolog.Logger
}

func New(store *shardedfilestore.ShardedFileStore, checkInterval time.Duration, log *zerolog.Logger) *Expirer {
	expirer := &Expirer{
		ticker:   time.NewTicker(checkInterval),
		store:    store,
		quitChan: make(chan struct{}),
		log:      log,
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

	var expiredIds []string
	err := expirer.store.DBConn.DB.Select(&expiredIds, `
		SELECT id
		FROM uploads
		WHERE deleted = 0 AND (
			expires_at <= ? OR (expires_at IS NULL AND created_at <= ?)
		)`,
		time.Now().Unix(),
		time.Now().Unix()-86400, // 1 day
	)
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
}
