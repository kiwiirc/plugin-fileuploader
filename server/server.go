package server

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/rs/zerolog"
	globalZerolog "github.com/rs/zerolog/log"
)

type RunContext struct {
	ShutdownPromise sync.WaitGroup

	parentRouter    *http.ServeMux
	configPath      string
	reloadSignals   chan os.Signal
	shutdownSignals chan os.Signal
	log             *zerolog.Logger
}

func NewRunContext(parentRouter *http.ServeMux, configPath string) *RunContext {
	runCtx := &RunContext{
		parentRouter:    parentRouter,
		configPath:      configPath,
		log:             &globalZerolog.Logger, // default global zerolog
		reloadSignals:   make(chan os.Signal, 1),
		shutdownSignals: make(chan os.Signal, 1),
	}
	runCtx.ShutdownPromise.Add(1)
	return runCtx
}

func (runCtx *RunContext) Run() {
	// signal handler
	go runCtx.signalHandler()

	// server run loop
	go runCtx.runLoop()

	runCtx.ShutdownPromise.Wait()

	runCtx.log.Info().
		Str("event", "shutdown").
		Msg("Shutdown complete")
}

func (runCtx *RunContext) signalHandler() {
	signals := make(chan os.Signal, 1)

	signal.Notify(signals, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

	for {
		switch sig := <-signals; sig {

		case syscall.SIGHUP:
			runCtx.reloadSignals <- sig

		case syscall.SIGINT:
			fallthrough
		case syscall.SIGTERM:
			runCtx.shutdownSignals <- sig
		}
	}
}

func (runCtx *RunContext) runLoop() {
	var replaceableHandler *ReplaceableHandler
	if runCtx.parentRouter != nil {
		replaceableHandler = &ReplaceableHandler{}
	}
	registeredPrefixes := make(map[string]struct{}, 0)

	for {
		// new server instance
		serv := UploadServer{}
		cfg := NewConfig()

		// refresh config
		md, err := cfg.Load(runCtx.log, runCtx.configPath)
		if err != nil {
			runCtx.log.Error().Err(err).Msg("Failed to load config")
			return
		}

		serv.cfg = *cfg

		multiLogger, err := createMultiLogger(serv.cfg.Loggers)
		if err != nil {
			runCtx.log.Err(err).Msg("Failed to create MultiLogger")
		}

		runCtx.log = multiLogger
		serv.log = runCtx.log
		runCtx.log.Info().Str("path", runCtx.configPath).Msg("Loaded config file")
		cfg.DoPostLoadLogging(runCtx.log, runCtx.configPath, md)

		// register handler on parentRouter if any, when prefix has not been previously registered
		if runCtx.parentRouter != nil {
			routePrefix, err := routePrefixFromBasePath(serv.cfg.Server.BasePath)
			if err != nil {
				panic(err)
			}
			if _, ok := registeredPrefixes[routePrefix]; !ok { // this prefix not yet registered
				registeredPrefixes[routePrefix] = struct{}{}
				runCtx.parentRouter.Handle(routePrefix, replaceableHandler)
				if !strings.HasSuffix(routePrefix, "/") {
					runCtx.parentRouter.Handle(routePrefix+"/", replaceableHandler)
				}
				runCtx.log.Info().
					Str("event", "startup").
					Str("routePrefix", routePrefix).
					Msg("Fileuploader handler mounted on parent router")
			}
		}

		errChan := make(chan error)

		// run server until .Shutdown() called or other error occurs
		go func() {
			err := serv.Run(replaceableHandler)
			if err != nil {
				errChan <- err
			}
		}()

		// wait for startup to complete
		<-serv.GetStartedChan()
		if runCtx.parentRouter == nil {
			runCtx.log.Info().
				Str("event", "startup").
				Str("address", serv.cfg.Server.ListenAddress).
				Msg("Server listening")
		}

		// wait for error or reload request
		shouldRestart := func() bool {
			select {

			case err := <-errChan:

				fmt.Printf("errChan: %#v\n", err)
				// quit if unexpected error occurred
				if err != http.ErrServerClosed {
					runCtx.log.Fatal().
						Err(err).
						Msg("Error running upload server")
				}

				// server closed by request, exit loop to allow it to restart
				return true

			case <-runCtx.reloadSignals:
				// Run in separate goroutine so we don't wait for .Shutdown()
				// to return before starting the new server.
				// This allows us to handle outstanding requests using the old
				// server instance while we've already replaced it as the listener
				// for new connections.
				go func() {
					runCtx.log.Info().
						Str("event", "config_reload").
						Msg("Reloading server config")
					serv.Shutdown()
				}()
				return true

			case <-runCtx.shutdownSignals:
				runCtx.log.Info().
					Str("event", "shutdown_started").
					Msg("Shutdown initiated. Handling existing requests")
				serv.Shutdown()
				runCtx.ShutdownPromise.Done()
				return false

			}
		}()

		if !shouldRestart {
			return
		}
	}
}
