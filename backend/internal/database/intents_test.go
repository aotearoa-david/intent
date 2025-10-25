package database

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

func TestGetIntentSuccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	id := uuid.New()
	createdAt := time.Now().UTC()

	rows := sqlmock.NewRows([]string{"id", "statement", "context", "expected_outcome", "collaborators", "created_at"}).
		AddRow(id, "statement", "context", "outcome", `["Jamie"]`, createdAt)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, statement, context, expected_outcome, collaborators, created_at FROM intents WHERE id = $1")).
		WithArgs(id).
		WillReturnRows(rows)

	intent, err := GetIntent(context.Background(), db, id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if intent.ID != id {
		t.Fatalf("expected id %s, got %s", id, intent.ID)
	}

	if len(intent.Collaborators) != 1 || intent.Collaborators[0] != "Jamie" {
		t.Fatalf("expected collaborators to be unmarshalled")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestGetIntentNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	id := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, statement, context, expected_outcome, collaborators, created_at FROM intents WHERE id = $1")).
		WithArgs(id).
		WillReturnError(sql.ErrNoRows)

	if _, err := GetIntent(context.Background(), db, id); err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestUpdateIntentSuccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	id := uuid.New()
	createdAt := time.Now().UTC()

	input := IntentInput{
		Statement:       "updated",
		Context:         "context",
		ExpectedOutcome: "outcome",
		Collaborators:   []string{"Jamie", "Ana"},
	}

	mock.ExpectQuery(regexp.QuoteMeta(`UPDATE intents
SET statement = $1,
    context = $2,
    expected_outcome = $3,
    collaborators = $4
WHERE id = $5
RETURNING id, statement, context, expected_outcome, collaborators, created_at`)).
		WithArgs(input.Statement, input.Context, input.ExpectedOutcome, `["Jamie","Ana"]`, id).
		WillReturnRows(sqlmock.NewRows([]string{"id", "statement", "context", "expected_outcome", "collaborators", "created_at"}).
			AddRow(id, input.Statement, input.Context, input.ExpectedOutcome, `["Jamie","Ana"]`, createdAt))

	intent, err := UpdateIntent(context.Background(), db, id, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if intent.Statement != input.Statement {
		t.Fatalf("expected statement %q got %q", input.Statement, intent.Statement)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestDeleteIntent(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	id := uuid.New()

	mock.ExpectExec("DELETE FROM intents").
		WithArgs(id).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := DeleteIntent(context.Background(), db, id); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestDeleteIntentNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	id := uuid.New()

	mock.ExpectExec("DELETE FROM intents").
		WithArgs(id).
		WillReturnResult(sqlmock.NewResult(0, 0))

	if err := DeleteIntent(context.Background(), db, id); err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestListIntentsWithFilters(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	filters := IntentFilters{Query: "improve", Collaborator: "Jamie"}
	pagination := Pagination{Limit: 10, Offset: 10}

	createdAt := time.Now().UTC()
	id := uuid.New()

	pattern := "%improve%"

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM intents WHERE (statement ILIKE $1 OR context ILIKE $2 OR expected_outcome ILIKE $3) AND EXISTS (SELECT 1 FROM jsonb_array_elements_text(collaborators) AS c WHERE LOWER(c) = LOWER($4))")).
		WithArgs(pattern, pattern, pattern, filters.Collaborator).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, statement, context, expected_outcome, collaborators, created_at FROM intents WHERE (statement ILIKE $1 OR context ILIKE $2 OR expected_outcome ILIKE $3) AND EXISTS (SELECT 1 FROM jsonb_array_elements_text(collaborators) AS c WHERE LOWER(c) = LOWER($4)) ORDER BY created_at DESC LIMIT $5 OFFSET $6")).
		WithArgs(pattern, pattern, pattern, filters.Collaborator, pagination.Limit, pagination.Offset).
		WillReturnRows(sqlmock.NewRows([]string{"id", "statement", "context", "expected_outcome", "collaborators", "created_at"}).
			AddRow(id, "statement", "context", "outcome", `["Jamie"]`, createdAt))

	result, err := ListIntents(context.Background(), db, filters, pagination)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalCount != 1 {
		t.Fatalf("expected total count 1 got %d", result.TotalCount)
	}

	if len(result.Intents) != 1 {
		t.Fatalf("expected 1 intent got %d", len(result.Intents))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}
