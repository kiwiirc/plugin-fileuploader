package server

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/davecgh/go-spew/spew"
	"github.com/kiwiirc/plugin-fileuploader/shardedfilestore"
	"github.com/rs/zerolog"
	"github.com/tus/tusd"
)

// ImageProxyCache is an implementation of httpcache.Cache that supplements the in-memory map with persistent storage
type ImageProxyCache struct {
	store  *shardedfilestore.ShardedFileStore
	log    *zerolog.Logger
	urlMap sync.Map
}

// Get returns the response corresponding to key if present
func (c *ImageProxyCache) Get(key string) (resp []byte, ok bool) {
	urlHash := getHash(key)
	spew.Dump("Get", key, urlHash)

	idInterface, ok := c.urlMap.Load(urlHash)
	if !ok {
		fmt.Println("Not in map")
		fmt.Println("")
		return []byte{}, false
	}
	id := idInterface.(string)

	reader, err := c.store.GetReader(id)
	if err != nil {
		fmt.Println("No reader")
		fmt.Println("")
		c.urlMap.Delete(urlHash)
		return []byte{}, false
	}

	buffer := new(bytes.Buffer)
	_, err = buffer.ReadFrom(reader)
	if err != nil {
		fmt.Println("Failed read")
		fmt.Println("")
		c.urlMap.Delete(urlHash)
		return []byte{}, false
	}

	bytes := buffer.Bytes()

	spew.Dump(len(bytes))
	fmt.Println("Got from cache")
	fmt.Println("")
	return bytes, true
}

// Set saves a response to the cache as key
func (c *ImageProxyCache) Set(key string, resp []byte) {
	urlHash := getHash(key)
	spew.Dump("Set", key, len(resp), urlHash)

	metaData := tusd.MetaData{
		"Url": key,
	}
	fileInfo := tusd.FileInfo{
		Size:           int64(len(resp)),
		SizeIsDeferred: false,
		MetaData:       metaData,
		IsFinal:        false,
	}

	id, err := c.store.NewUpload(fileInfo)
	if err != nil {
		c.log.Error().
			Err(err).
			Msg("Failed to create new upload")
		return
	}

	_, err = c.store.WriteChunk(id, 0, bytes.NewReader(resp))
	if err != nil {
		c.log.Error().
			Err(err).
			Msg("Failed to write chunk")
		return
	}

	c.store.FinishUpload(id)
	c.urlMap.Store(urlHash, id)

	fmt.Println("Set in cache")
	fmt.Println("")
}

// Delete removes the response with key from the cache
func (c *ImageProxyCache) Delete(key string) {
	urlHash := getHash(key)
	c.urlMap.Delete(urlHash)
	spew.Dump("Delete", key)
	fmt.Println("")
}

// NewImageProxyCache returns a new Cache that will store files in basePath
func NewImageProxyCache(store *shardedfilestore.ShardedFileStore, log *zerolog.Logger) *ImageProxyCache {
	return &ImageProxyCache{
		store: store,
		log:   log,
	}
}
