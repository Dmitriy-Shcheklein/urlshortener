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

	netAddress := &NetAddress{Host: "localhost", Port: 8080}

	if serverAddress := os.Getenv("SERVER_ADDRESS"); serverAddress != "" {
		if err := netAddress.Set(serverAddress); err != nil {
			log.Fatalf("error while set SERVER_ADDRESS env: %s", err)
		}
		return netAddress
	}

	_ = flag.Value(netAddress)
	flag.Var(netAddress, "a", "Net address host:port")

	return netAddress
}

func (a *NetAddress) String() string {
	return a.Host + ":" + strconv.Itoa(a.Port)
}

func (a *NetAddress) Set(s string) error {
	hp := strings.Split(s, ":")
	if len(hp) != 2 {
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
