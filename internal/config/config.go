package config

import (
	"flag"
	"strconv"
)

type Config struct {
	Port        int
	Host        string
	BaseAddress []byte
}

func New() (*Config, error) {

	netAddress := new(NetAddress{Host: "localhost", Port: 8080})
	_ = flag.Value(netAddress)
	flag.Var(netAddress, "a", "Net address host:port")
	baseAddress := new(BaseAddress{Host: netAddress.Host, Port: netAddress.Port, Protocol: "http"})
	_ = flag.Value(baseAddress)
	flag.Var(baseAddress, "b", "Base address protocol://host:port")

	flag.Parse()

	return &Config{
		Host:        netAddress.Host,
		Port:        netAddress.Port,
		BaseAddress: []byte(baseAddress.Protocol + ":" + baseAddress.Host + ":" + strconv.Itoa(baseAddress.Port)),
	}, nil

}

func (c *Config) GetNetAddress() string {
	return c.Host + ":" + strconv.Itoa(c.Port)
}

func (c *Config) GetBaseAddress() []byte {
	return c.BaseAddress
}
