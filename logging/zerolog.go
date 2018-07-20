package logging

import (
	"io"

	"github.com/rs/zerolog"
)

// SelectiveLevelWriter selectively writes log events to a writer based on the event level
type SelectiveLevelWriter struct {
	io.Writer
	Level zerolog.Level
}

// WriteLevel writes the payload to the wrapped io.Writer if the event level is within the desired range
func (slw SelectiveLevelWriter) WriteLevel(l zerolog.Level, p []byte) (n int, err error) {
	if l < slw.Level {
		// this log sink doesn't want to log events of this level

		// return length of payload we would have written. zerolog checks this even when error is nil
		return len(p), nil
	}
	return slw.Write(p)
}

// MaxLevel returns the more detailed of two logging levels
func MaxLevel(a, b zerolog.Level) zerolog.Level {
	// numerically lower levels are "higher" in detail
	if a < b {
		return a
	}
	return b
}
