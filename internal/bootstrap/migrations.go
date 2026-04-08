package bootstrap

import (
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func RunMigration(connString string) error {
	m, err := migrate.New("file://./migrations", connString)
	if err != nil {
		return err
	}
	return m.Up()

}
