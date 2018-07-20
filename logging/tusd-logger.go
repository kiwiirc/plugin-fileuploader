// based on https://github.com/tus/tusd/blob/0.10.0/cmd/tusd/cli/hooks.go

package logging

import (
	"encoding/json"
	"strings"

	"github.com/kiwiirc/plugin-fileuploader/events"
	"github.com/rs/zerolog/log"
)

func TusdLogger(broadcaster *events.TusEventBroadcaster) {
	channel := broadcaster.Listen()
	for {
		event, ok := <-channel
		if !ok {
			return // channel closed
		}
		go handleTusEvent(event)
	}
}

func handleTusEvent(event *events.TusEvent) {
	logEvent := log.Info().
		Str("event", strings.Replace(string(event.Type), "-", "_", -1)).
		Str("id", event.Info.ID).
		Int64("size", event.Info.Size).
		Int64("offset", event.Info.Offset)

	metadataJSON, err := json.Marshal(event.Info.MetaData)
	if err != nil {
		log.Error().Err(err).Msg("Failed to serialize metadata")
	}
	logEvent.RawJSON("metadata", metadataJSON)

	logEvent.
		Bool("isPartial", event.Info.IsPartial).
		Strs("partialUploads", event.Info.PartialUploads)

	logEvent.Msg("Tusd " + string(event.Type))
}
