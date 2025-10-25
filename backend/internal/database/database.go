package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Config encapsulates the connection settings for Postgres.
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
}

// DSN returns a Postgres connection string based on the configuration.
func (c Config) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s", c.User, c.Password, c.Host, c.Port, c.Database)
}

// Open returns a sql.DB configured with pgx driver.
func Open(cfg Config) (*sql.DB, error) {
	db, err := sql.Open("pgx", cfg.DSN())
	if err != nil {
		return nil, err
	}
	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	return db, nil
}

// FetchGreeting asks Postgres to produce a hello world greeting.
func FetchGreeting(ctx context.Context, db *sql.DB) (string, error) {
	var greeting string
	row := db.QueryRowContext(ctx, "select 'Hello, Intent!' as greeting")
	if err := row.Scan(&greeting); err != nil {
		return "", err
	}
	return greeting, nil
}
