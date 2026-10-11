package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"educore/internal/modules/auth"
)

func main() {
	emailFlag := flag.String("email", os.Getenv("SEED_USER_EMAIL"), "Email pengguna awal")
	pwdFlag := flag.String("password", os.Getenv("SEED_USER_PASSWORD"), "Password pengguna awal")
	roleFlag := flag.String("role", "admin", "Role awal (admin/dosen/mahasiswa/akademik)")
	nameFlag := flag.String("name", "Administrator SIAKAD", "Nama lengkap pengguna")
	flag.Parse()

	if *emailFlag == "" || *pwdFlag == "" {
		log.Fatal("Error: flag -email dan -password wajib diisi (atau melalui SEED_USER_EMAIL & SEED_USER_PASSWORD)")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/educore?sslmode=disable"
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer pool.Close()

	hasher := auth.NewArgon2Hasher(65536, 3, 4, 16, 32)
	pwdHash, err := hasher.Hash(*pwdFlag)
	if err != nil {
		log.Fatalf("failed to hash password: %v", err)
	}

	var userID string
	err = pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, name, is_active)
		VALUES ($1, $2, $3, true)
		ON CONFLICT (email) DO UPDATE SET password_hash = $2, name = $3, is_active = true
		RETURNING id
	`, *emailFlag, pwdHash, *nameFlag).Scan(&userID)
	if err != nil {
		log.Fatalf("failed to insert/update user: %v", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO user_roles (user_id, role_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, role_id) DO NOTHING
	`, userID, *roleFlag)
	if err != nil {
		log.Fatalf("failed to assign role: %v", err)
	}

	fmt.Printf("Pengguna berhasil dibuat/diperbarui: %s (ID: %s, Role: %s)\n", *emailFlag, userID, *roleFlag)
}
