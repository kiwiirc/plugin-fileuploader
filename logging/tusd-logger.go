// based on https://github.com/tus/tusd/blob/0.10.0/cmd/tusd/cli/hooks.go

package logging

import (
	"github.com/rs/zerolog/log"
	"github.com/tus/tusd"
	"github.com/tus/tusd/cmd/tusd/cli"
)

func TusdLogger(handler *tusd.UnroutedHandler) {
	go func() {
		for {
			select {
			case info := <-handler.CompleteUploads:
				logHook(cli.HookPostFinish, info)
			case info := <-handler.TerminatedUploads:
				logHook(cli.HookPostTerminate, info)
			case info := <-handler.UploadProgress:
				logHook(cli.HookPostReceive, info)
			case info := <-handler.CreatedUploads:
				logHook(cli.HookPostCreate, info)
			}
		}
	}()
}

func logHook(typ cli.HookType, info tusd.FileInfo) {
	go func() {
		logEvent := log.Info().
			Str("id", info.ID).
			Int64("size", info.Size).
			Int64("offset", info.Offset)

		for k, v := range info.MetaData {
			logEvent.Str(k, v)
		}

		logEvent.
			Bool("isPartial", info.IsPartial).
			Strs("partialUploads", info.PartialUploads)

		logEvent.Msg(string(typ))
	}()
}
