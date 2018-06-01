package main

import (
	"errors"
	"net"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/c2h5oh/datasize"
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

	StoragePath             string            `env:"STORAGE_PATH"              envDefault:"./uploads"`
	StorageShardLayers      int               `env:"STORAGE_SHARD_LAYERS"      envDefault:"6"`
	DBType                  string            `env:"DATABASE_TYPE"             envDefault:"sqlite3"`
	DBPath                  string            `env:"DATABASE_PATH"             envDefault:"./uploads.db"`
	MaximumUploadSize       datasize.ByteSize `env:"MAXIMUM_UPLOAD_SIZE"       envDefault:"10 MB"`
	ExpirationAge           time.Duration     `env:"EXPIRATION_AGE"            envDefault:"168h"` // 1 week
	ExpirationCheckInterval time.Duration     `env:"EXPIRATION_CHECK_INTERVAL" envDefault:"5m"`

	// set global loglevel
	LogLevel zerolog.Level `env:"LOG_LEVEL" envDefault:"info"`

	TrustedReverseProxyRanges []*net.IPNet `env:"TRUSTED_REVERSE_PROXY_RANGES" envDefault:"10.0.0.0/8,172.16.0.0/12,192.168.0.0/16,fc00::/7,127.0.0.0/8,::1/128"`
}

// LoadFromEnv populates the config from the process environment and .env file
func (cfg *UploadServerConfig) LoadFromEnv() {
	// load values from .env file
	err := godotenv.Overload()
	if err != nil {
		if _, ok := err.(*os.PathError); ok {
			log.Debug().Msg("No .env file loaded")
		} else {
			log.Fatal().Err(err).Msg("failed to load .env")
		}
	}

	// populate config struct
	err = env.ParseWithFuncs(cfg, env.CustomParsers{
		reflect.TypeOf(zerolog.DebugLevel): logLevelParser,
		reflect.TypeOf(datasize.B):         byteSizeParser,
		reflect.TypeOf([]*net.IPNet{}):     ipNetSliceParser,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse config")
	}

	if len(cfg.TrustedReverseProxyRanges) > 0 {
		var trustedCidrs []string
		for _, cidr := range cfg.TrustedReverseProxyRanges {
			trustedCidrs = append(trustedCidrs, cidr.String())
		}
		log.Debug().Strs("trustedCidrs", trustedCidrs).Msg("Trusting reverse proxies")
	}

	zerolog.SetGlobalLevel(cfg.LogLevel)
}

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

func byteSizeParser(v string) (interface{}, error) {
	var bytes datasize.ByteSize
	err := bytes.UnmarshalText([]byte(v))
	if err != nil {
		log.Error().Err(err).Str("stringValue", v).Msg("failed to parse byteSize")
		return nil, err
	}
	return bytes, nil
}

func ipNetSliceParser(v string) (interface{}, error) {
	cidrStrings := strings.Split(v, ",")
	var cidrs []*net.IPNet
	for _, cidrString := range cidrStrings {
		_, cidr, err := net.ParseCIDR(cidrString)
		if err != nil {
			log.Error().Err(err).Str("cidrString", cidrString).Msg("failed to parse CIDR")
			return nil, err
		}
		cidrs = append(cidrs, cidr)
	}
	return cidrs, nil
}
