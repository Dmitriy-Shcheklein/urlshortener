package service

import (
	"encoding/base64"
	"encoding/binary"
	"hash/crc32"
)

type Repository interface {
	GetById(ID string) ([]byte, error)
	Save(path []byte, short []byte) error
}

type Config interface {
	GetBaseAddress() []byte
}
type Service struct {
	repository Repository
	config     Config
}

func New(repository Repository, config Config) *Service {
	return new(Service{repository: repository, config: config})
}

func (s *Service) GetById(ID string) ([]byte, error) {
	link, err := s.repository.GetById(ID)
	if err != nil {
		return link, err
	}
	return link, nil
}

func (s *Service) CreateShort(originalUrl []byte) ([]byte, error) {
	short := shortenURLCRC32(originalUrl)

	var result []byte
	result = append(s.config.GetBaseAddress(), "/"...)
	result = append(result, short...)

	if err := s.repository.Save(originalUrl, short); err != nil {
		return short, err
	}

	return result, nil
}

func shortenURLCRC32(url []byte) []byte {
	checksum := crc32.ChecksumIEEE(url)
	hashBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(hashBytes, checksum)
	result := make([]byte, base64.URLEncoding.EncodedLen(len(hashBytes)))
	base64.URLEncoding.Encode(result, hashBytes)
	return result
}
