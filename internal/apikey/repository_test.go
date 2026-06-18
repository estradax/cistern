package apikey_test

import (
	"context"
	"testing"
	"time"

	"github.com/estradax/cistern/internal/apikey"
	"github.com/estradax/cistern/internal/client"
	"github.com/estradax/cistern/internal/testutil"
)

func TestAPIKeyRepository_Create(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	t.Run("successful creation", func(t *testing.T) {
		testutil.CleanDatabase(t, db)
		clientRepo := client.NewRepository(db)
		keyRepo := apikey.NewRepository(db)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		c, err := clientRepo.Create(ctx, client.CreateClientInput{Name: "Acme Corp"})
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		keyName := "Production Key"
		res, err := keyRepo.Create(ctx, apikey.CreateAPIKeyInput{
			ClientID: c.ID,
			Name:     &keyName,
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if res.APIKey.ID == "" {
			t.Fatal("expected API key ID to be populated")
		}
		if res.APIKey.ClientID != c.ID {
			t.Errorf("expected client ID %q, got %q", c.ID, res.APIKey.ClientID)
		}
		if *res.APIKey.Name != keyName {
			t.Errorf("expected name %q, got %q", keyName, *res.APIKey.Name)
		}
		if res.SecretKey == "" {
			t.Fatal("expected secret key to be returned raw in the result")
		}

		if !apikey.VerifySecretKey(res.SecretKey, res.APIKey.SecretKeyHash) {
			t.Error("expected raw secret key to match stored hash")
		}
	})

	t.Run("empty client ID error", func(t *testing.T) {
		testutil.CleanDatabase(t, db)
		keyRepo := apikey.NewRepository(db)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := keyRepo.Create(ctx, apikey.CreateAPIKeyInput{
			ClientID: "",
		})
		if err == nil {
			t.Fatal("expected error for empty client ID, got nil")
		}
	})
}

func TestAPIKeyRepository_GetAndGetByAccessKey(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	testutil.CleanDatabase(t, db)
	clientRepo := client.NewRepository(db)
	keyRepo := apikey.NewRepository(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c, err := clientRepo.Create(ctx, client.CreateClientInput{Name: "Acme Corp"})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	keyName := "Test Key"
	res, err := keyRepo.Create(ctx, apikey.CreateAPIKeyInput{
		ClientID: c.ID,
		Name:     &keyName,
	})
	if err != nil {
		t.Fatalf("failed to create API key: %v", err)
	}

	t.Run("get by ID", func(t *testing.T) {
		fetched, err := keyRepo.Get(ctx, res.APIKey.ID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if fetched == nil {
			t.Fatal("expected to find API key, got nil")
		}
		if fetched.ID != res.APIKey.ID {
			t.Errorf("expected ID %q, got %q", res.APIKey.ID, fetched.ID)
		}
	})

	t.Run("get by access key", func(t *testing.T) {
		fetched, err := keyRepo.GetByAccessKey(ctx, res.APIKey.AccessKey)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if fetched == nil {
			t.Fatal("expected to find API key, got nil")
		}
		if fetched.ID != res.APIKey.ID {
			t.Errorf("expected ID %q, got %q", res.APIKey.ID, fetched.ID)
		}
	})

	t.Run("get non-existent ID", func(t *testing.T) {
		fetched, err := keyRepo.Get(ctx, "non-existent-id")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if fetched != nil {
			t.Errorf("expected nil API key, got %v", fetched)
		}
	})
}

func TestAPIKeyRepository_ListByClient(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	testutil.CleanDatabase(t, db)
	clientRepo := client.NewRepository(db)
	keyRepo := apikey.NewRepository(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c1, _ := clientRepo.Create(ctx, client.CreateClientInput{Name: "Client 1"})
	c2, _ := clientRepo.Create(ctx, client.CreateClientInput{Name: "Client 2"})

	name1 := "Key 1"
	name2 := "Key 2"
	keyRepo.Create(ctx, apikey.CreateAPIKeyInput{ClientID: c1.ID, Name: &name1})
	keyRepo.Create(ctx, apikey.CreateAPIKeyInput{ClientID: c1.ID, Name: &name2})
	keyRepo.Create(ctx, apikey.CreateAPIKeyInput{ClientID: c2.ID, Name: &name1})

	t.Run("list for client 1", func(t *testing.T) {
		list, err := keyRepo.ListByClient(ctx, c1.ID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(list) != 2 {
			t.Fatalf("expected 2 API keys, got %d", len(list))
		}
	})

	t.Run("list for empty client ID", func(t *testing.T) {
		_, err := keyRepo.ListByClient(ctx, "")
		if err == nil {
			t.Fatal("expected error for empty client ID, got nil")
		}
	})
}

func TestAPIKeyRepository_UpdateAndDelete(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	t.Run("update name", func(t *testing.T) {
		testutil.CleanDatabase(t, db)
		clientRepo := client.NewRepository(db)
		keyRepo := apikey.NewRepository(db)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		c, _ := clientRepo.Create(ctx, client.CreateClientInput{Name: "Client"})
		name := "Original Name"
		res, _ := keyRepo.Create(ctx, apikey.CreateAPIKeyInput{ClientID: c.ID, Name: &name})

		newName := "Updated Name"
		updated, err := keyRepo.Update(ctx, apikey.UpdateAPIKeyInput{
			ID:   res.APIKey.ID,
			Name: &newName,
		})
		if err != nil {
			t.Fatalf("expected no error on update, got %v", err)
		}
		if *updated.Name != newName {
			t.Errorf("expected name to be updated to %q, got %q", newName, *updated.Name)
		}
	})

	t.Run("delete API key", func(t *testing.T) {
		testutil.CleanDatabase(t, db)
		clientRepo := client.NewRepository(db)
		keyRepo := apikey.NewRepository(db)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		c, _ := clientRepo.Create(ctx, client.CreateClientInput{Name: "Client"})
		res, _ := keyRepo.Create(ctx, apikey.CreateAPIKeyInput{ClientID: c.ID})

		err := keyRepo.Delete(ctx, res.APIKey.ID)
		if err != nil {
			t.Fatalf("expected no error on delete, got %v", err)
		}

		fetched, _ := keyRepo.Get(ctx, res.APIKey.ID)
		if fetched != nil {
			t.Error("expected key to be deleted, but still exists")
		}
	})

	t.Run("cascading delete on client deletion", func(t *testing.T) {
		testutil.CleanDatabase(t, db)
		clientRepo := client.NewRepository(db)
		keyRepo := apikey.NewRepository(db)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		c, _ := clientRepo.Create(ctx, client.CreateClientInput{Name: "Client to delete"})
		res, _ := keyRepo.Create(ctx, apikey.CreateAPIKeyInput{ClientID: c.ID})

		err := clientRepo.Delete(ctx, c.ID)
		if err != nil {
			t.Fatalf("failed to delete client: %v", err)
		}

		fetched, _ := keyRepo.Get(ctx, res.APIKey.ID)
		if fetched != nil {
			t.Error("expected API key to be cascade deleted, but it still exists")
		}
	})
}
