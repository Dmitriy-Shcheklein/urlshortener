package a

import (
	"log"
	"os"
)

func DoSomething() {
	panic("error")       // want "найден вызов panic\\(\\)"
	log.Fatal("fatal")   // want "найден вызов log\\.Fatal"
	os.Exit(1)           // want "найден вызов os\\.Exit"
}
