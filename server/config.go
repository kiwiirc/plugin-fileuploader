package server

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/c2h5oh/datasize"
	"github.com/kiwiirc/plugin-fileuploader/logging"
	"github.com/rs/zerolog"
)

type LoggerConfig struct {
	Level  logLevel
	Format logFormat
	Output logOutput
}

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
		DeletedMaxAge    duration
		CheckInterval    duration
	}
	JwtSecretsByIssuer map[string]string
	Loggers            []LoggerConfig

	// WebPreview config options
	WebPreview struct {
		WebPreviewDisabled bool
		OembedProviderFile string
		TemplatesDirectory string
		CacheMaxAge        duration
		CacheCleanInterval duration

		// Fallback provider configuration
		FallbackProviderDisabled bool
		FallbackProviderURL      string
		FallbackProviderFile     string
		FallbackProviderJsonKey  string
	}
}

func NewConfig() *Config {
	cfg := &Config{}
	md, err := toml.Decode(defaultConfig, cfg)
	if err != nil {
		panic("Failed to decode defaultConfig")
	}
	undecoded := md.Undecoded()
	if len(undecoded) > 0 {
		panic(fmt.Sprintf("Undecoded keys: %q", undecoded))
	}
	return cfg
}

func (cfg *Config) Load(log *zerolog.Logger, configPath string) (toml.MetaData, error) {
	md, configLoadErr := toml.DecodeFile(configPath, cfg)
	return md, configLoadErr
}

func (cfg *Config) DoPostLoadLogging(log *zerolog.Logger, configPath string, md toml.MetaData) {
	undecoded := md.Undecoded()
	if len(undecoded) > 0 {
		var keys []string
		for _, key := range undecoded {
			keys = append(keys, key.String())
		}
		log.Warn().
			Strs("keys", keys).
			Msg("Extraneous configuration data")
	}

	if len(cfg.Server.TrustedReverseProxyRanges) > 0 {
		ranges := []string{}
		for _, rang := range cfg.Server.TrustedReverseProxyRanges {
			ranges = append(ranges, rang.String())
		}
		log.Debug().
			Strs("trustedCidrs", ranges).
			Msg("Trusting reverse proxies")
	}
}

func createMultiLogger(loggerConfigs []LoggerConfig) (*zerolog.Logger, error) {
	var writers []io.Writer
	for _, loggerCfg := range loggerConfigs {
		var output io.Writer
		url := loggerCfg.Output.URL
		switch url.Scheme {
		case "file":
			file, err := os.OpenFile(url.Opaque, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0640)
			if err != nil {
				return nil, err
			}
			output = file
		case "stderr":
			output = os.Stderr
		case "stdout":
			output = os.Stdout
		case "unix":
			fallthrough
		case "udp":
			fallthrough
		case "tcp":
			conn, err := net.Dial(url.Scheme, url.Opaque)
			if err != nil {
				return nil, err
			}
			output = conn
		default:
			fmt.Printf("working url %#v\n", url)
			return nil, errors.New("invalid log url scheme: " + url.Scheme)
		}

		switch loggerCfg.Format {
		case logFormat{"json"}:
			break
		case logFormat{"pretty"}:
			output = zerolog.ConsoleWriter{Out: output}
		default:
			return nil, errors.New("invalid log format")
		}

		levelWriter := logging.SelectiveLevelWriter{
			Writer: output,
			Level:  loggerCfg.Level.Level,
		}
		writers = append(writers, levelWriter)
	}

	multiLogger := zerolog.New(zerolog.MultiLevelWriter(writers...)).With().Timestamp().Logger()
	return &multiLogger, nil
}

////////////////////////////////////////////////////////////////
//     private types implementing encoding.TextUnmarshaler    //
////////////////////////////////////////////////////////////////

type logLevel struct {
	zerolog.Level
}

func (l *logLevel) UnmarshalText(text []byte) error {
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

type logFormat struct {
	string
}

func (f *logFormat) UnmarshalText(text []byte) error {
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

type logOutput struct {
	*url.URL
}

func (o *logOutput) UnmarshalText(text []byte) error {
	str := string(text)
	u, err := url.Parse(str)
	if err != nil {
		return err
	}
	o.URL = u
	return nil
}
