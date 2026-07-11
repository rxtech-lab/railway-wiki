// Package database wires GORM to a TursoDB (libSQL) / SQLite backend.
package database

import (
	"fmt"
	"log"
	"strings"

	"github.com/glebarez/sqlite"
	_ "github.com/tursodatabase/libsql-client-go/libsql" // registers the pure-Go "libsql" database/sql driver
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/rxtech-lab/railway-wiki/internal/models"
)

// NewConnection opens a GORM connection.
//
// Remote Turso URLs (libsql://…?authToken=…, or ws/wss/http/https) are served by
// the pure-Go libSQL driver. Anything else (file:…, :memory:, plain paths) uses
// the embedded modernc SQLite driver, which is convenient for local dev and tests.
func NewConnection(url string) (*gorm.DB, error) {
	dialector := dialectorFor(url)

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// SQLite/libSQL is single-writer; keep the pool tight to avoid "database is
	// locked" errors under concurrent writes.
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	sqlDB.SetMaxOpenConns(1)

	if err := Migrate(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

// dialectorFor picks the right driver based on the URL scheme.
func dialectorFor(url string) gorm.Dialector {
	if isRemoteLibSQL(url) {
		return sqlite.Dialector{DriverName: "libsql", DSN: url}
	}
	return sqlite.Open(url)
}

func isRemoteLibSQL(url string) bool {
	for _, p := range []string{"libsql://", "wss://", "ws://", "https://", "http://"} {
		if strings.HasPrefix(url, p) {
			return true
		}
	}
	return false
}

// Migrate runs GORM AutoMigrate for every model.
func Migrate(db *gorm.DB) error {
	log.Println("Running database migrations...")
	if err := db.AutoMigrate(models.AllModels...); err != nil {
		return fmt.Errorf("failed to auto migrate: %w", err)
	}
	log.Println("Database migrations completed successfully")
	return nil
}
