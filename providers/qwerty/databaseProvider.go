package databaseProvider

import (
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type PostgresProvider struct {
	db *sqlx.DB
}

func NewDBProvider(connectionStr string) *PostgresProvider {
	db, err := sqlx.Connect("postgres", connectionStr)
	if err != nil {
		log.Fatalf("failed to connect to Postgres<>: %+v", err)
	}
	fmt.Println("Connected to PostgreSQL...")

	if err := migrateUp(db); err != nil {
		log.Fatalf("migration failed: %+v", err)
	}
	return &PostgresProvider{db: db}
}

func (p *PostgresProvider) DB() *sqlx.DB {
	return p.db
}

func (p *PostgresProvider) Close() error {
	return p.db.Close()
}

func migrateUp(db *sqlx.DB) error {
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance("file://database/migrations", "postgres", driver)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && err.Error() != "no change" {
		return err
	}

	fmt.Println("Migration complete.")
	return nil
}
