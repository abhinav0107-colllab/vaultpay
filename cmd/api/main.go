package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"

	// 🔥 ADD THIS PROMETHEUS LINE TO YOUR IMPORTS:
	"github.com/hibiken/asynq"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/abhinav0107-collab/vaultpay/internal/config"
	"github.com/abhinav0107-collab/vaultpay/internal/database"
	"github.com/abhinav0107-collab/vaultpay/internal/handler"
	customMW "github.com/abhinav0107-collab/vaultpay/internal/middleware"
	"github.com/abhinav0107-collab/vaultpay/internal/repository"
	"github.com/abhinav0107-collab/vaultpay/internal/service"
	"github.com/abhinav0107-collab/vaultpay/internal/tasks"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"

	"github.com/redis/go-redis/v9"
)

func main() {
	log.Println("--- VaultPay Initializing ---")
	// 🔥 DAY 19: EXPOSE PPROF PROFILING GATEWAY
	// Exposes internal execution metrics on port :6060 for live load debugging
	go func() {
		log.Println("⏱️  PPROF PROFILER SERVER LISTENING ON PORT :6060")
		if err := http.ListenAndServe("0.0.0.0:6060", nil); err != nil {
			log.Printf("Pprof backend server error: %v", err)
		}
	}()

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
	// 🔥 DAY 19 PERFORMANCE OPTIMIZATION: TUNE CONNECTION POOLS
	// This prevents the Go engine from overwhelming Postgres and throwing EOF panics
	db.SetMaxOpenConns(25)                 // Maximum number of open connections to the database
	db.SetMaxIdleConns(25)                 // Maximum number of connections in the idle connection pool
	db.SetConnMaxLifetime(5 * time.Minute) // Maximum amount of time a connection may be reused

	// 3. Run Database Migrations Automatically
	log.Println("Preparing database migrations...")
	migrationURL := "postgres://" + cfg.DBUser + ":" + cfg.DBPassword + "@" + cfg.DBHost + ":" + cfg.DBPort + "/" + cfg.DBName + "?sslmode=disable"
	database.RunMigrations(migrationURL)

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

	// ========================================================================
	// 🚀 ROUTER & HTTP ENGINE CORE (DAYS 4, 5 & 6)
	// ========================================================================
	log.Println("Configuring routing architecture and middleware pipelines...")

	// 1. Initialize All Infrastructure Repositories
	userRepo := repository.NewUserRepository(db)
	keyRepo := repository.NewKeyRepository(db)
	paymentRepo := repository.NewPaymentRepository(db)

	// 2. Initialize Core Domain Business Services
	authServ := service.NewAuthService(keyRepo)
	paymentServ := service.NewPaymentService(paymentRepo)

	// 3. Initialize REST Presentation Handlers
	authHand := handler.NewAuthHandler(authServ)
	paymentHand := handler.NewPaymentHandler(paymentServ)

	// 4. Setup Day 6 Idempotency Middleware Protection Engine
	idemMW := customMW.NewIdempotencyMiddleware(rdb)
	rateMW := customMW.NewRateLimiterMiddleware(rdb)

	// Inject a standby test merchant on startup for easy routing tests
	_, _ = userRepo.CreateUser("merchant_india@gmail.com", 1000000)

	// 5. Instantiate the Chi Router Engine
	r := chi.NewRouter()

	// 6. Attach Core Middleware Checkpoints
	r.Use(middleware.RequestID)      // Injects a unique tracking UUID per request
	r.Use(customMW.MetricsTracker)   // 🔥 Day 17: Track every single transaction metric
	r.Use(customMW.StructuredLogger) // Processes custom log statistics per request
	r.Use(middleware.Recoverer)      // Halts panics to maintain 100% server uptime

	// 🔥 DAY 16 (LEFT CARD): INITIALIZE REDIS ASYNQ BACKGROUND WORKER
	// Connects our distributed background server to our Redis container
	asynqServer := asynq.NewServer(
		asynq.RedisClientOpt{Addr: "redis:6379"},
		asynq.Config{
			Concurrency: 5, // Process up to 5 concurrent webhook retry events safely
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
			},
		},
	)

	advancedWorker := service.NewAdvancedWebhookWorker(db, paymentRepo)
	mux := asynq.NewServeMux()

	// Bind our distributed task pattern routing
	mux.HandleFunc(tasks.TypeWebhookRetryEvent, advancedWorker.ProcessWebhookTask)

	// Spin up our Asynq engine inside a dedicated background goroutine thread
	go func() {
		if err := asynqServer.Run(mux); err != nil {
			log.Fatalf("Critical loss of background worker engine execution loop: %v", err)
		}
	}()

	// 🔥 DAY 14: EXPOSE OPEN PROMETHEUS METRICS SCRAPE ENDPOINT
	// This lets Prometheus pull performance and latency counters from your system
	r.Handle("/metrics", promhttp.Handler())

	// 7. Map REST API Endpoint Resource Routes
	r.Route("/v1", func(r chi.Router) {
		// FORCE ALL ENDPOINTS TO PASS THROUGH THE ATOMIC REDIS LUA LIMITER FIRST
		r.Use(rateMW.Limit)

		r.Post("/auth/keys", authHand.CreateAPIKeyHandler)

		// Shields payment routes with defensive transactional idempotency
		r.With(idemMW.Handle).Post("/charges", paymentHand.CreateChargeHandler)

		r.Post("/refunds", paymentHand.CreateRefundHandler)
	})
	// 🔥 DAY 19: PRODUCTION DATA SEEDING FOR K6 STRESS TESTS
	// This ensures our load tests hit a valid record, bypassing database 404/422 crash barriers
	_, err = userRepo.CreateUser("merchant_india@gmail.com", 10000000) // 100,000.00 minor currency units
	if err != nil {
		log.Printf("⚠️  Seeding Note: Test merchant user might already exist in tables: %v", err)
	} else {
		log.Println("✅ Production Test Merchant 'merchant_india@gmail.com' successfully seeded into PostgreSQL!")
	}

	// 8. Fire up the actual HTTP web server listener
	log.Println("🚀 VAULTPAY INTERNET GATEWAY IS ONLINE & ACTIVE ON PORT :8080")
	server := &http.Server{
		Addr:         ":8080",
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("CRITICAL: Network gateway engine failure: %v", err)
	}
}
