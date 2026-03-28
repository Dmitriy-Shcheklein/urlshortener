package config

import (
	"errors"
	"strconv"
	"strings"
)

type BaseAddress struct {
	Host     string
	Port     int
	Protocol string
}

func (a *BaseAddress) String() string {
	return a.Host + ":" + strconv.Itoa(a.Port)
}

func (a *BaseAddress) Set(s string) error {
	hp := strings.Split(s, ":")
	if len(hp) != 3 {
		return errors.New("need address in a form protocol://host:port")
	}
	port, err := strconv.Atoi(hp[2])
	if err != nil {
		return err
	}
	a.Protocol = hp[0]
	a.Host = hp[1]
	a.Port = port
	return nil
}
