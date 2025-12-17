package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// Cargar .env
	godotenv.Load()

	if len(os.Args) < 2 {
		fmt.Println("Usage: go run cmd/migrate/main.go [up|down|status]")
		os.Exit(1)
	}

	command := os.Args[1]

	// Obtener URL de la base de datos
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		fmt.Println("Error: DATABASE_URL environment variable is required")
		os.Exit(1)
	}

	// Conectar a la base de datos
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Printf("Error connecting to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Verificar conexión
	if err := db.Ping(); err != nil {
		fmt.Printf("Error pinging database: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Connected to database")

	// Crear tabla de migraciones si no existe
	if err := createMigrationsTable(db); err != nil {
		fmt.Printf("Error creating migrations table: %v\n", err)
		os.Exit(1)
	}

	switch command {
	case "up":
		if err := migrateUp(db); err != nil {
			fmt.Printf("Error running migrations: %v\n", err)
			os.Exit(1)
		}
	case "down":
		if err := migrateDown(db); err != nil {
			fmt.Printf("Error rolling back migration: %v\n", err)
			os.Exit(1)
		}
	case "status":
		if err := showStatus(db); err != nil {
			fmt.Printf("Error showing status: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Println("Usage: go run cmd/migrate/main.go [up|down|status]")
		os.Exit(1)
	}
}

func createMigrationsTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)
	`
	_, err := db.Exec(query)
	return err
}

func getAppliedMigrations(db *sql.DB) (map[string]bool, error) {
	applied := make(map[string]bool)

	rows, err := db.Query("SELECT version FROM schema_migrations ORDER BY version")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}

	return applied, rows.Err()
}

func getMigrationFiles(suffix string) ([]string, error) {
	var files []string

	entries, err := os.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("error reading migrations directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, suffix) {
			files = append(files, name)
		}
	}

	sort.Strings(files)
	return files, nil
}

func getVersionFromFilename(filename string) string {
	// Formato: 000001_initial_schema.up.sql -> 000001_initial_schema
	parts := strings.Split(filename, ".")
	if len(parts) >= 2 {
		return parts[0]
	}
	return filename
}

func migrateUp(db *sql.DB) error {
	applied, err := getAppliedMigrations(db)
	if err != nil {
		return err
	}

	files, err := getMigrationFiles(".up.sql")
	if err != nil {
		return err
	}

	if len(files) == 0 {
		fmt.Println("No migration files found")
		return nil
	}

	migrationsRun := 0

	for _, file := range files {
		version := getVersionFromFilename(file)

		if applied[version] {
			continue
		}

		fmt.Printf("Applying migration: %s\n", file)

		content, err := os.ReadFile(filepath.Join("migrations", file))
		if err != nil {
			return fmt.Errorf("error reading migration file %s: %w", file, err)
		}

		// Ejecutar migración en una transacción
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("error starting transaction: %w", err)
		}

		if _, err := tx.Exec(string(content)); err != nil {
			tx.Rollback()
			return fmt.Errorf("error executing migration %s: %w", file, err)
		}

		if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", version); err != nil {
			tx.Rollback()
			return fmt.Errorf("error recording migration %s: %w", file, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("error committing migration %s: %w", file, err)
		}

		fmt.Printf("✓ Applied: %s\n", file)
		migrationsRun++
	}

	if migrationsRun == 0 {
		fmt.Println("✓ Database is up to date")
	} else {
		fmt.Printf("✓ Applied %d migration(s)\n", migrationsRun)
	}

	return nil
}

func migrateDown(db *sql.DB) error {
	// Obtener la última migración aplicada
	var lastVersion string
	err := db.QueryRow("SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1").Scan(&lastVersion)
	if err == sql.ErrNoRows {
		fmt.Println("No migrations to rollback")
		return nil
	}
	if err != nil {
		return fmt.Errorf("error getting last migration: %w", err)
	}

	// Buscar el archivo down correspondiente
	downFile := lastVersion + ".down.sql"

	content, err := os.ReadFile(filepath.Join("migrations", downFile))
	if err != nil {
		return fmt.Errorf("error reading migration file %s: %w", downFile, err)
	}

	fmt.Printf("Rolling back migration: %s\n", lastVersion)

	// Ejecutar rollback en una transacción
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}

	if _, err := tx.Exec(string(content)); err != nil {
		tx.Rollback()
		return fmt.Errorf("error executing rollback %s: %w", downFile, err)
	}

	if _, err := tx.Exec("DELETE FROM schema_migrations WHERE version = $1", lastVersion); err != nil {
		tx.Rollback()
		return fmt.Errorf("error removing migration record %s: %w", lastVersion, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing rollback %s: %w", lastVersion, err)
	}

	fmt.Printf("✓ Rolled back: %s\n", lastVersion)
	return nil
}

func showStatus(db *sql.DB) error {
	applied, err := getAppliedMigrations(db)
	if err != nil {
		return err
	}

	files, err := getMigrationFiles(".up.sql")
	if err != nil {
		return err
	}

	fmt.Println("\nMigration Status:")
	fmt.Println("─────────────────────────────────────────────")

	for _, file := range files {
		version := getVersionFromFilename(file)
		status := "Pending"
		if applied[version] {
			status = "Applied"
		}
		fmt.Printf("[%s] %s\n", status, version)
	}

	fmt.Println("─────────────────────────────────────────────")
	fmt.Printf("Total: %d migrations, %d applied, %d pending\n",
		len(files), len(applied), len(files)-len(applied))

	return nil
}

