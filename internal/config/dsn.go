package config

import (
	"flag"
	"log"
	"os"
)

type DSN struct {
	Value string
}

func NewDSN() *DSN {
	dsn := &DSN{}

	flag.Var(dsn, "d", "Database dsn")

	return dsn
}

func (a *DSN) ApplyEnv() {
	dsn, ok := os.LookupEnv("DATABASE_DSN")
	if !ok {
		return
	}
	if err := a.Set(dsn); err != nil {
		log.Fatalf("error while set DATABASE_DSN env: %s", err)
	}
}

func (a *DSN) String() string {
	return a.Value
}

func (a *DSN) Set(s string) error {
	a.Value = s
	return nil
}
