package db

import (
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v4/stdlib"
	"go.uber.org/zap"
)

type Migrator struct {
	log            *zap.Logger
	dsn            string
	migrationsPath string
}

func NewMigrator(dsn, path string, log *zap.Logger) *Migrator {
	return &Migrator{dsn: dsn, migrationsPath: path, log: log}
}

func (m *Migrator) Up() error {
	db, err := sql.Open("pgx", m.dsn)
	if err != nil {
		return fmt.Errorf("failed to open sql.DB for migrations: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping DB for migrations: %w", err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create postgres migrate driver: %w", err)
	}

	mg, err := migrate.NewWithDatabaseInstance("file://"+m.migrationsPath, "postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to initialize migrate: %w", err)
	}

	if err := mg.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate up failed: %w", err)
	}

	if err == migrate.ErrNoChange {
		m.log.Info("No new migrations to apply")
	} else {
		m.log.Info("Database migrated successfully")
	}

	return nil
}
