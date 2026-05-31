package config

import (
	"flag"
	"log"
	"os"
)

// Salt структура для хранения соли токена авторизации
type Salt struct {
	value string
}

// NewSalt конструктор для конфигурации соли токена
func NewSalt() *Salt {
	salt := &Salt{value: "secret_key"}
	flag.Var(salt, "s", "Token salt")
	return salt
}

// ApplyEnv метод для применения данных из .ENV
func (s *Salt) ApplyEnv() {
	dsn, ok := os.LookupEnv("TOKEN_SALT")
	if !ok || dsn == "" {
		return
	}
	if err := s.Set(dsn); err != nil {
		log.Fatalf("error while set TOKEN_SALT env: %s", err)
	}
}

// String реализация для соответствия интерфейсу flag.Var
func (s *Salt) String() string {
	return s.value
}

// Set реализация для соответствия интерфейсу flag.Var
func (s *Salt) Set(str string) error {
	s.value = str
	return nil
}
