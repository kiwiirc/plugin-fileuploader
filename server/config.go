package server

import (
	"errors"
	"net"
	"os"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/c2h5oh/datasize"
	"github.com/kiwiirc/plugin-fileuploader/logging"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Config struct {
	Server struct {
		ListenAddress             string
		BasePath                  string
		CorsOrigins               []string
		TrustedReverseProxyRanges []ipnet
	}
	Storage struct {
		Path              string
		ShardLayers       int
		MaximumUploadSize datasize.ByteSize
	}
	Database struct {
		Type string
		Path string
	}
	Expiration struct {
		MaxAge           duration
		IdentifiedMaxAge duration
		CheckInterval    duration
	}
	JwtSecretsByIssuer map[string]string
	Logging            struct {
		Level      loglevel
		Format     format
		Output     output
		RemoteSink *struct {
			LogLevel loglevel
			Format   format
			Protocol string
			Address  string
		}
	}
}

func NewConfig() *Config {
	cfg := &Config{}
	_, err := toml.Decode(defaultConfig, cfg)
	if err != nil {
		log.Fatal().Err(err).
			Msg("Failed to decode defaultConfig")
	}
	return cfg
}

var defaultGlobalLogger = log.Logger

func (cfg *Config) Load(configPath string) error {
	_, configLoadErr := toml.DecodeFile(configPath, cfg)

	// configure main logger
	zerolog.SetGlobalLevel(cfg.Logging.Level.Level)
	switch cfg.Logging.Format.string {
	case "pretty":
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: cfg.Logging.Output})
	case "json":
		log.Logger = defaultGlobalLogger
	}

	if configLoadErr == nil {
		log.Info().Str("path", configPath).Msg("Loaded config file")
	} else {
		log.Error().Err(configLoadErr).Msg("Failed to parse config")
	}

	// just debug logging
	if len(cfg.Server.TrustedReverseProxyRanges) > 0 {
		ranges := []string{}
		for _, rang := range cfg.Server.TrustedReverseProxyRanges {
			ranges = append(ranges, rang.String())
		}
		log.Debug().
			Strs("trustedCidrs", ranges).
			Msg("Trusting reverse proxies")
	}

	// dial the remote sinks and configure logger to output to them
	if sink := cfg.Logging.RemoteSink; sink != nil {
		log.Debug().
			Str("protocol", sink.Protocol).
			Str("address", sink.Address).
			Str("format", sink.Format.string).
			Str("level", sink.LogLevel.String()).
			Msg("Sending logs to")
		conn, err := net.Dial(sink.Protocol, sink.Address)
		level := sink.LogLevel.Level
		if err != nil {
			log.Error().
				Err(err).
				Str("protocol", sink.Protocol).
				Str("address", sink.Address).
				Msg("Failed to dial network log sink")
		} else {
			log.Logger = log.Output(
				zerolog.MultiLevelWriter(
					logging.SelectiveLevelWriter{
						zerolog.ConsoleWriter{Out: os.Stderr},
						cfg.Logging.Level.Level,
					},
					logging.SelectiveLevelWriter{
						conn,
						level,
					},
				),
			)

			// events must pass through the global log level before our
			// SelectiveLevelWriters can filter them down
			zerolog.SetGlobalLevel(logging.MaxLevel(cfg.Logging.Level.Level, level))
		}
		return err
	}
	return nil
}

////////////////////////////////////////////////////////////////
//     private types implementing encoding.TextUnmarshaler    //
////////////////////////////////////////////////////////////////

type loglevel struct {
	zerolog.Level
}

func (l *loglevel) UnmarshalText(text []byte) error {
	levelStr := strings.ToLower(string(text))
	level, err := zerolog.ParseLevel(levelStr)
	l.Level = level
	return err
}

////////////////////////////////////////////////////////////////

type ipnet struct {
	net.IPNet
}

func (i *ipnet) UnmarshalText(text []byte) error {
	_, cidr, err := net.ParseCIDR(string(text))
	i.IPNet = *cidr
	return err
}

////////////////////////////////////////////////////////////////

type duration struct {
	time.Duration
}

func (d *duration) UnmarshalText(text []byte) error {
	dur, err := time.ParseDuration(string(text))
	d.Duration = dur
	return err
}

////////////////////////////////////////////////////////////////

type format struct {
	string
}

func (f *format) UnmarshalText(text []byte) error {
	formatStr := string(text)
	switch formatStr {
	case "json":
		fallthrough
	case "pretty":
		f.string = formatStr
	default:
		return errors.New("Unsupported log serialization format: " + formatStr)
	}
	return nil
}

////////////////////////////////////////////////////////////////

type output struct {
	*os.File
}

func (o *output) UnmarshalText(text []byte) error {
	outputStr := string(text)
	switch outputStr {
	case "stderr":
		o.File = os.Stderr
	case "stdout":
		o.File = os.Stdout
	default:
		return errors.New("Unsupported log output: " + outputStr)
	}
	return nil
}
