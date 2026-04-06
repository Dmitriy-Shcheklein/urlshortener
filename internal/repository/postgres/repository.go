package postgres

import "errors"

//go:generate minimock -i Pool -o repository_mock_test.go
type Pool interface {
	Ping() error
}

type Repository struct {
	pool Pool
}

func New(pool Pool) (*Repository, error) {
	repository := &Repository{}
	if pool == nil {
		return repository, errors.New("pool must be not nil")
	}
	repository.pool = pool
	return repository, nil
}

func (r *Repository) Ping() error {
	return r.pool.Ping()
}
