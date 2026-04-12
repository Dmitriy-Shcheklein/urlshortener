package shortener

import (
	"encoding/base64"
	"encoding/binary"
	"hash/crc32"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/model"
)

type LinkRepository interface {
	GetByID(ID string) ([]byte, error)
	Save(path []byte, short []byte) error
	SaveMany(values []model.LinkRow) error
}

type Service struct {
	linkRepository LinkRepository
}

func New(repository LinkRepository) *Service {
	return &Service{linkRepository: repository}
}

func (s *Service) GetByID(id string) ([]byte, error) {
	link, err := s.linkRepository.GetByID(id)
	if err != nil {
		return link, err
	}
	return link, nil
}

func (s *Service) CreateShort(originalURL []byte) ([]byte, error) {
	short := shortenURLCRC32(originalURL)

	if err := s.linkRepository.Save(originalURL, short); err != nil {
		return short, err
	}

	return short, nil
}

func (s *Service) CreateMany(values []model.CreateManyBodyRaw) (
	[]model.CreateManyResponseRaw, error,
) {
	shorts := make([]model.CreateManyResponseRaw, len(values))
	for i := range values {
		shortValue := shortenURLCRC32([]byte(values[i].OriginalUrl))
		shorts[i].ShortURL = string(shortValue)
		shorts[i].CorrelationId = values[i].CorrelationID
	}

	payload := make([]model.LinkRow, len(values))
	for i := range values {
		payload[i].OriginalURL = values[i].OriginalUrl
		payload[i].ShortURL = shorts[i].ShortURL
	}

	err := s.linkRepository.SaveMany(payload)
	return shorts, err
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
