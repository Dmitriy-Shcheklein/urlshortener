package config

import (
	"flag"
	"strconv"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/config/db/postgres"
)

type Config struct {
	Port            int
	Host            string
	BaseAddress     []byte
	FileStoragePath string
	DbDSN           DbDSN
}

type DbDSN struct {
	Value   string
	IsValid bool
}

type FromEnv struct {
	ServerAddress []string `env:"SERVER_ADDRESS"`
	BaseURL       string   `env:"BASE_URL"`
}

func New() (*Config, error) {
	netAddress := NewNetAddress()
	baseAddress := NewBaseAddress()
	fileStoragePath := NewFileStoragePath()
	dsn := postgres.NewDSN()
	flag.Parse()
	netAddress.ApplyEnv()
	baseAddress.ApplyEnv()
	fileStoragePath.ApplyEnv()
	dsn.ApplyEnv()

	cfg := Config{
		Host:            netAddress.Host,
		Port:            netAddress.Port,
		FileStoragePath: fileStoragePath.Path,
	}

	if dsn.Value != "" {
		cfg.DbDSN = DbDSN{Value: dsn.Value, IsValid: true}
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
