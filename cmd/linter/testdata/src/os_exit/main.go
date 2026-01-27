package main

import "os"

func main() {
	os.Exit(0)
}

func helper() {
	os.Exit(1) // want "os\\.Exit\\(\\) should only be called from main function in main package called from os\\.Exit"
}
