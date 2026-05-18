package database

import (
	"errors"
	"log"

	"github.com/golang-migrate/migrate/v4"

	// These blank imports are CRITICAL. They load the SQL file driver
	// and the PostgreSQL driver behind the scenes so the migration tool
	// knows how to read your files and speak to Postgres.
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations connects to your database using the connection string (databaseURL)
// and automatically executes any new .sql files found in the migrations folder.
func RunMigrations(databaseURL string) {
	log.Println("Starting database migration process...")

	// 1. Tell golang-migrate where your files are and give it the DB connection details
	// "file://migrations" points to the 'migrations' folder at the root of your project
	m, err := migrate.New("file://migrations", databaseURL)
	if err != nil {
		log.Fatalf("❌ CRITICAL: Could not create migration engine instance: %v", err)
	}

	// 2. Execute the migrations "Up" (creates your tables)
	err = m.Up()
	if err != nil {
		// If the database tables are already built, golang-migrate will return an error
		// called 'ErrNoChange'. That's perfectly fine! It just means everything is up to date.
		if errors.Is(err, migrate.ErrNoChange) {
			log.Println("ℹ️ Database schema is already up to date. No new migrations to apply.")
			return
		}

		// If it's any other error (like a syntax mistake in your SQL file), stop the app completely!
		log.Fatalf("❌ CRITICAL: Could not run up migrations successfully: %v", err)
	}

	log.Println("✅ Success: All database migrations applied successfully! Your tables are live.")
}
