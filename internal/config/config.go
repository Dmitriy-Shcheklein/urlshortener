// Package config provides application configuration management.
// It reads settings from command-line flags and environment variables,
// with environment variables taking precedence.
package config

import (
	"flag"
	"strconv"
)

// Config holds all application configuration values.
// Use [New] to create an instance that loads settings from flags and environment variables.
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

// FromEnv represents environment variable bindings for configuration.
type FromEnv struct {
	ServerAddress []string `env:"SERVER_ADDRESS"`
	BaseURL       string   `env:"BASE_URL"`
}

// New creates a new Config by parsing command-line flags and applying
// environment variable overrides. Call flag.Parse() is called internally.
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

// GetNetAddress returns the server listen address in "host:port" format.
func (c *Config) GetNetAddress() string {
	return c.host + ":" + strconv.Itoa(c.port)
}

// GetBaseAddress returns the base URL used for building shortened URLs.
// Returns nil if not configured.
func (c *Config) GetBaseAddress() []byte {
	return c.baseAddress
}

// GetFSPath returns the filesystem path for file-based URL storage.
func (c *Config) GetFSPath() string {
	return c.fileStoragePath
}

// GetDSN returns the PostgreSQL connection string.
func (c *Config) GetDSN() string {
	return c.dbDSN
}

// GetAuditUrl returns the HTTP endpoint URL for audit event delivery.
func (c *Config) GetAuditUrl() string {
	return c.auditUrl
}

// GetAuditFilePath returns the file path for filesystem-based audit logging.
func (c *Config) GetAuditFilePath() string {
	return c.auditFilePath
}

// GetSalt returns the secret key used for JWT token signing.
func (c *Config) GetSalt() []byte {
	return []byte(c.salt)
}
