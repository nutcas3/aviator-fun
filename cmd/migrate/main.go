package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"aviator/internal/database"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable&search_path=%s",
		getEnv("BLUEPRINT_DB_USERNAME", "postgres"),
		getEnv("BLUEPRINT_DB_PASSWORD", "postgres"),
		getEnv("BLUEPRINT_DB_HOST", "localhost"),
		getEnv("BLUEPRINT_DB_PORT", "5432"),
		getEnv("BLUEPRINT_DB_DATABASE", "crashdb"),
		getEnv("BLUEPRINT_DB_SCHEMA", "public"),
	)

	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	migrationsPath := getEnv("MIGRATIONS_PATH", "./migrations")

	switch command {
	case "up":
		log.Println("Running migrations...")
		if err := database.RunMigrations(db, migrationsPath); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
		log.Println("Migrations completed successfully")

	case "down":
		log.Println("Rolling back last migration...")
		if err := database.RollbackMigration(db, migrationsPath); err != nil {
			log.Fatalf("Rollback failed: %v", err)
		}
		log.Println("Rollback completed successfully")

	case "version":
		version, dirty, err := database.GetMigrationVersion(db, migrationsPath)
		if err != nil {
			log.Fatalf("Failed to get version: %v", err)
		}
		if dirty {
			log.Printf("Current version: %d (DIRTY - needs manual intervention)", version)
		} else {
			log.Printf("Current version: %d", version)
		}

	case "create":
		if len(os.Args) < 3 {
			log.Fatal("Usage: migrate create <migration_name>")
		}
		createMigration(os.Args[2])

	default:
		log.Printf("Unknown command: %s", command)
		printUsage()
		os.Exit(1)
	}
}

func createMigration(name string) {
	files, err := os.ReadDir("./migrations")
	if err != nil {
		log.Fatalf("Failed to read migrations directory: %v", err)
	}

	nextVersion := 1
	for _, file := range files {
		if !file.IsDir() {
			nextVersion++
		}
	}
	nextVersion = (nextVersion / 2) + 1 // Each migration has up and down files

	upFile := fmt.Sprintf("./migrations/%06d_%s.up.sql", nextVersion, name)
	downFile := fmt.Sprintf("./migrations/%06d_%s.down.sql", nextVersion, name)

	upContent := fmt.Sprintf("-- Migration: %s\n-- Created: %s\n\n-- Add your SQL here\n", name, "now")
	if err := os.WriteFile(upFile, []byte(upContent), 0644); err != nil {
		log.Fatalf("Failed to create up migration: %v", err)
	}
	downContent := fmt.Sprintf("-- Rollback: %s\n\n-- Add your rollback SQL here\n", name)
	if err := os.WriteFile(downFile, []byte(downContent), 0644); err != nil {
		log.Fatalf("Failed to create down migration: %v", err)
	}

	log.Printf("Created migration files:")
	log.Printf("   - %s", upFile)
	log.Printf("   - %s", downFile)
}

func printUsage() {
	fmt.Println("Database Migration Tool")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  migrate up              Run all pending migrations")
	fmt.Println("  migrate down            Rollback the last migration")
	fmt.Println("  migrate version         Show current migration version")
	fmt.Println("  migrate create <name>   Create a new migration file")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  BLUEPRINT_DB_HOST       Database host (default: localhost)")
	fmt.Println("  BLUEPRINT_DB_PORT       Database port (default: 5432)")
	fmt.Println("  BLUEPRINT_DB_DATABASE   Database name (default: crashdb)")
	fmt.Println("  BLUEPRINT_DB_USERNAME   Database user (default: postgres)")
	fmt.Println("  BLUEPRINT_DB_PASSWORD   Database password (default: postgres)")
	fmt.Println("  MIGRATIONS_PATH         Path to migrations (default: ./migrations)")
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
