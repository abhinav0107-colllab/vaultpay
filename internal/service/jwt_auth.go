package service

import (
	"context"
	"crypto/rsa"
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

var (
	ErrTokenRevoked = errors.New("authentication token has been explicitly revoked and blacklisted")
	ErrInvalidToken = errors.New("cryptographic signature mismatch or token expired")
)

type JWTAuthService struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	rdb        *redis.Client
}

func NewJWTAuthService(rdb *redis.Client, privateKeyPath, publicKeyPath string) (*JWTAuthService, error) {
	privBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, err
	}
	privKey, err := jwt.ParseRSAPrivateKeyFromPEM(privBytes)
	if err != nil {
		return nil, err
	}

	pubBytes, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, err
	}
	pubKey, err := jwt.ParseRSAPublicKeyFromPEM(pubBytes)
	if err != nil {
		return nil, err
	}

	return &JWTAuthService{
		privateKey: privKey,
		publicKey:  pubKey,
		rdb:        rdb,
	}, nil
}

// GenerateAccessToken signs an OAuth2-style access token using your RSA private key
func (s *JWTAuthService) GenerateAccessToken(userID, clientType string) (string, error) {
	claims := jwt.MapClaims{
		"sub":         userID,
		"client_type": clientType,
		"exp":         time.Now().Add(15 * time.Minute).Unix(),
		"iat":         time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(s.privateKey)
}

// RevokeToken flags a token string inside Redis to immediately block it from future requests
func (s *JWTAuthService) RevokeToken(ctx context.Context, tokenStr string, ttl time.Duration) error {
	return s.rdb.Set(ctx, "blacklist:"+tokenStr, "true", ttl).Err()
}

// ValidateAccessToken verifies the RSA public signature and cross-checks the Redis revocation state
func (s *JWTAuthService) ValidateAccessToken(ctx context.Context, tokenStr string) (jwt.MapClaims, error) {
	// 1. Immediately verify against real-time Redis blacklists
	blacklisted, _ := s.rdb.Get(ctx, "blacklist:"+tokenStr).Result()
	if blacklisted == "true" {
		return nil, ErrTokenRevoked
	}

	// 2. Validate cryptographic signature using your RSA Public Key
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, ErrInvalidToken
		}
		return s.publicKey, nil
	})

	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	return token.Claims.(jwt.MapClaims), nil
}
