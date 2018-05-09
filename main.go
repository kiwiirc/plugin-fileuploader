package main

import (
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	var wg sync.WaitGroup

	reloadRequested := make(chan struct{}, 1)
	done := make(chan struct{}, 1)

	// signal handler
	go signalHandler(reloadRequested, done)

	// server run loop
	wg.Add(1)
	go runServer(reloadRequested, done, &wg)

	wg.Wait()
	log.Info().Msg("Shutdown complete")
}

func signalHandler(reloadRequested, done chan struct{}) {
	signals := make(chan os.Signal, 1)

	signal.Notify(signals, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

	for {
		switch sig := <-signals; sig {

		case syscall.SIGHUP:
			reloadRequested <- struct{}{}

		case syscall.SIGINT:
			fallthrough
		case syscall.SIGTERM:
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
		log.Info().
			Str("address", serv.cfg.ListenAddr).
			Msg("Server listening")

		// wait for error or reload request
		shouldRestart := func() bool {
			for {
				select {

				case err := <-errChan:
					// quit if unexpected error occurred
					if err != http.ErrServerClosed {
						log.Fatal().
							Err(err).
							Msg("Error running upload server")
					}

					// server closed by request, exit loop to allow it to restart
					return true

				case <-reloadRequested:
					// Run in separate goroutine so we don't wait for .Shutdown()
					// to return before starting the new server.
					// This allows us to handle outstanding requests using the old
					// server instance while we've already replaced it as the listener
					// for new connections.
					go func() {
						log.Info().Msg("Reloading server config")
						serv.Shutdown()
					}()

				case <-done:
					log.Info().Msg("Shutdown initiated. Handling existing requests")
					serv.Shutdown()
					wg.Done()
					return false

				}
			}
		}()

		if !shouldRestart {
			return
		}
	}
}
