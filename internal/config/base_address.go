package config

import (
	"errors"
	"flag"
	"log"
	"os"
	"strconv"
	"strings"
)

type BaseAddress struct {
	Host      string
	Port      int
	Protocol  string
	IsFromEnv bool
}

func NewBaseAddress() *BaseAddress {
	baseAddress := &BaseAddress{}

	flag.Var(baseAddress, "b", "Base address protocol://host:port")

	return baseAddress
}

func (a *BaseAddress) ApplyEnv() {
	baseURL, ok := os.LookupEnv("BASE_URL")
	if !ok {
		return
	}
	if err := a.Set(baseURL); err != nil {
		log.Fatalf("error while set BASE_URL env: %s", err)
	}
}

func (a *BaseAddress) String() string {
	return a.Host + ":" + strconv.Itoa(a.Port)
}

func (a *BaseAddress) Set(s string) error {
	if a.IsFromEnv {
		return nil
	}
	hp := strings.Split(s, ":")
	validLength := 3
	if len(hp) != validLength {
		return errors.New("need address in a form protocol://host:port")
	}
	port, err := strconv.Atoi(hp[2])
	if err != nil {
		return err
	}
	a.Protocol = hp[0]
	a.Host = hp[1]
	a.Port = port
	return nil
}

func (a *BaseAddress) IsFulfilled() bool {
	return a.Host != "" && a.Protocol != "" && a.Port != 0
}
