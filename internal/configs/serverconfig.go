package configs

import (
	"flag"
	"github.com/caarlos0/env/v6"
	"log"
)

type ServerConfigParams struct {
	Host            string
	LogLevel        string
	StorageFile     string
	StorageInterval uint
	Restore         bool
	DBConnect       string
}

type ServerEnvParams struct {
	Address         string `env:"ADDRESS"`
	LogLevel        string `env:"LOG_LEVEL"`
	StorageFile     string `env:"FILE_STORAGE_PATH"`
	StorageInterval uint   `env:"STORE_INTERVAL"`
	Restore         bool   `env:"RESTORE"`
	DBConnect       string `env:"DATABASE_DSN"`
}

var ServerParams ServerConfigParams
var ServerEnvs ServerEnvParams

func GetServerConfigs() {
	flag.StringVar(&ServerParams.Host, "a", ":8080", "Server address and port number")
	flag.StringVar(&ServerParams.LogLevel, "l", "info", "log level")
	flag.StringVar(&ServerParams.StorageFile, "f", "/tmp/metrics-db.json", "Metrics metricsstorage file")
	flag.UintVar(&ServerParams.StorageInterval, "i", 300, "Write to file interval")
	flag.BoolVar(&ServerParams.Restore, "r", true, "Should be metrics loaded from file on start server")
	flag.StringVar(&ServerParams.DBConnect, "d", "host=localhost port=5432 dbname=postgres user=postgres password=123password", "All parameters for connection to database")
	flag.Parse()

	err := env.Parse(&ServerEnvs)
	if err != nil {
		log.Fatalln(err)
	}
	if address := ServerEnvs.Address; address != "" {
		ServerParams.Host = address
	}
	if logLevel := ServerEnvs.LogLevel; logLevel != "" {
		ServerParams.LogLevel = logLevel
	}
	if storageInterval := ServerEnvs.StorageInterval; storageInterval != 0 {
		ServerParams.StorageInterval = storageInterval
	}
	if storagePath := ServerEnvs.StorageFile; storagePath != "" {
		ServerParams.StorageFile = storagePath
	}
	if restore := ServerEnvs.Restore; restore {
		ServerParams.Restore = restore
	}
	if dbConnect := ServerEnvs.DBConnect; dbConnect != "" {
		ServerParams.DBConnect = dbConnect
	}
}

func (c *ServerConfigParams) SyncSave() bool {
	return c.StorageInterval == 0
}

func (c *ServerConfigParams) InitLoad() bool {
	return c.Restore == true
}
