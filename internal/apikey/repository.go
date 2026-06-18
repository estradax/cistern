package apikey

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) Create(ctx context.Context, input CreateAPIKeyInput) (*CreateAPIKeyResult, error) {
	if input.ClientID == "" {
		return nil, errors.New("client ID cannot be empty")
	}

	accessKey, secretKey, secretKeyHash, err := GenerateKeys()
	if err != nil {
		return nil, err
	}

	apiKey := &APIKey{
		ID:            uuid.New().String(),
		ClientID:      input.ClientID,
		Name:          input.Name,
		AccessKey:     accessKey,
		SecretKeyHash: secretKeyHash,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	query := `INSERT INTO api_keys (id, client_id, name, access_key, secret_key_hash, created_at, updated_at) 
	          VALUES (:id, :client_id, :name, :access_key, :secret_key_hash, :created_at, :updated_at)`

	_, err = r.db.NamedExecContext(ctx, query, apiKey)
	if err != nil {
		return nil, err
	}

	return &CreateAPIKeyResult{
		APIKey:    apiKey,
		SecretKey: secretKey,
	}, nil
}

func (r *Repository) Get(ctx context.Context, id string) (*APIKey, error) {
	if id == "" {
		return nil, errors.New("API key ID cannot be empty")
	}

	var apiKey APIKey
	query := `SELECT id, client_id, name, access_key, secret_key_hash, created_at, updated_at 
	          FROM api_keys WHERE id = ?`

	err := r.db.GetContext(ctx, &apiKey, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &apiKey, nil
}

func (r *Repository) GetByAccessKey(ctx context.Context, accessKey string) (*APIKey, error) {
	if accessKey == "" {
		return nil, errors.New("access key cannot be empty")
	}

	var apiKey APIKey
	query := `SELECT id, client_id, name, access_key, secret_key_hash, created_at, updated_at 
	          FROM api_keys WHERE access_key = ?`

	err := r.db.GetContext(ctx, &apiKey, query, accessKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &apiKey, nil
}

func (r *Repository) ListByClient(ctx context.Context, clientID string) ([]*APIKey, error) {
	if clientID == "" {
		return nil, errors.New("client ID cannot be empty")
	}

	var keys []*APIKey
	query := `SELECT id, client_id, name, access_key, secret_key_hash, created_at, updated_at 
	          FROM api_keys WHERE client_id = ? ORDER BY created_at DESC`

	err := r.db.SelectContext(ctx, &keys, query, clientID)
	if err != nil {
		return nil, err
	}

	return keys, nil
}

func (r *Repository) Update(ctx context.Context, input UpdateAPIKeyInput) (*APIKey, error) {
	if input.ID == "" {
		return nil, errors.New("API key ID cannot be empty")
	}

	apiKey, err := r.Get(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if apiKey == nil {
		return nil, errors.New("API key not found")
	}

	apiKey.Name = input.Name
	apiKey.UpdatedAt = time.Now()

	query := `UPDATE api_keys SET name = :name, updated_at = :updated_at WHERE id = :id`
	_, err = r.db.NamedExecContext(ctx, query, apiKey)
	if err != nil {
		return nil, err
	}

	return apiKey, nil
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("API key ID cannot be empty")
	}

	_, err := r.db.ExecContext(ctx, "DELETE FROM api_keys WHERE id = ?", id)
	return err
}
