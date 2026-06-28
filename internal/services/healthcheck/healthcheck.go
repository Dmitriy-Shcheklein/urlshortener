// Package healthcheck provides a service for checking database connectivity.
package healthcheck

import "errors"

//go:generate minimock -i Repository -o healthcheck_mock_test.go

// Repository defines the persistence layer interface for health check operations.
type Repository interface {
	// Ping verifies connectivity to the underlying data store.
	Ping() error
}

// Service implements the health check business logic.
type Service struct {
	repository Repository
}

// New creates a new health check Service. The repository parameter must not be nil.
func New(repository Repository) (*Service, error) {
	service := &Service{}
	if repository == nil {
		return service, errors.New("repository must be not nil")
	}
	service.repository = repository
	return service, nil
}

// PingDB checks database connectivity by delegating to the repository.
func (s *Service) PingDB() error {
	return s.repository.Ping()
}
