package main

import (
	"fmt"
	"time"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/agent"
)

func main() {

	client := agent.NewClient("http://localhost:8080/update")

	for {
		_, err := agent.SendMetrics(client)
		if err != nil {
			fmt.Println("error from server: ", err)
		}
		time.Sleep(2 * time.Second)
	}
}
