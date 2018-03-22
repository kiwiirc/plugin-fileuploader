package main

import (
	"github.com/gin-gonic/gin"
)

const storagePath = "./uploads"

func main() {
	router := gin.Default()

	err := registerTusHandlers("/files", router)
	if err != nil {
		panic(err)
	}

	router.Run("0.0.0.0:8088")
}
