package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	_ "net/http/pprof"
)

func main() {
	// pprof
	/* go func() {
		pprofAddr := "localhost:6060"
		log.Println("Starting pprof on", pprofAddr)
		log.Println(http.ListenAndServe(pprofAddr, nil))
	}() */

	var wg sync.WaitGroup

	reloadRequested := make(chan struct{}, 1)
	done := make(chan struct{}, 1)

	// signal handler
	go signalHandler(reloadRequested, done)

	// server run loop
	wg.Add(1)
	go runServer(reloadRequested, done, &wg)

	wg.Wait()
	log.Println("Shutdown complete")
}

func signalHandler(reloadRequested, done chan struct{}) {
	signals := make(chan os.Signal, 1)

	signal.Notify(signals, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

	for {
		sig := <-signals
		switch sig {
		case syscall.SIGHUP:
			reloadRequested <- struct{}{}
		default:
			done <- struct{}{}
		}
	}
}

func runServer(reloadRequested, done chan struct{}, wg *sync.WaitGroup) {
	for {
		// new server instance
		serv := UploadServer{}

		// refresh config
		serv.cfg.LoadFromEnv()

		errChan := make(chan error)

		// run server until .Shutdown() called or other error occurs
		go func() {
			err := serv.Run()
			errChan <- err
		}()

		// wait for startup to complete
		<-serv.GetStartedChan()
		log.Println("Server listening on", serv.cfg.ListenAddr)

	ServerLoop:
		// wait for error or reload request
		for {
			select {

			case err := <-errChan:
				// quit if unexpected error occurred
				if err != http.ErrServerClosed {
					log.Fatalf("Error running upload server: %s", err)
				}

				// server closed by request, exit loop to allow it to restart
				break ServerLoop

			case <-reloadRequested:
				// Run in separate goroutine so we don't wait for .Shutdown()
				// to return before starting the new server.
				// This allows us to handle outstanding requests using the old
				// server instance while we've already replaced it as the listener
				// for new connections.
				go func() {
					log.Println("Reloading server config")
					serv.Shutdown()
				}()

			case <-done:
				log.Println("Shutting initiated. Handling existing requests")
				serv.Shutdown()
				wg.Done()
				return

			}
		}
	}
}
