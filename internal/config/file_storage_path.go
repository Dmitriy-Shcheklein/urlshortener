package config

import (
	"flag"
	"log"
	"os"
)

type FileStoragePath struct {
	Path      string
	IsFromEnv bool
}

func NewFileStoragePath() *FileStoragePath {
	path := &FileStoragePath{Path: "default"}

	_ = flag.Value(path)
	flag.Var(path, "f", "file storage path")

	return path
}

func (f *FileStoragePath) ApplyEnv() {
	if path := os.Getenv("FILE_STORAGE_PATH"); path != "" {
		if err := f.Set(path); err != nil {
			log.Fatalf("error while set FILE_STORAGE_PATH env: %s", err)
		}
	}
}

func (f *FileStoragePath) String() string {
	return f.Path
}

func (f *FileStoragePath) Set(s string) error {
	if f.IsFromEnv == true {
		return nil
	}

	f.Path = s
	return nil
}
