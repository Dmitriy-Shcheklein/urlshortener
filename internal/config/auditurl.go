package config

import (
	"errors"
	"flag"
	"log"
	"os"
	"strconv"
	"strings"
)

type AuditUrl struct {
	Host string
	Port int
}

func NewAuditUrl() *AuditUrl {
	netAddress := &AuditUrl{}

	flag.Var(netAddress, "audit-url", "audit url path")

	return netAddress
}

func (a *AuditUrl) ApplyEnv() {
	serverAddress, ok := os.LookupEnv("AUDIT_URL")
	if !ok {
		return
	}
	if err := a.Set(serverAddress); err != nil {
		log.Fatalf("error while set AUDIT_URL env: %s", err)
	}
}

func (a *AuditUrl) String() string {
	return a.Host + ":" + strconv.Itoa(a.Port)
}

func (a *AuditUrl) Set(s string) error {
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
