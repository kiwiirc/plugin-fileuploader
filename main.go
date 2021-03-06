package main

import (
	"flag"

	"github.com/kiwiirc/plugin-fileuploader/server"
)

func main() {
	var configPath = flag.String("config", "fileuploader.config.toml", "path to config file")
	flag.Parse()
	runCtx := server.NewRunContext(nil, *configPath)
	runCtx.Run()
}
