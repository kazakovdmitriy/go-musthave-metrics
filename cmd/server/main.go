package main

import (
	"net/http"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/handler"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	handler := handler.SetupHandler()
	return http.ListenAndServe(":8080", handler)
}
