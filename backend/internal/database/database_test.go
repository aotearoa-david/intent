package database_test

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/example/intent/backend/internal/database"
)

func TestFetchGreeting(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	mock.ExpectQuery("select 'Hello, Intent!' as greeting").WillReturnRows(sqlmock.NewRows([]string{"greeting"}).AddRow("Hello, Intent!"))

	greeting, err := database.FetchGreeting(context.Background(), db)
	if err != nil {
		t.Fatalf("FetchGreeting returned error: %v", err)
	}

	if greeting != "Hello, Intent!" {
		t.Fatalf("expected greeting to be Hello, Intent!, got %s", greeting)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestConfigDSN(t *testing.T) {
	cfg := database.Config{
		Host:     "db",
		Port:     5432,
		User:     "postgres",
		Password: "secret",
		Database: "intent",
	}

	dsn := cfg.DSN()
	expected := "postgres://postgres:secret@db:5432/intent"

	if dsn != expected {
		t.Fatalf("unexpected DSN. got %s want %s", dsn, expected)
	}
}
