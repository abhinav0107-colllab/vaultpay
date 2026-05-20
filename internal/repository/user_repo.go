package repository

import (
	"database/sql"
	"fmt"
)

type User struct {
	ID       string
	Email    string
	Balance  int64
	Currency string
}

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// CreateUser inserts a fresh merchant client into our database
func (r *UserRepository) CreateUser(email string, initialBalance int64) (*User, error) {
	query := `
		INSERT INTO users (email, balance, currency) 
		VALUES ($1, $2, 'INR') 
		RETURNING id, email, balance, currency`

	user := &User{}
	err := r.db.QueryRow(query, email, initialBalance).Scan(&user.ID, &user.Email, &user.Balance, &user.Currency)
	if err != nil {
		return nil, fmt.Errorf("failed to execute insert user statement: %w", err)
	}
	return user, nil
}
