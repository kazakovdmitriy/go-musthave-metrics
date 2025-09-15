package config

import "flag"

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

	flag.StringVar(&cfg.ServerAddr, "a", ":8080", "HTTP server port")

	flag.Parse()

	return &cfg
}

func ParseFlagsAgent() *AgentFlags {
	var cfg AgentFlags

	flag.StringVar(&cfg.ServerAddr, "a", "http://localhost:8080", "HTTP server port")
	flag.IntVar(&cfg.ReportInterval, "r", 10, "report interval in sec")
	flag.IntVar(&cfg.PollingInterval, "p", 2, "pooling interval in sec")

	flag.Parse()

	return &cfg
}
