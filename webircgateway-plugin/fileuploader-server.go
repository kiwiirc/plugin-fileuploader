package main

// symlink or copy this file into your webircgateway/plugins/fileuploader/plugin.go

import (
	"sync"

	"github.com/kiwiirc/plugin-fileuploader/server"
	"github.com/kiwiirc/webircgateway/pkg/webircgateway"
)

func Start(gateway *webircgateway.Gateway, pluginsQuit *sync.WaitGroup) {
	gateway.Log(1, "Starting fileuploader-server plugin. webircgateway version: %s", webircgateway.Version)

	go func() {
		defer pluginsQuit.Done()
		s := server.NewRunContext(gateway.HttpRouter, "fileuploader.config.toml")
		s.Run()
	}()
}
