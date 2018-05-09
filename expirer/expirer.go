package expirer

import (
	"log"
	"time"

	"github.com/kiwiirc/plugin-fileuploader/shardedfilestore"
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
		make(chan struct{}, 1),
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
	log.Println("Filestore GC tick", t)

	expiredIds, err := expirer.getExpired()
	if err != nil {
		log.Println("Failed to enumerate expired uploads", err)
		return
	}

	for _, id := range expiredIds {
		err = expirer.store.Terminate(id)
		if err != nil {
			log.Println("Failed to terminate expired upload", err)
			continue
		}
		log.Println("Terminated upload id", id)
	}
}

func (expirer *Expirer) getExpired() (expiredIds []string, err error) {
	rows, err := expirer.store.Db.Query(`
		SELECT id FROM uploads
		WHERE created_at < ?
		AND deleted != 1
	`, time.Now().Add(-expirer.maxAge).Unix())
	if rows != nil {
		defer rows.Close()
	}
	if err != nil {
		return
	}

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
