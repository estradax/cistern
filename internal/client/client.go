package client

import "time"

type Client struct {
	ID        string    `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type CreateClientInput struct {
	Name string `json:"name"`
}

type UpdateClientInput struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
