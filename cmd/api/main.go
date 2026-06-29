package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"x``
	"os"
	"time"

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
	// EXPOSE PPROF PROFILING GATEWAY
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

	// 2. Establish PostgreSQL Connection Pool
	var db *sql.DB
	dsn := cfg.GetDSN()

	for i := 1; i <= 5; i++ {
		log.Printf("Connecting to PostgreSQL (Attempt %d/5)...", i)
		db, err = sql.Open("postgres", dsn)
		if err == nil {
			err = db.Ping()
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

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	// 3. Run Database Migrations Automatically
	log.Println("Preparing database migrations...")
	migrationURL := "postgres://" + cfg.DBUser + ":" + cfg.DBPassword + "@" + cfg.DBHost + ":" + cfg.DBPort + "/" + cfg.DBName + "?sslmode=disable"
	database.RunMigrations(migrationURL)

	// 4. Establish Redis Connection
	log.Println("Connecting to Redis Cache Layer...")
	// 4. Establish Redis Connection
	log.Println("Connecting to Redis Cache Layer...")
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisHost + ":" + cfg.RedisPort,
		Password: os.Getenv("REDIS_PASSWORD"), // 🔥 FORCE it to read the raw variable directly!
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("CRITICAL: Failed to connect to Redis cache: %v", err)
	}
	log.Println("🟢 Redis Connection: SUCCESSFUL")

	// ========================================================================
	// 🚀 ROUTER & HTTP ENGINE CORE
	// ========================================================================
	log.Println("Configuring routing architecture and middleware pipelines...")

	// 1. Initialize All Infrastructure Repositories
	userRepo := repository.NewUserRepository(db)
	keyRepo := repository.NewKeyRepository(db)
	paymentRepo := repository.NewPaymentRepository(db)
	auditRepo := repository.NewAuditRepository(db)

	// 2. Initialize Core Domain Business Services
	authServ := service.NewAuthService(keyRepo)
	paymentServ := service.NewPaymentService(paymentRepo)
	subServ := service.NewSubscriptionService()
	disputeServ := service.NewDisputeService()

	jwtServ, err := service.NewJWTAuthService(rdb, "private_key.pem", "public_key.pem")
	if err != nil {
		log.Fatalf("CRITICAL: Failed to seed RSA Cryptographic Keys: %v", err)
	}

	// 3. Initialize REST Presentation Handlers
	authHand := handler.NewAuthHandler(authServ)
	paymentHand := handler.NewPaymentHandler(paymentServ)
	subHand := handler.NewSubscriptionHandler(subServ)
	disputeHand := handler.NewDisputeHandler(disputeServ)
	jwtHand := handler.NewJWTAuthHandler(jwtServ)

	// 4. Setup Idempotency & Rate Middleware Engines
	idemMW := customMW.NewIdempotencyMiddleware(rdb)
	rateMW := customMW.NewRateLimiterMiddleware(rdb)

	// Inject a standby test merchant on startup for easy routing tests
	_, _ = userRepo.CreateUser("merchant_india@gmail.com", 1000000)

	// 5. Instantiate the Chi Router Engine
	r := chi.NewRouter()

	// THE CORS FIX: Explicitly grant permission to your Swagger dashboard container
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:8081")
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	// 6. Attach Core Middleware Checkpoints
	r.Use(middleware.RequestID)
	r.Use(customMW.MetricsTracker)
	r.Use(customMW.StructuredLogger)
	r.Use(middleware.Recoverer)

	// INITIALIZE REDIS ASYNQ BACKGROUND WORKER
	// INITIALIZE REDIS ASYNQ BACKGROUND WORKER
	asynqServer := asynq.NewServer(
		asynq.RedisClientOpt{
			Addr:     cfg.RedisHost + ":" + cfg.RedisPort,
			Password: os.Getenv("REDIS_PASSWORD"), // 🔥 Pass the direct environment password
		},
		asynq.Config{
			Concurrency: 5,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
			},
		},
	)

	advancedWorker := service.NewAdvancedWebhookWorker(db, paymentRepo)
	subService := service.NewSubscriptionService()
	subHandler := handler.NewSubscriptionHandler(subService)
	mux := asynq.NewServeMux()
	mux.HandleFunc(tasks.TypeWebhookRetryEvent, advancedWorker.ProcessWebhookTask)

	go func() {
		if err := asynqServer.Run(mux); err != nil {
			log.Fatalf("Critical loss of background worker engine execution loop: %v", err)
		}
	}()

	// EXPOSE OPEN PROMETHEUS METRICS SCRAPE ENDPOINT
	r.Handle("/metrics", promhttp.Handler())
	// 👇 ADD THIS EXACT LINE RIGHT BELOW IT:
	r.HandleFunc("/v1/healthcheck", healthcheckHandler)
	r.HandleFunc("POST /v1/subscriptions", subHandler.CreateSubscriptionHandler)

	// 7. Map REST API Endpoint Resource Routes
	r.Route("/v1", func(r chi.Router) {
		r.Use(rateMW.Limit)
		r.Post("/auth/token", jwtHand.IssueTokenHandler)
		r.Post("/auth/revoke", jwtHand.RevokeTokenHandler)

		r.Post("/auth/keys", authHand.CreateAPIKeyHandler)
		r.With(idemMW.Handle).Post("/charges", paymentHand.CreateChargeHandler)
		r.Post("/refunds", paymentHand.CreateRefundHandler)
		r.Post("/subscriptions", subHand.CreateSubscriptionHandler)
		r.Post("/disputes", disputeHand.CreateDisputeHandler)
		r.Use(handler.CORSSecurityMiddleware) // Enforces CORS filters first
		r.Use(handler.RateLimitMiddleware)    // Then enforces the Token Bucket rate limit

		// Day 23 Event Sourcing Audit History Track mapping
		r.Get("/payments/{id}/history", func(w http.ResponseWriter, r *http.Request) {
			paymentID := chi.URLParam(r, "id")
			analysis, err := auditRepo.AnalyzeTransitions(paymentID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(analysis)
		})
	})

	// PRODUCTION DATA SEEDING FOR K6 STRESS TESTS
	_, err = userRepo.CreateUser("merchant_india@gmail.com", 10000000)
	if err != nil {
		log.Printf("⚠️  Seeding Note: Test merchant user might already exist in tables: %v", err)
	} else {
		log.Println("✅ Production Test Merchant 'merchant_india@gmail.com' successfully seeded into PostgreSQL!")
	}
	protectedHandler := handler.RateLimitMiddleware(r)
	securePipeline := handler.CORSSecurityMiddleware(protectedHandler)

	// 8. Fire up the actual HTTP web server listener
	log.Println("VaultPay API Engine Gateway launching on port :8080...")
err := http.ListenAndServe(":8080", securePipeline)
if err != nil {
    log.Fatalf("Critical boot pipeline crash: %v", err)
}
}

// Place this at the absolute bottom of the file (outside of any other functions)
func healthcheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{
		"status": "available",
		"system_info": {
			"environment": "production",
			"version": "1.0.0"
		}
	}`))
}
