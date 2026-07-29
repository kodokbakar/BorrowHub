//go:build integration

package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestPostgresConnection(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	expectedDatabase := os.Getenv("EXPECTED_DATABASE")

	if databaseURL == "" || expectedDatabase == "" {
		t.Fatal("DATABASE_URL and EXPECTED_DATABASE are required")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(context.Background())

	var database string

	if err := conn.QueryRow(
		ctx,
		"SELECT current_database()",
	).Scan(&database); err != nil {
		t.Fatal(err)
	}

	if database != expectedDatabase {
		t.Fatalf(
			"connected to %q, expected %q",
			database,
			expectedDatabase,
		)
	}
}
