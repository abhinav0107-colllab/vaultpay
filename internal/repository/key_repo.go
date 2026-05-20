package repository

import (
	"database/sql"
	"fmt"
)

type APIKey struct {
	ID      string
	UserID  string
	KeyHash string
	Label   string
}

type KeyRepository struct {
	db *sql.DB
}

func NewKeyRepository(db *sql.DB) *KeyRepository {
	return &KeyRepository{db: db}
}

// InsertKey saves the encrypted fingerprint of the key into our relational ledger
func (r *KeyRepository) InsertKey(userID, hash, label string) error {
	query := `INSERT INTO api_keys (user_id, key_hash, label) VALUES ($1, $2, $3)`
	_, err := r.db.Exec(query, userID, hash, label)
	if err != nil {
		return fmt.Errorf("failed to write key hash to repository: %w", err)
	}
	return nil
}
