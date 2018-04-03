package main

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/kiwiirc/fileuploader/shardedfilestore"
	"github.com/tus/tusd"
)

const tusRoutePrefix = "/files"

func (serv *UploadServer) registerTusHandlers(r *gin.Engine, store *shardedfilestore.ShardedFileStore) error {
	composer := tusd.NewStoreComposer()
	store.UseIn(composer)

	config := tusd.Config{
		BasePath:      tusRoutePrefix,
		StoreComposer: composer,
		MaxSize:       serv.cfg.MaximumUploadSize,
	}

	handler, err := tusd.NewUnroutedHandler(config)
	if err != nil {
		return err
	}

	noopHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

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
		getFile := gin.WrapF(handler.GetFile)
		rg.GET(":id", getFile)
		rg.GET(":id/:filename", func(c *gin.Context) {
			// rewrite request path to ":id" route pattern
			c.Request.URL.Path = fmt.Sprintf("/files/%s", url.PathEscape(c.Param("id")))

			// call the normal handler
			getFile(c)
		})
	}

	return nil
}
