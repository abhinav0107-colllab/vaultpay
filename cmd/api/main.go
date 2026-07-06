package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/hibiken/asynq"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/abhinav0107-collab/vaultpay/internal/config"
	"github.com/abhinav0107-collab/vaultpay/internal/database"
	"github.com/abhinav0107-collab/vaultpay/internal/handler"
	"github.com/abhinav0107-collab/vaultpay/internal/logger"
	customMW "github.com/abhinav0107-collab/vaultpay/internal/middleware"
	"github.com/abhinav0107-collab/vaultpay/internal/repository"
	"github.com/abhinav0107-collab/vaultpay/internal/service"
	"github.com/abhinav0107-collab/vaultpay/internal/tasks"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func main() {
	logger.InitializeLogger()
	defer logger.Log.Sync()
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

	// 2. Establish PostgreSQL Connection Pool (Master & Replica Cluster)
	logger.Log.Info("Connecting to Database Cluster (Master & Replica Nodes)...")

	dbCluster, err := database.InitCluster(cfg)
	if err != nil {
		logger.Log.Fatal("Critical boot failure: Database Cluster unreachable", zap.Error(err))
	}
	defer dbCluster.Master.Close()
	defer dbCluster.Replica.Close()

	logger.Log.Info("Database Cluster Status: MASTER (Connected) | REPLICA (Connected)")
	log.Println("🟢 PostgreSQL Connection: SUCCESSFUL")

	// 3. Run Database Migrations Automatically (Targeting Master)
	log.Println("Preparing database migrations...")
	migrationURL := "postgres://" + cfg.DBUser + ":" + cfg.DBPassword + "@" + cfg.DBHost + ":" + cfg.DBPort + "/" + cfg.DBName + "?sslmode=disable"
	database.RunMigrations(migrationURL)

	// 4. Establish Redis Connection
	log.Println("Connecting to Redis Cache Layer...")
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisHost + ":" + cfg.RedisPort,
		Password: os.Getenv("REDIS_PASSWORD"),
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
	userRepo := repository.NewUserRepository(dbCluster.Master)
	keyRepo := repository.NewKeyRepository(dbCluster.Master)
	paymentRepo := repository.NewPaymentRepository(dbCluster.Master)
	auditRepo := repository.NewAuditRepository(dbCluster.Replica)

	// 2. Initialize Core Domain Business Services
	authServ := service.NewAuthService(keyRepo)
	paymentServ := service.NewPaymentService(paymentRepo)
	disputeServ := service.NewDisputeService()

	jwtServ, err := service.NewJWTAuthService(rdb, "private_key.pem", "public_key.pem")
	if err != nil {
		logger.Log.Fatal("Critical boot failure: Failed to seed RSA Cryptographic Keys", zap.Error(err))
	}

	// 3. Initialize REST Presentation Handlers
	authHand := handler.NewAuthHandler(authServ)
	paymentHand := handler.NewPaymentHandler(paymentServ)
	subService := service.NewSubscriptionService()
	subHand := handler.NewSubscriptionHandler(subService)
	disputeHand := handler.NewDisputeHandler(disputeServ)
	jwtHand := handler.NewJWTAuthHandler(jwtServ)

	// 4. Setup Idempotency & Rate Middleware Engines
	idemMW := customMW.NewIdempotencyMiddleware(rdb)
	rateMW := customMW.NewRateLimiterMiddleware(rdb)

	// Inject a standby test merchant on startup for easy routing tests
	_, _ = userRepo.CreateUser("merchant_india@gmail.com", 1000000)

	// 5. Instantiate the Chi Router Engine
	r := chi.NewRouter()

	// --- GLOBAL MIDDLEWARES PLACED SECURELY AT THE VERY TOP ---
	r.Use(middleware.RequestID)
	r.Use(customMW.MetricsTracker)
	r.Use(customMW.StructuredLogger)
	r.Use(middleware.Recoverer)
	r.Use(handler.CORSSecurityMiddleware)
	r.Use(handler.RateLimitMiddleware)

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

	// 6. INITIALIZE REDIS ASYNQ BACKGROUND WORKER
	asynqServer := asynq.NewServer(
		asynq.RedisClientOpt{
			Addr:     cfg.RedisHost + ":" + cfg.RedisPort,
			Password: os.Getenv("REDIS_PASSWORD"),
		},
		asynq.Config{
			Concurrency: 5,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
			},
		},
	)

	advancedWorker := service.NewAdvancedWebhookWorker(dbCluster.Master, paymentRepo)
	mux := asynq.NewServeMux()
	mux.HandleFunc(tasks.TypeWebhookRetryEvent, advancedWorker.ProcessWebhookTask)

	go func() {
		if err := asynqServer.Run(mux); err != nil {
			log.Fatalf("Critical loss of background worker engine execution loop: %v", err)
		}
	}()

	// 7. Route Endpoint Registration
	r.Handle("/metrics", promhttp.Handler())
	r.HandleFunc("/v1/healthcheck", healthcheckHandler)
	r.HandleFunc("POST /v1/subscriptions", subHand.CreateSubscriptionHandler)

	// Map REST API Endpoint Resource Routes
	r.Route("/v1", func(r chi.Router) {
		r.Use(rateMW.Limit)
		r.Post("/auth/token", jwtHand.IssueTokenHandler)
		r.Post("/auth/revoke", jwtHand.RevokeTokenHandler)

		r.Post("/auth/keys", authHand.CreateAPIKeyHandler)
		r.With(idemMW.Handle).Post("/charges", paymentHand.CreateChargeHandler)
		r.Post("/refunds", paymentHand.CreateRefundHandler)
		r.Post("/subscriptions", subHand.CreateSubscriptionHandler)
		r.Post("/disputes", disputeHand.CreateDisputeHandler)

		// Day 23 Event Sourcing Audit History Track mapping (Using Replica Node!)
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

	// 8. Fire up the actual HTTP web server listener
	log.Println("VaultPay API Engine Gateway launching on port :8080...")
	err = http.ListenAndServe(":8080", r)
	if err != nil {
		log.Fatalf("Critical boot pipeline crash: %v", err)
	}
}

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
