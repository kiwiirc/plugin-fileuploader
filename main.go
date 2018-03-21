package main

import (
	"os"

	"github.com/gin-gonic/gin"
)

const storagePath = "./uploads"

func main() {
	err := os.MkdirAll(storagePath, os.ModePerm)
	if err != nil {
		panic(err)
	}
	router := gin.Default()

	err = registerTusHandlers("/files", router)
	if err != nil {
		panic(err)
	}

	router.Run("0.0.0.0:8088")
}
