package logging

// modeled after https://github.com/gin-gonic/gin/blob/v1.2/logger.go

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func GinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		end := time.Now()

		duration := end.Sub(start)

		var logEvent *zerolog.Event

		status := c.Writer.Status()
		switch {
		case 100 <= status && status <= 399:
			logEvent = log.Debug()
		case 400 <= status && status <= 499:
			logEvent = log.Warn()
		case 500 <= status && status <= 599:
			logEvent = log.Error()
		default:
			panic("Invalid status code")
		}

		logEvent.
			Str("event", "http_request").
			Int("status", status).
			Dur("duration", duration).
			Str("client", c.ClientIP()).
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path)

		unwrap := func(ginErrs []*gin.Error) (errs []error) {
			for _, ginErr := range ginErrs {
				errs = append(errs, ginErr.Err)
			}
			return
		}

		privateErrors := unwrap(c.Errors.ByType(gin.ErrorTypePrivate))
		if len(privateErrors) > 0 {
			logEvent.Errs("privateErrors", privateErrors)
		}

		publicErrors := unwrap(c.Errors.ByType(gin.ErrorTypePublic))
		if len(publicErrors) > 0 {
			logEvent.Errs("publicErrors", publicErrors)
		}

		logEvent.Msg("Handled request")
	}
}
