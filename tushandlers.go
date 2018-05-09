package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"

	"github.com/gin-gonic/gin"
	"github.com/kiwiirc/plugin-fileuploader/logging"
	"github.com/kiwiirc/plugin-fileuploader/shardedfilestore"
	"github.com/tus/tusd"
)

func routePrefixFromBasePath(basePath string) (string, error) {
	url, err := url.Parse(basePath)
	if err != nil {
		return "", err
	}

	return url.Path, nil
}

func customizedCors(allowedOrigins []string) gin.HandlerFunc {
	// convert slice values to keys of map for "contains" test
	originSet := make(map[string]struct{}, len(allowedOrigins))
	exists := struct{}{}
	for _, origin := range allowedOrigins {
		originSet[origin] = exists
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		respHeader := c.Writer.Header()

		// only allow the origin if it's in the list from the config, * is not supported!
		if _, ok := originSet[origin]; ok {
			respHeader.Set("Access-Control-Allow-Origin", origin)
		} else {
			respHeader.Del("Access-Control-Allow-Origin")
		}

		// lets the user-agent know the response can vary depending on the origin of the request.
		// ensures correct behavior of browser cache.
		respHeader.Add("Vary", "Origin")
	}
}

func (serv *UploadServer) registerTusHandlers(r *gin.Engine, store *shardedfilestore.ShardedFileStore) error {
	composer := tusd.NewStoreComposer()
	store.UseIn(composer)

	config := tusd.Config{
		BasePath:                serv.cfg.BasePath,
		StoreComposer:           composer,
		MaxSize:                 serv.cfg.MaximumUploadSize,
		Logger:                  log.New(ioutil.Discard, "", 0),
		NotifyCompleteUploads:   true,
		NotifyCreatedUploads:    true,
		NotifyTerminatedUploads: true,
		NotifyUploadProgress:    true,
	}

	routePrefix, err := routePrefixFromBasePath(serv.cfg.BasePath)
	if err != nil {
		return err
	}

	handler, err := tusd.NewUnroutedHandler(config)
	if err != nil {
		return err
	}

	logging.TusdLogger(handler)

	noopHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	// For unknown reasons, this middleware must be mounted on the top level router.
	// When attached to the RouterGroup, it does not get called for some requests.
	tusdMiddleware := gin.WrapH(handler.Middleware(noopHandler))
	r.Use(tusdMiddleware)
	r.Use(customizedCors(serv.cfg.CorsOrigins))

	rg := r.Group(routePrefix)
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
			c.Request.URL.Path = path.Join(routePrefix, url.PathEscape(c.Param("id")))

			// call the normal handler
			getFile(c)
		})
	}

	return nil
}
