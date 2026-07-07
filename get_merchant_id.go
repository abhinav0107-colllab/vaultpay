package main

import (
	"context"
	"fmt"
	"log"

	"github.com/abhinav0107-collab/vaultpay/internal/config"
	"github.com/abhinav0107-collab/vaultpay/internal/database"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Config load failed: %v", err)
	}

	dbCluster, err := database.InitCluster(cfg)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer dbCluster.Master.Close()

	query := `
		SELECT enumlabel 
		FROM pg_enum 
		JOIN pg_type ON pg_enum.enumtypid = pg_type.oid 
		WHERE pg_type.typname = 'payment_status';`

	rows, err := dbCluster.Master.QueryContext(context.Background(), query)
	if err != nil {
		log.Fatalf("Failed to query enum values: %v", err)
	}
	defer rows.Close()

	fmt.Println("\n📋 ALLOWED PAYMENT STATUS ENUM VALUES IN DATABASE:")
	for rows.Next() {
		var label string
		if err := rows.Scan(&label); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("- '%s'\n", label)
	}
	fmt.Println()
}
