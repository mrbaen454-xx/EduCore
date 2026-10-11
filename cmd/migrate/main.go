package main

import (
	"flag"
	"fmt"
	"os"

	"educore/internal/config"
	"educore/internal/platform/database"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	migrationsDir := "migrations"

	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading configuration: %v\n", err)
		os.Exit(1)
	}

	switch command {
	case "up":
		fmt.Printf("Applying migrations from %s to %s...\n", migrationsDir, cfg.SanitizedDSN())
		if err := database.MigrateUp(migrationsDir, cfg.DSN()); err != nil {
			fmt.Fprintf(os.Stderr, "migration up failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Migrations applied successfully.")

	case "down":
		downCmd := flag.NewFlagSet("down", flag.ExitOnError)
		steps := downCmd.Int("steps", 1, "number of migration steps to rollback")
		_ = downCmd.Parse(os.Args[2:])

		fmt.Printf("Rolling back %d migration step(s) from %s...\n", *steps, cfg.SanitizedDSN())
		if err := database.MigrateDown(migrationsDir, cfg.DSN(), *steps); err != nil {
			fmt.Fprintf(os.Stderr, "migration down failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Rollback completed successfully.")

	case "version":
		version, dirty, err := database.MigrationVersion(migrationsDir, cfg.DSN())
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to get migration version: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Current migration version: %d (dirty: %v)\n", version, dirty)

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("EduCore Database Migration Tool")
	fmt.Println("Usage:")
	fmt.Println("  go run cmd/migrate/main.go up             - Apply all pending migrations")
	fmt.Println("  go run cmd/migrate/main.go down [-steps N] - Rollback N migration steps (default 1)")
	fmt.Println("  go run cmd/migrate/main.go version        - Display current migration version")
}
