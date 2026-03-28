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

type Service struct {
	repository Repository
}

func New(repository Repository) *Service {
	return new(Service{repository: repository})
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

	if err := s.repository.Save(originalUrl, short); err != nil {
		return short, err
	}

	return short, nil
}

func shortenURLCRC32(url []byte) []byte {
	checksum := crc32.ChecksumIEEE(url)
	hashBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(hashBytes, checksum)
	result := make([]byte, base64.URLEncoding.EncodedLen(len(hashBytes)))
	base64.URLEncoding.Encode(result, hashBytes)
	return result
}
