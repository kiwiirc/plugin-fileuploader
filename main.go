package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/tus/tusd"
	"github.com/tus/tusd/filestore"
)

const storagePath = "./uploads"

func main() {
	err := os.MkdirAll(storagePath, os.ModePerm)
	if err != nil {
		panic(err)
	}
	router := gin.Default()
	/* router.Use(func(c *gin.Context) {
		fmt.Println("debug me")
	}) */
	// tusGroup := router.Group("/files")
	// tusGroup.Use(cors.New(cors.Config{
	// 	AllowOrigins: []string{"http://127.0.0.1:8080"},
	// 	AllowHeaders: []string{
	// 		"Tus-Resumable",
	// 		"Upload-Length",
	// 		"Upload-Offset",
	// 		"Content-Type",
	// 		"Upload-Metadata",
	// 	},
	// 	AllowMethods: []string{ /* "GET", "POST", "PUT", "HEAD", */ "PATCH"},
	// }))
	err = registerTusHandlers(router)
	if err != nil {
		panic(err)
	}
	// tusHandler := makeTusHandler()
	// router.POST("/files", gin.WrapH(tusHandler.Handler))
	// router.Any("/files", gin.WrapH(handler.Handler))
	router.Run("127.0.0.1:8088")
}

/* func makeTusHandler() *tusd.Handler {
	store := filestore.FileStore{
		Path: "./uploads",
	}

	composer := tusd.NewStoreComposer()
	store.UseIn(composer)

	handler, err := tusd.NewHandler(tusd.Config{
		BasePath:      "/files",
		StoreComposer: composer,
	})
	if err != nil {
		panic(fmt.Errorf("Unable to create handler: %s", err))
	}
	return handler
} */

func registerTusHandlers(r *gin.Engine) error {
	store := filestore.FileStore{
		Path: "./uploads",
	}

	composer := tusd.NewStoreComposer()
	store.UseIn(composer)

	config := tusd.Config{
		BasePath:      "/files",
		StoreComposer: composer,
	}

	handler, err := tusd.NewUnroutedHandler(config)
	if err != nil {
		return err
	}

	noopHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("noop handler")
	})

	r.Use(gin.WrapH(handler.Middleware(noopHandler)))

	rg := r.Group("/files")

	// routedHandler.Handler = handler.Middleware(mux)

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
