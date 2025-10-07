package config

type ServerFlags struct {
	ServerAddr string `env:"ADDRESS"`
	LogLevel   string `env:"LOGLEVEL" envDefault:"info"`
}

type AgentFlags struct {
	ServerAddr      string `env:"ADDRESS"`
	ReportInterval  int    `env:"REPORT_INTERVAL"`
	PollingInterval int    `env:"POLL_INTERVAL"`
	LogLevel        string `env:"LOGLEVEL" envDefault:"info"`
}
