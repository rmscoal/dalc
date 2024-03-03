package postgres

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

type Postgres struct {
	DB *sql.DB
}

var (
	pg   *Postgres
	once sync.Once
)

func New(url string) *Postgres {
	if pg == nil {
		once.Do(func() {
			db, err := sql.Open("postgres", url)
			if err != nil {
				log.Fatalf("unable to connect to postgres: %s", err)
			}

			driver, err := postgres.WithInstance(db, &postgres.Config{
				StatementTimeout: 1 * time.Minute,
			})
			if err != nil {
				log.Fatalf("unable to create driver to migrate: %s", err)
			}

			m, err := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
			if err != nil {
				log.Fatalf("unable to initialize migrator: %s", err)
			}

			if err := m.Up(); err != nil {
				// If the error is ErrNoChange, we do not exit.
				if !errors.Is(err, migrate.ErrNoChange) {
					log.Fatalf("migration failed: %s", err)
				}
			}

			pg = &Postgres{db}
		})
	}
	return pg
}

func (p *Postgres) Shutdown(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return errors.New("unable to shutdown postgres")
	default:
		return p.DB.Close()
	}
}
