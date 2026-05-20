package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/abhinav0107-collab/vaultpay/internal/repository" // Adjust module prefix if needed
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	keyRepo *repository.KeyRepository
}

func NewAuthService(kr *repository.KeyRepository) *AuthService {
	return &AuthService{keyRepo: kr}
}

// GenerateAPIKey generates a cryptographically secure random key string, hashes it, and saves the hash
func (s *AuthService) GenerateAPIKey(userID, label string) (string, error) {
	// 1. Allocate a 16-byte memory block buffer
	bytes := make([]byte, 16)

	// 2. Fill the buffer with highly chaotic, hardware-generated random numbers (CSPRNG)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate secure random source bytes: %w", err)
	}

	// 3. Convert bytes into a clean, human-readable string and append prefix
	plainKey := "vpay_live_" + hex.EncodeToString(bytes)

	// 4. Securely mix and lock the key using bcrypt algorithm
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(plainKey), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to cryptographically protect plain key: %w", err)
	}
	keyHash := string(hashBytes)

	// 5. Save only the secure hash string into Postgres
	err = s.keyRepo.InsertKey(userID, keyHash, label)
	if err != nil {
		return "", fmt.Errorf("auth_service failed to persist key configuration: %w", err)
	}

	// Return the unhashed raw plain text key so we can show it to the developer ONCE
	return plainKey, nil
}
