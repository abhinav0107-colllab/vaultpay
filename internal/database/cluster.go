package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/abhinav0107-collab/vaultpay/internal/config"
	_ "github.com/lib/pq"
)

type DatabaseCluster struct {
	Master  *sql.DB
	Replica *sql.DB
}

// InitCluster initializes connection pools for both writing and reading nodes
func InitCluster(cfg *config.Config) (*DatabaseCluster, error) {
	// 1. Establish Master Connection
	masterDSN := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	
	masterPool, err := sql.Open("postgres", masterDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open master DB: %w", err)
	}
	configurePool(masterPool)

	// 2. Establish Replica Connection
	replicaDSN := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBReplicaHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	
	replicaPool, err := sql.Open("postgres", replicaDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open replica DB: %w", err)
	}
	configurePool(replicaPool)

	// 3. Verify health
	if err := masterPool.Ping(); err != nil {
		return nil, fmt.Errorf("master node unreachable: %w", err)
	}
	if err := replicaPool.Ping(); err != nil {
		return nil, fmt.Errorf("replica node unreachable: %w", err)
	}

	return &DatabaseCluster{
		Master:  masterPool,
		Replica: replicaPool,
	}, nil
}

func configurePool(db *sql.DB) {
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)
}