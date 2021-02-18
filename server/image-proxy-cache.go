package server

import (
	"bytes"
	"sync"

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

	idInterface, ok := c.urlMap.Load(urlHash)
	if !ok {
		// Not in map
		return []byte{}, false
	}
	id := idInterface.(string)

	reader, err := c.store.GetReader(id)
	if err != nil {
		// No file to read
		c.log.Debug().
			Err(err).
			Msg("Image missing from shardedfilestore, maybe it was cleaned")
		c.urlMap.Delete(urlHash)
		return []byte{}, false
	}

	buffer := new(bytes.Buffer)
	_, err = buffer.ReadFrom(reader)
	if err != nil {
		// Read error
		c.log.Debug().
			Err(err).
			Msg("Failed to read image from shardedfilestore")
		c.urlMap.Delete(urlHash)
		return []byte{}, false
	}

	bytes := buffer.Bytes()
	return bytes, true
}

// Set saves a response to the cache as key
func (c *ImageProxyCache) Set(key string, resp []byte) {
	urlHash := getHash(key)

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
}

// Delete removes the response with key from the cache
func (c *ImageProxyCache) Delete(key string) {
	urlHash := getHash(key)
	c.urlMap.Delete(urlHash)
}

// NewImageProxyCache returns a new Cache that will store files in basePath
func NewImageProxyCache(store *shardedfilestore.ShardedFileStore, log *zerolog.Logger) *ImageProxyCache {
	return &ImageProxyCache{
		store: store,
		log:   log,
	}
}
