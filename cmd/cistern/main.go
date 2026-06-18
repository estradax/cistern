package main

import (
	"context"
	"log"
	"os"
	"time"

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

	if sub != "clients" && sub != "apikeys" && sub != "buckets" && sub != "objects" {
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

	var cmdErr error
	switch sub {
	case "clients":
		cmdErr = handleClients(ctx, db, action, payload)
	case "apikeys":
		cmdErr = handleAPIKeys(ctx, db, action, payload)
	case "buckets":
		cmdErr = handleBuckets(ctx, db, action, payload)
	case "objects":
		extraArgs := os.Args[4:]
		cmdErr = handleObjects(ctx, db, action, payload, extraArgs)
	}

	if cmdErr != nil {
		log.Fatalf("Error: %v", cmdErr)
	}
}
