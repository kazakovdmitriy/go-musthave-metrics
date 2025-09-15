package main

import (
	"fmt"
	"net/http"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/config"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/handler"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {

	cfg := config.ParseFlagsServer()

	handler := handler.SetupHandler()

	fmt.Println("Run server on: ", cfg.ServerAddr)
	return http.ListenAndServe(cfg.ServerAddr, handler)
}
