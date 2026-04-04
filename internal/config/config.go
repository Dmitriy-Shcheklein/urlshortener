package config

import (
	"flag"
	"strconv"
)

type Config struct {
	Port            int
	Host            string
	BaseAddress     []byte
	FileStoragePath string
}

type FromEnv struct {
	ServerAddress []string `env:"SERVER_ADDRESS"`
	BaseURL       string   `env:"BASE_URL"`
}

func New() (*Config, error) {
	netAddress := NewNetAddress()
	baseAddress := NewBaseAddress()
	fileStoragePath := NewFileStoragePath()
	flag.Parse()
	netAddress.ApplyEnv()
	baseAddress.ApplyEnv()
	fileStoragePath.ApplyEnv()

	cfg := Config{
		Host:            netAddress.Host,
		Port:            netAddress.Port,
		FileStoragePath: fileStoragePath.Path,
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
