package main

import "log"

func main() {
	var serv UploadServer

	serv.cfg.LoadFromEnv()

	err := serv.Run()
	if err != nil {
		log.Fatalf("Could not start upload server: %s", err)
	}
}
