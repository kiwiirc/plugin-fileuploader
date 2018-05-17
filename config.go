package main

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/caarlos0/env"
	"github.com/joho/godotenv"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// UploadServerConfig contains settings that control the behavior of the UploadServer
type UploadServerConfig struct {
	ListenAddr string `env:"LISTEN_ADDR" envDefault:"127.0.0.1:8088"`

	// the externally reachable URL. this can differ from the listen address
	// when using a reverse proxy. can be relative ("/files") or absolute
	// ("https://example.com/files")
	BasePath string `env:"BASE_PATH" envDefault:"/files"`

	// comma separated list of CORS Origins to allow
	CorsOrigins []string `env:"CORS_ORIGINS" envSeparator:","`

	StoragePath             string        `env:"STORAGE_PATH"              envDefault:"./uploads"`
	StorageShardLayers      int           `env:"STORAGE_SHARD_LAYERS"      envDefault:"6"`
	DBType                  string        `env:"DATABASE_TYPE"             envDefault:"sqlite3"`
	DBPath                  string        `env:"DATABASE_PATH"             envDefault:"./uploads.db"`
	MaximumUploadSize       int64         `env:"MAXIMUM_UPLOAD_SIZE"       envDefault:"10485760"` // 10 MiB
	ExpirationAge           time.Duration `env:"EXPIRATION_AGE"            envDefault:"168h"`     // 1 week
	ExpirationCheckInterval time.Duration `env:"EXPIRATION_CHECK_INTERVAL" envDefault:"5m"`

	// enable debug logging
	LogLevel zerolog.Level `env:"LOG_LEVEL" envDefault:"info"`
}

// LoadFromEnv populates the config from the process environment and .env file
func (cfg *UploadServerConfig) LoadFromEnv() {
	// load values from .env file
	err := godotenv.Overload()
	if err != nil {
		if _, ok := err.(*os.PathError); ok {
			log.Debug().Msg("No .env file loaded")
		} else {
			log.Fatal().Err(err)
		}
	}

	// populate config struct
	err = env.ParseWithFuncs(cfg, env.CustomParsers{
		logLevelType: logLevelParser,
	})
	if err != nil {
		log.Fatal().Err(err)
	}

	zerolog.SetGlobalLevel(cfg.LogLevel)
}

var logLevelType = reflect.TypeOf(zerolog.DebugLevel)

func logLevelParser(v string) (interface{}, error) {
	level := strings.ToLower(v)
	switch level {
	case "debug":
		return zerolog.DebugLevel, nil
	case "info":
		return zerolog.InfoLevel, nil
	case "warn":
		return zerolog.WarnLevel, nil
	case "error":
		return zerolog.ErrorLevel, nil
	case "fatal":
		return zerolog.FatalLevel, nil
	case "panic":
		return zerolog.PanicLevel, nil
	default:
		return 0, errors.New("Invalid log level")
	}
}
