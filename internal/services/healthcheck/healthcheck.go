package healthcheck

import "errors"

//go:generate minimock -i Repository -o healthcheck_mock_test.go
type Repository interface {
	Ping() error
}

type Service struct {
	repository Repository
}

func New(repository Repository) (*Service, error) {
	service := &Service{}
	if repository == nil {
		return service, errors.New("repository must be not nil")
	}
	service.repository = repository
	return service, nil
}

func (s *Service) PingDB() error {
	return s.repository.Ping()
}
