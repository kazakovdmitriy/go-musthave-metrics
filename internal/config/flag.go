package config

import (
	"fmt"
	"os"

	"github.com/spf13/pflag"
)

type ServerFlags struct {
	ServerAddr string
}

type AgentFlags struct {
	ServerAddr      string
	ReportInterval  int
	PollingInterval int
}

func ParseFlagsServer() *ServerFlags {

	var cfg ServerFlags

	flags := pflag.NewFlagSet("server", pflag.ExitOnError)

	flags.StringVarP(&cfg.ServerAddr, "address", "a", ":8080", "HTTP server port")

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

	return &cfg
}

func ParseFlagsAgent() *AgentFlags {
	var cfg AgentFlags

	flags := pflag.NewFlagSet("agent", pflag.ExitOnError)

	flags.StringVarP(&cfg.ServerAddr, "address", "a", "http://localhost:8080", "HTTP server port")
	flags.IntVarP(&cfg.ReportInterval, "report", "r", 10, "report interval in sec")
	flags.IntVarP(&cfg.PollingInterval, "poll", "p", 2, "polling interval in sec")

	if err := flags.Parse(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Проверяем наличие неизвестных флагов
	if flags.NArg() > 0 {
		for i := 0; i < flags.NArg(); i++ {
			arg := flags.Arg(i)
			if len(arg) > 0 && arg[0] == '-' {
				fmt.Fprintf(os.Stderr, "Error: unknown flag: %s\n", arg)
				os.Exit(1)
			}
		}
	}

	return &cfg
}
