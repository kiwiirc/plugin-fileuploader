package main

import (
	"github.com/gin-gonic/gin"
)

// UploadServer is a simple configurable service for file sharing.
// Compatible with TUS upload clients.
type UploadServer struct {
	cfg UploadServerConfig
}

// Run starts the UploadServer
func (serv *UploadServer) Run() error {
	router := gin.Default()

	err := serv.registerTusHandlers(router)
	if err != nil {
		return err
	}

	return router.Run(serv.cfg.ListenAddr)
}
