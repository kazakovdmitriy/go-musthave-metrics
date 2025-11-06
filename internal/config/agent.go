package config

import (
	"fmt"
	"os"

	"github.com/caarlos0/env/v6"
	"github.com/spf13/pflag"
)

type AgentFlags struct {
	ServerAddr      string `env:"ADDRESS"`
	ReportInterval  int    `env:"REPORT_INTERVAL"`
	PollingInterval int    `env:"POLL_INTERVAL"`
	LogLevel        string `env:"LOGLEVEL" envDefault:"info"`
	SecretKey       string `env:"KEY"`
	RateLimit       int    `env:"RATE_LIMIT"`
}

func ParseAgentConfig() *AgentFlags {
	var cfg AgentFlags

	// Устанавливаем значения по умолчанию
	setDefaultAgentFlag(&cfg)

	// Перезаписываем флагами
	parseFlagsAgent(&cfg)

	// Устанавливаем переменные окружения
	parseEnvAgent(&cfg)

	return &cfg
}

func setDefaultAgentFlag(cfg *AgentFlags) {
	cfg.ServerAddr = "http://localhost:8080"
	cfg.ReportInterval = 10
	cfg.PollingInterval = 2
}

func parseEnvAgent(cfg *AgentFlags) {
	err := env.Parse(cfg)
	if err != nil {
		fmt.Println(err)
	}
}

func parseFlagsAgent(cfg *AgentFlags) {
	flags := pflag.NewFlagSet("agent", pflag.ExitOnError)

	flags.StringVarP(&cfg.ServerAddr, "address", "a", "http://localhost:8080", "HTTP server port")
	flags.IntVarP(&cfg.ReportInterval, "report", "r", 10, "report interval in sec")
	flags.IntVarP(&cfg.PollingInterval, "poll", "p", 2, "polling interval in sec")
	flags.StringVarP(&cfg.SecretKey, "", "k", "", "Secret key")
	flags.IntVarP(&cfg.RateLimit, "ratelimit", "l", 0, "Rate limit")

	if err := flags.Parse(os.Args[1:]); err != nil {
		_, err := fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		if err != nil {
			return
		}
		os.Exit(1)
	}

	// Проверяем наличие неизвестных флагов
	if flags.NArg() > 0 {
		for i := 0; i < flags.NArg(); i++ {
			arg := flags.Arg(i)
			if len(arg) > 0 && arg[0] == '-' {
				_, err := fmt.Fprintf(os.Stderr, "Error: unknown flag: %s\n", arg)
				if err != nil {
					return
				}
				os.Exit(1)
			}
		}
	}
}
