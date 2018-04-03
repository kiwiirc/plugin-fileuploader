package main

import (
	"github.com/gin-gonic/gin"
	"github.com/kiwiirc/fileuploader/expirer"
	"github.com/kiwiirc/fileuploader/shardedfilestore"
)

// UploadServer is a simple configurable service for file sharing.
// Compatible with TUS upload clients.
type UploadServer struct {
	cfg     UploadServerConfig
	store   *shardedfilestore.ShardedFileStore
	router  *gin.Engine
	expirer *expirer.Expirer
}

// Run starts the UploadServer
func (serv *UploadServer) Run() error {
	serv.router = gin.Default()
	serv.store = shardedfilestore.New(
		serv.cfg.StoragePath,
		serv.cfg.StorageShardLayers,
		serv.cfg.DBPath,
		serv.cfg.MaximumUploadSize,
	)
	serv.expirer = expirer.New(
		serv.store,
		serv.cfg.ExpirationAge,
		serv.cfg.ExpirationCheckInterval,
	)

	err := serv.registerTusHandlers(serv.router, serv.store)
	if err != nil {
		return err
	}

	return serv.router.Run(serv.cfg.ListenAddr)
}
