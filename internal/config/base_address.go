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
	Host     string
	Port     int
	Protocol string
}

func NewBaseAddress() *BaseAddress {
	baseAddress := &BaseAddress{}

	if baseUrl := os.Getenv("BASE_URL"); baseUrl != "" {
		if err := baseAddress.Set(baseUrl); err != nil {
			log.Fatalf("error while set BASE_URL env: %s", err)
		}
		return baseAddress
	}
	_ = flag.Value(baseAddress)
	flag.Var(baseAddress, "b", "Base address protocol://host:port")

	return baseAddress
}

func (a *BaseAddress) String() string {
	return a.Host + ":" + strconv.Itoa(a.Port)
}

func (a *BaseAddress) Set(s string) error {
	hp := strings.Split(s, ":")
	if len(hp) != 3 {
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
