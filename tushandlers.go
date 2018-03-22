package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kiwiirc/fileuploader/shardedfilestore"
	"github.com/tus/tusd"
)

const tusRoutePrefix = "/files"

func (serv *UploadServer) registerTusHandlers(r *gin.Engine) error {
	store := shardedfilestore.New(serv.cfg.StoragePath, serv.cfg.StorageShardLayers)

	composer := tusd.NewStoreComposer()
	store.UseIn(composer)

	config := tusd.Config{
		BasePath:      tusRoutePrefix,
		StoreComposer: composer,
	}

	handler, err := tusd.NewUnroutedHandler(config)
	if err != nil {
		return err
	}

	noopHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("noop handler")
	})

	// For unknown reasons, this middleware must be mounted on the top level router.
	// When attached to the RouterGroup, it does not get called for some requests.
	r.Use(gin.WrapH(handler.Middleware(noopHandler)))

	rg := r.Group(tusRoutePrefix)
	rg.POST("", gin.WrapF(handler.PostFile))
	rg.HEAD(":id", gin.WrapF(handler.HeadFile))
	rg.PATCH(":id", gin.WrapF(handler.PatchFile))

	// Only attach the DELETE handler if the Terminate() method is provided
	if config.StoreComposer.UsesTerminater {
		rg.DELETE(":id", gin.WrapF(handler.DelFile))
	}

	// GET handler requires the GetReader() method
	if config.StoreComposer.UsesGetReader {
		rg.GET(":id", gin.WrapF(handler.GetFile))
	}

	return nil
}
