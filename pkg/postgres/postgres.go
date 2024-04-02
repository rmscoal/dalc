package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/lib/pq"
	"github.com/rmscoal/dalc/config"
	"github.com/rmscoal/dalc/migrations"
)

type Postgres struct {
	DB *sql.DB
}

var (
	pg   *Postgres
	once sync.Once
)

func New(cfg config.Database) *Postgres {
	if pg == nil {
		once.Do(func() {
			pgurl, err := url.Parse(fmt.Sprintf("postgres://%s/%s", cfg.Host, cfg.DBName))
			if err != nil {
				log.Fatal("unable to parse base dsn", "err", err)
			}
			pgurl.User = url.UserPassword(cfg.Username, cfg.Password)
			// Add sslmode query
			query := pgurl.Query()
			query.Set("sslmode", cfg.SSLMode)
			pgurl.RawQuery = query.Encode()

			db, err := sql.Open("postgres", pgurl.String())
			if err != nil {
				log.Fatal("unable to connect to postgres", "url", pgurl.String(), "err", err)
			}

			driver, err := postgres.WithInstance(db, &postgres.Config{
				StatementTimeout: 1 * time.Minute,
			})
			if err != nil {
				log.Fatalf("unable to create driver to migrate: %s", err)
			}

			d, err := iofs.New(migrations.SQLs, ".")
			if err != nil {
				log.Fatal(err)
			}

			m, err := migrate.NewWithInstance("iofs", d, "postgres", driver)
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
