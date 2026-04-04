package service

import (
	"encoding/base64"
	"encoding/binary"
	"hash/crc32"
)

type Repository interface {
	GetByID(ID string) ([]byte, error)
	Save(path []byte, short []byte) error
}

type Service struct {
	repository Repository
}

func New(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) GetByID(id string) ([]byte, error) {
	link, err := s.repository.GetByID(id)
	if err != nil {
		return link, err
	}
	return link, nil
}

func (s *Service) CreateShort(originalURL []byte) ([]byte, error) {
	short := shortenURLCRC32(originalURL)

	if err := s.repository.Save(originalURL, short); err != nil {
		return short, err
	}

	return short, nil
}

func shortenURLCRC32(url []byte) []byte {
	checksum := crc32.ChecksumIEEE(url)
	const byteSize = 4
	hashBytes := make([]byte, byteSize)
	binary.BigEndian.PutUint32(hashBytes, checksum)
	result := make([]byte, base64.URLEncoding.EncodedLen(len(hashBytes)))
	base64.URLEncoding.Encode(result, hashBytes)
	return result
}
