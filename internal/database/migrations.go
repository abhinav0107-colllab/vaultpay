package database

import (
	"errors"
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations connects to your root migrations folder and applies updates safely
func RunMigrations(migrationURL string) {
	log.Println("🔄 Executing automated database migrations pipeline...")

	// Point directly to your root level migrations directory path using file:// prefix
	m, err := migrate.New("file://migrations", migrationURL)
	if err != nil {
		log.Fatalf("CRITICAL: Failed to initialize migration driver engine: %v", err)
	}

	// Apply all outstanding migration changes onto disk
	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Println("🟢 Database schema is already up-to-date! No changes required.")
		} else {
			log.Fatalf("CRITICAL: Database migration processing execution failed: %v", err)
		}
	} else {
		log.Println("🎉 Database migrations completed successfully! All tables ready.")
	}
}
