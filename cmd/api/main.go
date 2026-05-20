package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/abhinav0107-collab/vaultpay/internal/config"
	"github.com/abhinav0107-collab/vaultpay/internal/database"
	"github.com/abhinav0107-collab/vaultpay/internal/handler"
	customMW "github.com/abhinav0107-collab/vaultpay/internal/middleware"
	"github.com/abhinav0107-collab/vaultpay/internal/repository"
	"github.com/abhinav0107-collab/vaultpay/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"
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
	// ... inside main.go, right before the trailing select {} block ...

	log.Println("🧪 Testing Day 3 Security Engines...")

	// 1. Initialize Repositories and Services
	userRepo := repository.NewUserRepository(db)
	keyRepo := repository.NewKeyRepository(db)
	authServ := service.NewAuthService(keyRepo)

	// 2. Create a Mock Merchant User
	testUser, err := userRepo.CreateUser("merchant_india@gmail.com", 500000) // 5000.00 INR balance
	if err != nil {
		log.Printf("⚠️ Test user creation skipped (likely already exists): %v", err)
	} else {
		log.Printf("👤 Created Test User Account: %s (ID: %s)", testUser.Email, testUser.ID)

		// 3. Generate a Secret Stripe-like API Key for this new user
		plainKey, err := authServ.GenerateAPIKey(testUser.ID, "Production Key")
		if err != nil {
			log.Fatalf("❌ Day 3 Test Failed: %v", err)
		}

		log.Println("================================================================")
		log.Printf("🔑 NEW PLAIN SECRET KEY GENERATED: %s", plainKey)
		log.Println("⚠️  Copy this key down! It will never be shown in plaintext again.")
		log.Println("================================================================")
	}

	// 5. Block the main process so the Docker container stays lit up green
	log.Println("🟢 Redis Connection: SUCCESSFUL")

	// ========================================================================
	// 🚀 DAY 4: INITIALIZE ROUTER & HTTP ENGINE CORE
	// ========================================================================
	log.Println("Configuring routing architecture and middleware pipelines...")

	// 1. Initialize repositories, services, and handlers
	userRepo = repository.NewUserRepository(db)
	keyRepo = repository.NewKeyRepository(db)
	authServ = service.NewAuthService(keyRepo)
	authHand := handler.NewAuthHandler(authServ)

	// Inject a standby test merchant on startup for easy routing tests
	_, _ = userRepo.CreateUser("merchant_india@gmail.com", 1000000)

	// 2. Instantiate the Chi Router Engine
	r := chi.NewRouter()

	// 3. Attach our core Middleware checkpoints
	r.Use(middleware.RequestID)      // Injects a unique tracking UUID per request
	r.Use(customMW.StructuredLogger) // Processes custom log statistics per request
	r.Use(middleware.Recoverer)      // Halts panics to maintain 100% server uptime

	// 4. Map our REST API Endpoints
	r.Route("/v1", func(r chi.Router) {
		r.Post("/auth/keys", authHand.CreateAPIKeyHandler)
	})

	// 5. Fire up the actual HTTP web server listener (This replaces select{})
	log.Println("🚀 VAULTPAY INTERNET GATEWAY IS ONLINE & ACTIVE ON PORT :8080")
	server := &http.Server{
		Addr:         ":8080",
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// This function stays open forever, processing requests and keeping the container green!
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("CRITICAL: Network gateway engine failure: %v", err)
	}
} // This closes func main()}
