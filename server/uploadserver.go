package server

import (
	"context"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/kiwiirc/plugin-fileuploader/config"
	"github.com/kiwiirc/plugin-fileuploader/db"
	"github.com/kiwiirc/plugin-fileuploader/events"
	"github.com/kiwiirc/plugin-fileuploader/expirer"
	"github.com/kiwiirc/plugin-fileuploader/logging"
	"github.com/kiwiirc/plugin-fileuploader/shardedfilestore"
	"github.com/rs/zerolog"
	tusd "github.com/tus/tusd/pkg/handler"
)

// UploadServer is a simple configurable service for file sharing.
// Compatible with TUS upload clients.
type UploadServer struct {
	DBConn *db.DatabaseConnection
	Router *gin.Engine

	cfg                 config.Config
	log                 *zerolog.Logger
	composer            *tusd.StoreComposer
	store               *shardedfilestore.ShardedFileStore
	expirer             *expirer.Expirer
	httpServer          *http.Server
	startedMu           sync.Mutex
	started             chan struct{}
	tusEventBroadcaster *events.TusEventBroadcaster
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
func (serv *UploadServer) Run(replaceableHandler *ReplaceableHandler) error {
	serv.Router = gin.New()
	serv.Router.Use(logging.GinLogger(serv.log), gin.Recovery())

	serv.DBConn = db.ConnectToDB(serv.log, db.DBConfig{
		DriverName: serv.cfg.Database.Type,
		DSN:        serv.cfg.Database.Path,
	})

	serv.store = shardedfilestore.New(
		serv.cfg.Storage.Path,
		serv.cfg.Storage.ShardLayers,
		serv.cfg.Expiration.MaxAge.Duration,
		serv.cfg.Expiration.IdentifiedMaxAge.Duration,
		serv.cfg.PreFinishCommands,
		serv.DBConn,
		serv.log,
	)

	serv.composer = tusd.NewStoreComposer()
	serv.store.UseIn(serv.composer)

	serv.expirer = expirer.New(
		serv.store,
		serv.cfg.Expiration.CheckInterval.Duration,
		serv.log,
	)

	err := serv.registerTusHandlers(serv.Router, serv.store)
	if err != nil {
		return err
	}

	// closed channel indicates that startup is complete
	close(serv.GetStartedChan())

	if replaceableHandler != nil {
		// set ReplaceableHandler that's mounted in an external server
		replaceableHandler.Handler = serv.Router
		return nil
	}

	// otherwise run our own http server
	serv.httpServer = &http.Server{
		Addr:    serv.cfg.Server.ListenAddress,
		Handler: serv.Router,
	}

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
	if serv.httpServer != nil {
		serv.httpServer.Shutdown(context.Background())
	}

	// stop running FileStore GC cycles
	serv.expirer.Stop()

	// close db connections
	serv.DBConn.DB.Close()

	// close event broadcaster
	serv.tusEventBroadcaster.Close()
}
