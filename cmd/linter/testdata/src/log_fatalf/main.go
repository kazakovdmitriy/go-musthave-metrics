package main

import "log"

func helper() {
	log.Fatalf("exit") // want "log\\.Fatalf\\(\\) should only be called from main function in main package"
}
