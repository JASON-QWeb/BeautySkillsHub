package database

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

type Migrator struct {
	databaseURL   string
	migrationsDir string
	sourceURL     string
}

func NewMigrator(databaseURL, migrationsDir string) (*Migrator, error) {
	if databaseURL == "" {
		return nil, errors.New("database url is required")
	}
	if migrationsDir == "" {
		return nil, errors.New("migrations dir is required")
	}

	absDir, err := filepath.Abs(migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("resolve migrations dir: %w", err)
	}

	return &Migrator{
		databaseURL:   databaseURL,
		migrationsDir: absDir,
		sourceURL:     "file://" + filepath.ToSlash(absDir),
	}, nil
}

func (m *Migrator) SourceURL() string {
	return m.sourceURL
}

func (m *Migrator) MigrationsDir() string {
	return m.migrationsDir
}

func (m *Migrator) Up() error {
	instance, err := migrate.New(m.sourceURL, m.databaseURL)
	if err != nil {
		return fmt.Errorf("create migrate instance: %w", err)
	}
	defer func() {
		_, _ = instance.Close()
	}()

	if err := instance.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run up migrations: %w", err)
	}

	return nil
}
