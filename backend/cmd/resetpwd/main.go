package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		fmt.Println("Error: DATABASE_URL is required")
		os.Exit(1)
	}

	conn, err := pgx.Connect(context.Background(), dbURL)
	if err != nil {
		fmt.Printf("Error connecting: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

	// Generar hash correcto para "admin123"
	password := "admin123"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Printf("Error generating hash: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated hash for '%s': %s\n", password, string(hash))

	// Actualizar el usuario admin
	result, err := conn.Exec(context.Background(),
		"UPDATE users SET password_hash = $1 WHERE email = $2",
		string(hash), "admin@fintech.com")
	if err != nil {
		fmt.Printf("Error updating password: %v\n", err)
		os.Exit(1)
	}

	if result.RowsAffected() == 0 {
		fmt.Println("No user found with email admin@fintech.com")
		os.Exit(1)
	}

	fmt.Println("✓ Password reset successfully for admin@fintech.com")
	fmt.Println("  Email: admin@fintech.com")
	fmt.Println("  Password: admin123")
}

