package config

import (
	"fmt"
	"os"

	"github.com/caarlos0/env/v6"
	"github.com/spf13/pflag"
)

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
}

func parseServerEnv(cfg *ServerFlags) {
	err := env.Parse(cfg)
	if err != nil {
		fmt.Println(err)
	}
}

func parseServerFlag(cfg *ServerFlags) {
	flags := pflag.NewFlagSet("server", pflag.ExitOnError)

	flags.StringVarP(&cfg.ServerAddr, "address", "a", ":8080", "HTTP server port")
	flags.StringVarP(&cfg.LogLevel, "loglevel", "l", "info", "Logger level")

	if err := flags.Parse(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if flags.NArg() > 0 {
		for i := 0; i < flags.NArg(); i++ {
			arg := flags.Arg(i)
			if len(arg) > 0 && arg[0] == '-' {
				fmt.Fprintf(os.Stderr, "Error: unknown flag: %s\n", arg)
				os.Exit(1)
			}
		}
	}
}
