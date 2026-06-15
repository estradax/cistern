package client

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

func (r *Repository) Create(ctx context.Context, input CreateClientInput) (*Client, error) {
	if input.Name == "" {
		return nil, errors.New("client name cannot be empty")
	}

	client := &Client{
		ID:        uuid.New().String(),
		Name:      input.Name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	query := `INSERT INTO clients (id, name, created_at, updated_at) VALUES (:id, :name, :created_at, :updated_at)`
	_, err := r.db.NamedExecContext(ctx, query, client)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (r *Repository) Get(ctx context.Context, id string) (*Client, error) {
	var client Client
	err := r.db.GetContext(ctx, &client, "SELECT id, name, created_at, updated_at FROM clients WHERE id = ?", id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &client, nil
}

func (r *Repository) Update(ctx context.Context, input UpdateClientInput) (*Client, error) {
	if input.ID == "" {
		return nil, errors.New("client ID cannot be empty")
	}
	if input.Name == "" {
		return nil, errors.New("client name cannot be empty")
	}

	client, err := r.Get(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, errors.New("client not found")
	}

	client.Name = input.Name
	client.UpdatedAt = time.Now()

	query := `UPDATE clients SET name = :name, updated_at = :updated_at WHERE id = :id`
	_, err = r.db.NamedExecContext(ctx, query, client)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("client ID cannot be empty")
	}
	_, err := r.db.ExecContext(ctx, "DELETE FROM clients WHERE id = ?", id)
	return err
}
