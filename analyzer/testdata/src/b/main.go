package main

import (
	"log"
	"os"
)

func main() {
	log.Fatal("allowed here")
	os.Exit(0)
}

func helper() {
	panic("report")   // want "найден вызов panic\\(\\)"
	log.Fatal("test") // want "найден вызов log\\.Fatal"
	os.Exit(1)        // want "найден вызов os\\.Exit"
}
