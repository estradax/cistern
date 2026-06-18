package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	migratemysql "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
)

func FindProjectRoot() string {
	_, b, _, _ := runtime.Caller(0)
	dir := filepath.Dir(b)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func SetupTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	projectRoot := FindProjectRoot()
	if projectRoot == "" {
		t.Fatal("Could not find project root containing go.mod")
	}

	envPath := filepath.Join(projectRoot, ".env.testing")
	if err := godotenv.Load(envPath); err != nil {
		t.Fatalf("Error loading %s: %v", envPath, err)
	}

	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		t.Fatal("DB_DSN environment variable not set in .env.testing")
	}

	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatalf("Failed to parse DSN: %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	pkgName := filepath.Base(wd)
	dbName := fmt.Sprintf("%s_%s", cfg.DBName, pkgName)

	cfg.DBName = ""
	serverDSN := cfg.FormatDSN()

	serverDb, err := sqlx.Connect("mysql", serverDSN)
	if err != nil {
		t.Fatalf("Failed to connect to MySQL server: %v", err)
	}

	_, err = serverDb.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`;", dbName))
	if err != nil {
		serverDb.Close()
		t.Fatalf("Failed to drop old test database: %v", err)
	}

	_, err = serverDb.Exec(fmt.Sprintf("CREATE DATABASE `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;", dbName))
	if err != nil {
		serverDb.Close()
		t.Fatalf("Failed to create test database: %v", err)
	}
	serverDb.Close()

	cfg.DBName = dbName
	dbDSN := cfg.FormatDSN()
	db, err := sqlx.Connect("mysql", dbDSN)
	if err != nil {
		t.Fatalf("Failed to connect to test database %s: %v", dbName, err)
	}

	driver, err := migratemysql.WithInstance(db.DB, &migratemysql.Config{})
	if err != nil {
		db.Close()
		t.Fatalf("Failed to initialize migration driver: %v", err)
	}

	migrationsPath := "file://" + filepath.Join(projectRoot, "db", "migrations")
	m, err := migrate.NewWithDatabaseInstance(migrationsPath, "mysql", driver)
	if err != nil {
		db.Close()
		t.Fatalf("Failed to construct migrate instance: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		db.Close()
		t.Fatalf("Failed to run migrations Up: %v", err)
	}

	return db
}

func CleanDatabase(t *testing.T, db *sqlx.DB) {
	t.Helper()

	queries := []string{
		"SET FOREIGN_KEY_CHECKS = 0;",
		"TRUNCATE TABLE objects;",
		"TRUNCATE TABLE buckets;",
		"TRUNCATE TABLE api_keys;",
		"TRUNCATE TABLE clients;",
		"SET FOREIGN_KEY_CHECKS = 1;",
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("Failed execution of query %q: %v", q, err)
		}
	}
}
