package client_test

import (
	"context"
	"testing"
	"time"

	"github.com/estradax/cistern/internal/client"
	"github.com/estradax/cistern/internal/testutil"
)

func TestClientRepository_Create(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	t.Run("successful creation", func(t *testing.T) {
		testutil.CleanDatabase(t, db)
		repo := client.NewRepository(db)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		c, err := repo.Create(ctx, client.CreateClientInput{Name: "Acme Corp"})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if c.ID == "" {
			t.Fatal("expected client ID to be generated")
		}
		if c.Name != "Acme Corp" {
			t.Errorf("expected client Name to be 'Acme Corp', got %q", c.Name)
		}
	})

	t.Run("empty name error", func(t *testing.T) {
		testutil.CleanDatabase(t, db)
		repo := client.NewRepository(db)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := repo.Create(ctx, client.CreateClientInput{Name: ""})
		if err == nil {
			t.Fatal("expected error for empty name, got nil")
		}
	})
}

func TestClientRepository_Get(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	t.Run("get existing client", func(t *testing.T) {
		testutil.CleanDatabase(t, db)
		repo := client.NewRepository(db)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		c, err := repo.Create(ctx, client.CreateClientInput{Name: "Acme Corp"})
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		fetched, err := repo.Get(ctx, c.ID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if fetched == nil {
			t.Fatal("expected to fetch client, got nil")
		}
		if fetched.ID != c.ID {
			t.Errorf("expected client ID %q, got %q", c.ID, fetched.ID)
		}
		if fetched.Name != "Acme Corp" {
			t.Errorf("expected client Name 'Acme Corp', got %q", fetched.Name)
		}
	})

	t.Run("get non-existent client", func(t *testing.T) {
		testutil.CleanDatabase(t, db)
		repo := client.NewRepository(db)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		fetched, err := repo.Get(ctx, "non-existent-id")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if fetched != nil {
			t.Errorf("expected nil client, got %v", fetched)
		}
	})
}

func TestClientRepository_Update(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	t.Run("successful update", func(t *testing.T) {
		testutil.CleanDatabase(t, db)
		repo := client.NewRepository(db)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		c, err := repo.Create(ctx, client.CreateClientInput{Name: "Old Name"})
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		updated, err := repo.Update(ctx, client.UpdateClientInput{
			ID:   c.ID,
			Name: "New Name",
		})
		if err != nil {
			t.Fatalf("expected no error on update, got %v", err)
		}
		if updated.Name != "New Name" {
			t.Errorf("expected client Name to be updated to 'New Name', got %q", updated.Name)
		}

		fetched, err := repo.Get(ctx, c.ID)
		if err != nil {
			t.Fatalf("failed to fetch client: %v", err)
		}
		if fetched.Name != "New Name" {
			t.Errorf("expected database Name to be 'New Name', got %q", fetched.Name)
		}
	})

	t.Run("validation and missing client error cases", func(t *testing.T) {
		testutil.CleanDatabase(t, db)
		repo := client.NewRepository(db)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := repo.Update(ctx, client.UpdateClientInput{Name: "New Name"})
		if err == nil {
			t.Error("expected error for empty ID, got nil")
		}

		_, err = repo.Update(ctx, client.UpdateClientInput{ID: "some-id", Name: ""})
		if err == nil {
			t.Error("expected error for empty Name, got nil")
		}

		_, err = repo.Update(ctx, client.UpdateClientInput{ID: "non-existent-id", Name: "Name"})
		if err == nil {
			t.Error("expected error for non-existent client, got nil")
		}
	})
}

func TestClientRepository_Delete(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	t.Run("successful delete", func(t *testing.T) {
		testutil.CleanDatabase(t, db)
		repo := client.NewRepository(db)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		c, err := repo.Create(ctx, client.CreateClientInput{Name: "To Be Deleted"})
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		err = repo.Delete(ctx, c.ID)
		if err != nil {
			t.Fatalf("expected no error on delete, got %v", err)
		}

		fetched, err := repo.Get(ctx, c.ID)
		if err != nil {
			t.Fatalf("failed to query client: %v", err)
		}
		if fetched != nil {
			t.Error("expected client to be deleted, but still exists")
		}
	})

	t.Run("empty ID error", func(t *testing.T) {
		testutil.CleanDatabase(t, db)
		repo := client.NewRepository(db)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := repo.Delete(ctx, "")
		if err == nil {
			t.Error("expected error for empty ID, got nil")
		}
	})
}
