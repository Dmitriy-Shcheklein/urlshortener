package config

import (
	"flag"
	"log"
	"os"
)

type FileStoragePath struct {
	Path string
}

func NewFileStoragePath() *FileStoragePath {
	path := &FileStoragePath{Path: "default"}

	flag.Var(path, "f", "file storage path")

	return path
}

func (f *FileStoragePath) ApplyEnv() {
	path, ok := os.LookupEnv("FILE_STORAGE_PATH")
	if !ok {
		return
	}
	if err := f.Set(path); err != nil {
		log.Fatalf("error while set FILE_STORAGE_PATH env: %s", err)
	}
}

func (f *FileStoragePath) String() string {
	return f.Path
}

func (f *FileStoragePath) Set(s string) error {
	f.Path = s
	return nil
}
