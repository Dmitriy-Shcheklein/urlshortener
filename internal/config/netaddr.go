package config

import (
	"errors"
	"flag"
	"log"
	"os"
	"strconv"
	"strings"
)

type NetAddress struct {
	Host string
	Port int
}

func NewNetAddress() *NetAddress {
	port := 8080
	netAddress := &NetAddress{Host: "localhost", Port: port}

	flag.Var(netAddress, "a", "Net address host:port")

	return netAddress
}

func (a *NetAddress) ApplyEnv() {
	serverAddress, ok := os.LookupEnv("SERVER_ADDRESS")
	if !ok {
		return
	}
	if err := a.Set(serverAddress); err != nil {
		log.Fatalf("error while set SERVER_ADDRESS env: %s", err)
	}
}

func (a *NetAddress) String() string {
	return a.Host + ":" + strconv.Itoa(a.Port)
}

func (a *NetAddress) Set(s string) error {
	hp := strings.Split(s, ":")
	validLength := 2
	if len(hp) != validLength {
		return errors.New("need address in a form host:port")
	}
	port, err := strconv.Atoi(hp[1])
	if err != nil {
		return err
	}
	a.Host = hp[0]
	a.Port = port
	return nil
}
