package c

import (
	"log"
	"os"
)

func helperInTest() {
	panic("in test")
	log.Fatal("in test")
	os.Exit(1)
}
