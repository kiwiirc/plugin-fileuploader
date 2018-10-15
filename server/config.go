package server

import (
	"errors"
	"fmt"
	"net"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/c2h5oh/datasize"
	"github.com/caarlos0/env"
	"github.com/joho/godotenv"
	"github.com/kiwiirc/plugin-fileuploader/logging"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type RemoteLogSink struct {
	Addr   net.Addr
	Format string
	Level  zerolog.Level
}

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

	// network address to send logs to
	//
	// format: <log-level>:<network-type>:<log-format>:<address/path>
	//
	// log level: debug, info, warn, error, fatal, panic
	// supported network types: udp, unix
	// supported log formats: json
	// examples:
	//   debug:unix:json:/run/fileuploader-log.sock
	//   info:udp:json:10.33.0.2:3333
	RemoteLogSink *RemoteLogSink `env:"REMOTE_LOG_SINK"`

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
	} else {
		log.Debug().Msg(".env file loaded")
	}

	// populate config struct
	err = env.ParseWithFuncs(cfg, env.CustomParsers{
		reflect.TypeOf(zerolog.DebugLevel): logLevelParser,
		reflect.TypeOf(datasize.B):         byteSizeParser,
		reflect.TypeOf([]*net.IPNet{}):     ipNetSliceParser,
		// reflect.TypeOf((*net.Addr)(nil)):   netAddrParser,
		reflect.TypeOf(&RemoteLogSink{}): remoteLogSinkParser,
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

	if cfg.RemoteLogSink != nil {
		network := cfg.RemoteLogSink.Addr.Network()
		address := cfg.RemoteLogSink.Addr.String()
		log.Debug().
			Str("network", network).
			Str("address", address).
			Str("format", cfg.RemoteLogSink.Format).
			Msg("sending logs to")
		conn, err := net.Dial(network, address)
		level := cfg.RemoteLogSink.Level
		if err != nil {
			log.Error().
				Err(err).
				Str("network", network).
				Str("address", address).
				Msg("Failed to dial network log sink")
		} else {
			log.Logger = log.Output(
				zerolog.MultiLevelWriter(
					logging.SelectiveLevelWriter{
						zerolog.ConsoleWriter{Out: os.Stderr},
						cfg.LogLevel,
					},
					logging.SelectiveLevelWriter{
						conn,
						level,
					},
				),
			)
			zerolog.SetGlobalLevel(logging.MaxLevel(cfg.LogLevel, level))
		}
	}
}

func logLevelParser(v string) (interface{}, error) {
	levelStr := strings.ToLower(v)
	return zerolog.ParseLevel(levelStr)
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

func remoteLogSinkParser(v string) (interface{}, error) {
	parts := strings.SplitN(v, ":", 4)
	if len(parts) != 4 {
		return nil, errors.New("Required format <log-level>:<network-type>:<log-format>:<address/path> not matched")
	}

	sink := &RemoteLogSink{}

	levelStr := parts[0]
	network := parts[1]
	format := parts[2]
	address := parts[3]

	level, err := zerolog.ParseLevel(levelStr)
	if err != nil {
		return nil, fmt.Errorf("Unknown Level String: '%s'", levelStr)
	}
	sink.Level = level

	switch format {
	case "json":
		sink.Format = format
		break
	default:
		return nil, errors.New("Unsupported log serialization format: " + format)
	}

	unixAddr, err := net.ResolveUnixAddr(network, address)
	if err == nil {
		sink.Addr = unixAddr
		return sink, nil
	}
	_, ok := err.(net.UnknownNetworkError)
	if !ok {
		return nil, err
	}

	udpAddr, err := net.ResolveUDPAddr(network, address)
	if err == nil {
		sink.Addr = udpAddr
		return sink, nil
	}
	_, ok = err.(net.UnknownNetworkError)
	if !ok {
		return nil, err
	}

	return nil, errors.New("Unhandled network type: " + network)
}
