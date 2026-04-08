package bootstrap

import "github.com/golang-migrate/migrate/v4"

func RunMigration(connString string) error {
	m, err := migrate.New("file://db/migrations", connString)
	if err != nil {
		return err
	}
	return m.Up()

}
