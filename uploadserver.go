package main

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/kiwiirc/plugin-fileuploader/expirer"
	"github.com/kiwiirc/plugin-fileuploader/shardedfilestore"
)

// UploadServer is a simple configurable service for file sharing.
// Compatible with TUS upload clients.
type UploadServer struct {
	cfg        UploadServerConfig
	store      *shardedfilestore.ShardedFileStore
	router     *gin.Engine
	expirer    *expirer.Expirer
	httpServer *http.Server
	startedMu  sync.Mutex
	started    chan struct{}
}

// GetStartedChan returns a channel that will close when the server startup is complete
func (serv *UploadServer) GetStartedChan() chan struct{} {
	serv.startedMu.Lock()
	defer serv.startedMu.Unlock()

	if serv.started == nil {
		serv.started = make(chan struct{})
	}

	return serv.started
}

func init() {
	gin.SetMode(gin.ReleaseMode)
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

	serv.httpServer = &http.Server{
		Addr:    serv.cfg.ListenAddr,
		Handler: serv.router,
	}

	// closed channel indicates that startup is complete
	close(serv.GetStartedChan())

	return serv.httpServer.ListenAndServe()
}

// Shutdown gracefully terminates the UploadServer instance.
// The HTTP listen socket will close immediately, causing the .Run() call to return.
// The call to .Shutdown() will block until all outstanding requests have been served and
// other resources like database connections and timers have been closed and stopped.
func (serv *UploadServer) Shutdown() {
	// wait for startup to complete
	<-serv.GetStartedChan()

	// wait for all requests to finish
	serv.httpServer.Shutdown(nil)

	// stop running FileStore GC cycles
	serv.expirer.Stop()

	// close db connections
	serv.store.Close()
}
