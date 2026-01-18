package notmain

import "log"

func main() {
	log.Fatal("exit") // want "log\\.Fatal\\(\\) should only be called from main function in main package"
}
