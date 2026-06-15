package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/estradax/cistern/internal/apikey"
	"github.com/estradax/cistern/internal/client"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
)

func main() {
	if len(os.Args) < 4 {
		printUsageAndExit()
	}

	sub := os.Args[1]
	action := os.Args[2]
	payload := os.Args[3]

	if sub != "clients" && sub != "apikeys" {
		printUsageAndExit()
	}

	_ = godotenv.Load()

	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		log.Fatal("DB_DSN environment variable is not set")
	}

	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	switch sub {
	case "clients":
		repo := client.NewRepository(db)
		switch action {
		case "create":
			var input client.CreateClientInput
			if err := json.Unmarshal([]byte(payload), &input); err != nil {
				log.Fatalf("Invalid JSON payload for create: %v", err)
			}

			c, err := repo.Create(ctx, input)
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}
			printJSON(c)

		case "read":
			id := extractID(payload)
			if id == "" {
				log.Fatal("Client ID cannot be empty")
			}

			c, err := repo.Get(ctx, id)
			if err != nil {
				log.Fatalf("Failed to retrieve client: %v", err)
			}
			if c == nil {
				log.Fatalf("Client not found: %s", id)
			}
			printJSON(c)

		case "update":
			var input client.UpdateClientInput
			if err := json.Unmarshal([]byte(payload), &input); err != nil {
				log.Fatalf("Invalid JSON payload for update: %v", err)
			}

			c, err := repo.Update(ctx, input)
			if err != nil {
				log.Fatalf("Failed to update client: %v", err)
			}
			printJSON(c)

		case "delete":
			id := extractID(payload)
			if id == "" {
				log.Fatal("Client ID cannot be empty")
			}

			err := repo.Delete(ctx, id)
			if err != nil {
				log.Fatalf("Failed to delete client: %v", err)
			}
			fmt.Printf(`{"status":"success","deleted_id":%q}`+"\n", id)

		default:
			printUsageAndExit()
		}
	case "apikeys":
		repo := apikey.NewRepository(db)
		switch action {
		case "generate":
			var input apikey.CreateAPIKeyInput
			if err := json.Unmarshal([]byte(payload), &input); err != nil {
				log.Fatalf("Invalid JSON payload for generate: %v", err)
			}

			res, err := repo.Create(ctx, input)
			if err != nil {
				log.Fatalf("Failed to generate API key: %v", err)
			}
			printJSON(res)

		case "read":
			id := extractID(payload)
			if id == "" {
				log.Fatal("API key ID cannot be empty")
			}

			key, err := repo.Get(ctx, id)
			if err != nil {
				log.Fatalf("Failed to retrieve API key: %v", err)
			}
			if key == nil {
				log.Fatalf("API key not found: %s", id)
			}
			printJSON(key)

		case "delete":
			id := extractID(payload)
			if id == "" {
				log.Fatal("API key ID cannot be empty")
			}

			err := repo.Delete(ctx, id)
			if err != nil {
				log.Fatalf("Failed to delete API key: %v", err)
			}
			fmt.Printf(`{"status":"success","deleted_id":%q}`+"\n", id)

		default:
			printUsageAndExit()
		}
	}
}

func printUsageAndExit() {
	fmt.Println("Usage:")
	fmt.Println("  cistern clients create '<json_payload>'   (e.g., '{\"name\": \"my-client\"}')")
	fmt.Println("  cistern clients read '<id_or_json>'       (e.g., 'some-uuid-here' or '{\"id\": \"some-uuid\"}')")
	fmt.Println("  cistern clients update '<json_payload>'   (e.g., '{\"id\": \"uuid\", \"name\": \"new-name\"}')")
	fmt.Println("  cistern clients delete '<id_or_json>'     (e.g., 'some-uuid-here' or '{\"id\": \"some-uuid\"}')")
	fmt.Println("  cistern apikeys generate '<json_payload>' (e.g., '{\"client_id\": \"client-uuid\", \"name\": \"my-key\"}')")
	fmt.Println("  cistern apikeys read '<id_or_json>'       (e.g., 'some-uuid-here' or '{\"id\": \"some-uuid\"}')")
	fmt.Println("  cistern apikeys delete '<id_or_json>'     (e.g., 'some-uuid-here' or '{\"id\": \"some-uuid\"}')")
	os.Exit(1)
}

func printJSON(v interface{}) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal JSON response: %v", err)
	}
	fmt.Println(string(data))
}

func extractID(payload string) string {
	trimmed := strings.TrimSpace(payload)
	if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
		var data struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal([]byte(trimmed), &data); err == nil && data.ID != "" {
			return data.ID
		}
	}
	return trimmed
}
