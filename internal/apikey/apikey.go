package apikey

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type APIKey struct {
	ID            string    `db:"id" json:"id"`
	ClientID      string    `db:"client_id" json:"client_id"`
	Name          *string   `db:"name" json:"name"`
	AccessKey     string    `db:"access_key" json:"access_key"`
	SecretKeyHash string    `db:"secret_key_hash" json:"-"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
}

type CreateAPIKeyInput struct {
	ClientID string  `json:"client_id"`
	Name     *string `json:"name"`
}

type CreateAPIKeyResult struct {
	APIKey    *APIKey `json:"api_key"`
	SecretKey string  `json:"secret_key"`
}

type UpdateAPIKeyInput struct {
	ID   string  `json:"id"`
	Name *string `json:"name"`
}

func GenerateRandomString(bytesLen int) (string, error) {
	b := make([]byte, bytesLen)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func GenerateKeys() (accessKey string, secretKey string, secretKeyHash string, err error) {
	akRand, err := GenerateRandomString(16)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to generate access key: %w", err)
	}
	accessKey = "ak_" + akRand

	skRand, err := GenerateRandomString(24)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to generate secret key: %w", err)
	}
	secretKey = "sk_" + skRand

	hashBytes, err := bcrypt.GenerateFromPassword([]byte(secretKey), bcrypt.DefaultCost)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to hash secret key: %w", err)
	}
	secretKeyHash = string(hashBytes)

	return accessKey, secretKey, secretKeyHash, nil
}

func VerifySecretKey(secretKey, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(secretKey))
	return err == nil
}
