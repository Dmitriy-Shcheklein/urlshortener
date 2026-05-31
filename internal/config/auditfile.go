package config

import (
	"flag"
	"log"
	"os"
)

type AuditFilePath struct {
	Path      string
	IsFromEnv bool
}

func NewAuditFilePath() *AuditFilePath {
	path := &AuditFilePath{Path: "default_audit"}

	flag.Var(path, "audit-file", "audit file path")

	return path
}

func (f *AuditFilePath) ApplyEnv() {
	path, ok := os.LookupEnv("AUDIT_FILE")
	if !ok {
		return
	}
	if err := f.Set(path); err != nil {
		log.Fatalf("error while set AUDIT_FILE env: %s", err)
	}
}

func (f *AuditFilePath) String() string {
	return f.Path
}

func (f *AuditFilePath) Set(s string) error {
	if f.IsFromEnv {
		return nil
	}

	f.Path = s
	return nil
}
