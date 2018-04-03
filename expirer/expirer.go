package expirer

import (
	"log"
	"time"

	"github.com/kiwiirc/fileuploader/shardedfilestore"
)

type Expirer struct {
	ticker *time.Ticker
	store  *shardedfilestore.ShardedFileStore
	maxAge time.Duration
}

func New(store *shardedfilestore.ShardedFileStore, maxAge, checkInterval time.Duration) *Expirer {
	expirer := &Expirer{
		time.NewTicker(checkInterval),
		store,
		maxAge,
	}

	go func() {
		for t := range expirer.ticker.C {
			expirer.gc(t)
		}
	}()

	return expirer
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
			return
		}
		log.Println("Terminated upload id", id)
	}
}

func (expirer *Expirer) getExpired() (expiredIds []string, err error) {
	rows, err := expirer.store.Db.Query(`SELECT id FROM uploads WHERE created_at < $1`, time.Now().Add(-expirer.maxAge).Unix())
	defer rows.Close()
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
