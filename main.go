package main

import (
	"flag"

	"github.com/kiwiirc/plugin-fileuploader/server"
)

func main() {
	var configPath = flag.String("config", "fileuploader.config.toml", "path to config file")
	flag.Parse()
	server.RunServer(nil, *configPath)
}
