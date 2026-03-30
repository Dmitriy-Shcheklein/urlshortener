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

type FromEnv struct {
	ServerAddress []string `env:"SERVER_ADDRESS"`
	BaseUrl       string   `env:"BASE_URL"`
}

func New() (*Config, error) {
	netAddress := NewNetAddress()
	baseAddress := NewBaseAddress()
	flag.Parse()
	netAddress.ApplyEnv()
	baseAddress.ApplyEnv()

	cfg := Config{
		Host: netAddress.Host,
		Port: netAddress.Port,
	}

	if baseAddress.IsFulfilled() {
		cfg.BaseAddress = []byte(baseAddress.Protocol + ":" + baseAddress.Host + ":" + strconv.Itoa(baseAddress.Port))
	}

	return &cfg, nil

}

func (c *Config) GetNetAddress() string {
	return c.Host + ":" + strconv.Itoa(c.Port)
}

func (c *Config) GetBaseAddress() []byte {
	return c.BaseAddress
}
