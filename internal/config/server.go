package config

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/caarlos0/env/v6"
	"github.com/spf13/pflag"
)

type ServerFlags struct {
	ServerAddr      string   `env:"ADDRESS"`
	LogLevel        string   `env:"LOGLEVEL" envDefault:"info"`
	StoreInterval   int      `env:"STORE_INTERVAL"`
	FileStoragePath string   `env:"FILE_STORAGE_PATH"`
	Restore         bool     `env:"RESTORE"`
	DatabaseDSN     string   `env:"DATABASE_DSN"`
	SecretKet       string   `env:"KEY"`
	RateLimit       int      `env:"RATE_LIMIT"`
	MaxRetries      int      `env:"MAX_RETRIES"`
	RetryDelays     []string `env:"RETRY_DELAYS"`
}

func ParseServerConfig() *ServerFlags {

	var cfg ServerFlags

	setDefaultServerFlag(&cfg)
	parseServerFlag(&cfg)
	parseServerEnv(&cfg)

	return &cfg
}

func setDefaultServerFlag(cfg *ServerFlags) {
	cfg.ServerAddr = "http://localhost:8080"
	cfg.LogLevel = "info"
	cfg.StoreInterval = 300
	cfg.FileStoragePath = "metrics.json"
	cfg.Restore = false
	cfg.MaxRetries = 3
	cfg.RetryDelays = []string{"1s", "3s", "5s"}
}

func parseServerEnv(cfg *ServerFlags) {
	err := env.Parse(cfg)
	if err != nil {
		log.Printf("Warning: failed to parse environment variables: %v", err)
	}
}

func parseServerFlag(cfg *ServerFlags) {
	flags := pflag.NewFlagSet("server", pflag.ExitOnError)

	flags.StringVarP(&cfg.ServerAddr, "address", "a", ":8080", "HTTP server port")
	flags.StringVarP(&cfg.LogLevel, "loglevel", "g", "info", "Logger level")
	flags.IntVarP(&cfg.StoreInterval, "strIntrvl", "i", 300, "Disc save interval, s")
	flags.StringVarP(&cfg.FileStoragePath, "filePath", "f", "metrics.json", "Path to file to save metrics")
	flags.BoolVarP(&cfg.Restore, "restore", "r", false, "Load metrics on start")
	flags.StringVarP(&cfg.DatabaseDSN, "database_dsn", "d", "", "DSN string for db connection")
	flags.StringVarP(&cfg.SecretKet, "", "k", "", "Secret key")
	flags.IntVarP(&cfg.RateLimit, "ratelimit", "l", 0, "Rate limit")
	flags.IntVarP(&cfg.MaxRetries, "max-retries", "m", 3, "Maximum number of retry attempts")
	flags.StringArrayVarP(&cfg.RetryDelays, "retry-delays", "s", []string{"1s", "3s", "5s"}, "Retry delays between attempts")

	if err := flags.Parse(os.Args[1:]); err != nil {
		log.Printf("Error parsing command-line flags: %v", err)
	}

	if flags.NArg() > 0 {
		for i := 0; i < flags.NArg(); i++ {
			arg := flags.Arg(i)
			if len(arg) > 0 && arg[0] == '-' {
				log.Printf("Unknown flag: %s", arg)
			}
		}
	}
}

func (a *ServerFlags) GetRetryDelaysAsDuration() ([]time.Duration, error) {
	delays := make([]time.Duration, len(a.RetryDelays))
	for i, delayStr := range a.RetryDelays {
		delay, err := time.ParseDuration(delayStr)
		if err != nil {
			return nil, fmt.Errorf("invalid duration format '%s': %w", delayStr, err)
		}
		delays[i] = delay
	}
	return delays, nil
}
