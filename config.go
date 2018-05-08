package main

import (
	"log"
	"os"
	"time"

	"github.com/caarlos0/env"
	"github.com/joho/godotenv"
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
}

// LoadFromEnv populates the config from the process environment and .env file
func (cfg *UploadServerConfig) LoadFromEnv() {
	// load values from .env file
	err := godotenv.Overload()
	if err != nil {
		if _, ok := err.(*os.PathError); ok {
			log.Println("no .env file loaded")
		} else {
			log.Fatalf("%#v", err)
		}
	}

	// populate config struct
	err = env.Parse(cfg)
	if err != nil {
		log.Fatal(err)
	}
}
