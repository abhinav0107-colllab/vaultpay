package main

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/abhinav0107-collab/vaultpay/internal/config"
	"github.com/abhinav0107-collab/vaultpay/internal/database"

	_ "github.com/lib/pq" // Side-effect import for the PostgreSQL driver
	"github.com/redis/go-redis/v9"
)

func main() {
	log.Println("--- VaultPay Initializing ---")

	// 1. Load Configurations from Environment Variables
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to initialize configuration settings: %v", err)
	}

	// 2. Establish PostgreSQL Connection Pool (With a robust Retry Loop)
	var db *sql.DB
	dsn := cfg.GetDSN()

	for i := 1; i <= 5; i++ {
		log.Printf("Connecting to PostgreSQL (Attempt %d/5)...", i)
		db, err = sql.Open("postgres", dsn)
		if err == nil {
			err = db.Ping() // Verifies the connection is actually functional
		}

		if err == nil {
			break
		}
		log.Printf("PostgreSQL not ready yet, retrying in 2s: %v", err)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		log.Fatalf("CRITICAL: Failed to connect to PostgreSQL database: %v", err)
	}
	defer db.Close()
	log.Println("🟢 PostgreSQL Connection: SUCCESSFUL")

	// ==========================================
	// 🔥 NEW: 3. RUN DATABASE MIGRATIONS AUTOMATICALLY
	// ==========================================
	log.Println("Preparing database migrations...")

	// Construct a standard URL string that the golang-migrate library understands
	migrationURL := "postgres://" + cfg.DBUser + ":" + cfg.DBPassword + "@" + cfg.DBHost + ":" + cfg.DBPort + "/" + cfg.DBName + "?sslmode=disable"

	// Execute the function we built in internal/database/migrations.go
	database.RunMigrations(migrationURL)
	// ==========================================

	// 4. Establish Redis Connection
	log.Println("Connecting to Redis Cache Layer...")
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisHost + ":" + cfg.RedisPort,
	})

	// Use a context background timeline to ping Redis
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("CRITICAL: Failed to connect to Redis cache: %v", err)
	}
	log.Println("🟢 Redis Connection: SUCCESSFUL")

	log.Println("🚀 VAULTPAY ENGINE IS ONLINE & ACTIVE. All systems operating normally.")

	// 5. Block the main process so the Docker container stays lit up green
	select {}
}
