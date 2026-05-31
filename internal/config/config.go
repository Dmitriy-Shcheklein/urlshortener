package config

import (
	"flag"
	"strconv"
)

type Config struct {
	port            int
	host            string
	baseAddress     []byte
	fileStoragePath string
	dbDSN           string
	auditFilePath   string
	auditUrl        string
	salt            string
}

type FromEnv struct {
	ServerAddress []string `env:"SERVER_ADDRESS"`
	BaseURL       string   `env:"BASE_URL"`
}

func New() (*Config, error) {
	netAddress := NewNetAddress()
	baseAddress := NewBaseAddress()
	fileStoragePath := NewFileStoragePath()
	dsn := NewDSN()
	auditFile := NewAuditFilePath()
	auditUrl := NewAuditUrl()
	salt := NewSalt()
	flag.Parse()
	netAddress.ApplyEnv()
	baseAddress.ApplyEnv()
	fileStoragePath.ApplyEnv()
	dsn.ApplyEnv()
	auditFile.ApplyEnv()
	auditUrl.ApplyEnv()
	salt.ApplyEnv()

	cfg := Config{
		host:            netAddress.Host,
		port:            netAddress.Port,
		fileStoragePath: fileStoragePath.Path,
		auditFilePath:   auditFile.Path,
		dbDSN:           dsn.Value,
		auditUrl:        auditUrl.String(),
		salt:            salt.String(),
	}

	if baseAddress.IsFulfilled() {
		cfg.baseAddress = []byte(baseAddress.Protocol + ":" + baseAddress.Host + ":" + strconv.Itoa(baseAddress.Port))
	}

	return &cfg, nil
}

func (c *Config) GetNetAddress() string {
	return c.host + ":" + strconv.Itoa(c.port)
}

func (c *Config) GetBaseAddress() []byte {
	return c.baseAddress
}

func (c *Config) GetFSPath() string {
	return c.fileStoragePath
}

func (c *Config) GetDSN() string {
	return c.dbDSN
}

func (c *Config) GetAuditUrl() string {
	return c.auditUrl
}

func (c *Config) GetSalt() []byte {
	return []byte(c.salt)
}
