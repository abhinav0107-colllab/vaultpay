package config

import (
	"fmt"
	"os"
)

type Config struct {
	DBHost        string
	DBReplicaHost string // ◄ Added to track our read-only replica connection node
	DBPort        string
	DBUser        string
	DBPassword    string
	DBName        string
	RedisHost     string
	RedisPort     string
}

// Load reads values from the environment variables set by Docker Compose
// Load reads values from the environment variables set by Docker Compose
func Load() (*Config, error) {
	return &Config{
		DBHost:        getEnv("DB_HOST", "localhost"),
		DBReplicaHost: getEnv("DB_REPLICA_HOST", "localhost"), // ◄ Add this line cleanly here!
		DBPort:        getEnv("DB_PORT", "5432"),
		DBUser:        getEnv("DB_USER", "postgres"),
		DBPassword:    getEnv("DB_PASSWORD", "secretpassword"),
		DBName:        getEnv("DB_NAME", "vaultpay"),
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
	}, nil
}
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

// GetDSN formats the connection string for PostgreSQL
func (c *Config) GetDSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName)
}
