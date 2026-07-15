// Package shortener provides the core URL shortening business logic.
package shortener

import (
	"encoding/base64"
	"encoding/binary"
	"hash/crc32"

	"github.com/Dmitriy-Shcheklein/urlshortener/internal/model"
)

// LinkRepository defines the persistence layer interface for storing and
// retrieving shortened URLs. Implementations include PostgreSQL and filesystem backends.
type LinkRepository interface {
	// GetByID retrieves the original URL by its short identifier.
	GetByID(ID string) ([]byte, error)
	// Save persists a new short URL mapping. Returns a ConflictError if the
	// original URL already exists.
	Save(path []byte, short []byte, userID []byte) error
	// SaveMany persists multiple short URL mappings in a single batch operation.
	SaveMany(values []model.LinkRow, userID []byte) error
	// FindByUserID returns all non-deleted URLs owned by the given user.
	FindByUserID(userID []byte) ([]model.LinkRow, error)
	// Delete marks the specified links as deleted for the given user.
	Delete(in []*model.LinkToDelete) error
}

// Service implements the URL shortening business logic.
// It generates short identifiers using CRC32 hashing and delegates
// persistence to the configured LinkRepository.
type Service struct {
	linkRepository LinkRepository
}

// New creates a new shortener Service backed by the given repository.
func New(repository LinkRepository) *Service {
	return &Service{linkRepository: repository}
}

// GetByID retrieves the original URL by its short identifier.
// Returns pgx.ErrNoRows if the link does not exist or has been deleted.
func (s *Service) GetByID(id string) ([]byte, error) {
	link, err := s.linkRepository.GetByID(id)
	if err != nil {
		return nil, err
	}
	return link, nil
}

// CreateShort generates a short URL identifier for the given original URL
// using CRC32 hashing and persists the mapping. Returns a [postgres.ConflictError]
// if the URL was already shortened.
func (s *Service) CreateShort(originalURL []byte, userID []byte) ([]byte, error) {
	short := shortenURLCRC32(originalURL)

	if err := s.linkRepository.Save(originalURL, short, userID); err != nil {
		return nil, err
	}

	return short, nil
}

// CreateMany performs a batch URL shortening operation, generating short
// identifiers for each item and persisting them in a single batch insert.
func (s *Service) CreateMany(values []model.CreateManyBodyRaw, userID []byte) (
	[]model.CreateManyResponseRaw, error,
) {
	shorts := make([]model.CreateManyResponseRaw, len(values))
	for i := range values {
		shortValue := shortenURLCRC32([]byte(values[i].OriginalURL))
		shorts[i].ShortURL = string(shortValue)
		shorts[i].CorrelationID = values[i].CorrelationID
	}

	payload := make([]model.LinkRow, len(values))
	for i := range values {
		payload[i].OriginalURL = values[i].OriginalURL
		payload[i].ShortURL = shorts[i].ShortURL
	}

	err := s.linkRepository.SaveMany(payload, userID)
	return shorts, err
}

// FindByUserID returns all non-deleted shortened URLs owned by the given user.
func (s *Service) FindByUserID(userID []byte) ([]model.LinkRow, error) {
	return s.linkRepository.FindByUserID(userID)
}

// Delete marks the specified links as deleted, verifying user ownership.
func (s *Service) Delete(in []*model.LinkToDelete) error {
	return s.linkRepository.Delete(in)
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
