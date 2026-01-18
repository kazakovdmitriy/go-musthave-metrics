package config

import (
	"fmt"
	"os"
	"time"

	"github.com/caarlos0/env/v6"
	"github.com/spf13/pflag"
)

type AgentFlags struct {
	ServerAddr      string   `env:"ADDRESS"`
	ReportInterval  int      `env:"REPORT_INTERVAL"`
	PollingInterval int      `env:"POLL_INTERVAL"`
	LogLevel        string   `env:"LOGLEVEL" envDefault:"info"`
	SecretKey       string   `env:"KEY"`
	RateLimit       int      `env:"RATE_LIMIT"`
	MaxRetries      int      `env:"MAX_RETRIES"`
	RetryDelays     []string `env:"RETRY_DELAYS"`
}

func ParseAgentConfig() (*AgentFlags, error) {
	var cfg AgentFlags

	// Устанавливаем значения по умолчанию
	setDefaultAgentFlag(&cfg)

	// Перезаписываем флагами
	err := parseFlagsAgent(&cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse flags: %w", err)
	}

	// Устанавливаем переменные окружения
	parseEnvAgent(&cfg)

	return &cfg, nil
}

func setDefaultAgentFlag(cfg *AgentFlags) {
	cfg.ServerAddr = "http://localhost:8080"
	cfg.ReportInterval = 10
	cfg.PollingInterval = 2
	cfg.MaxRetries = 3
	cfg.RetryDelays = []string{"1s", "3s", "5s"}
}

func parseEnvAgent(cfg *AgentFlags) {
	err := env.Parse(cfg)
	if err != nil {
		fmt.Println(err)
	}
}

func parseFlagsAgent(cfg *AgentFlags) error {
	flags := pflag.NewFlagSet("agent", pflag.ExitOnError)

	flags.StringVarP(&cfg.ServerAddr, "address", "a", "http://localhost:8080", "HTTP server port")
	flags.IntVarP(&cfg.ReportInterval, "report", "r", 10, "report interval in sec")
	flags.IntVarP(&cfg.PollingInterval, "poll", "p", 2, "polling interval in sec")
	flags.StringVarP(&cfg.SecretKey, "", "k", "", "Secret key")
	flags.IntVarP(&cfg.RateLimit, "ratelimit", "l", 0, "Rate limit")
	flags.IntVarP(&cfg.MaxRetries, "max-retries", "m", 3, "Maximum number of retry attempts")
	flags.StringArrayVarP(&cfg.RetryDelays, "retry-delays", "d", []string{"1s", "3s", "5s"}, "Retry delays between attempts")

	if err := flags.Parse(os.Args[1:]); err != nil {
		_, err := fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		if err != nil {
			return fmt.Errorf("failed to parse flags: %w", err)
		}
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	// Проверяем наличие неизвестных флагов
	if flags.NArg() > 0 {
		for i := 0; i < flags.NArg(); i++ {
			arg := flags.Arg(i)
			if len(arg) > 0 && arg[0] == '-' {
				_, err := fmt.Fprintf(os.Stderr, "Error: unknown flag: %s\n", arg)
				if err != nil {
					return fmt.Errorf("failed to parse flags: %w", err)
				}
				return fmt.Errorf("failed to parse flags: %w", err)
			}
		}
	}

	return nil
}

func (a *AgentFlags) GetRetryDelaysAsDuration() ([]time.Duration, error) {
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
