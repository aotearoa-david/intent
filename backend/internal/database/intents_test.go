package database_test

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"

	"github.com/example/intent/backend/internal/database"
)

func TestCreateIntent(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	input := database.IntentInput{
		Statement:       "I intend to document our deployment pipeline.",
		Context:         "The team lacks clarity on the steps.",
		ExpectedOutcome: "A shareable runbook for Chapter review.",
		Collaborators:   []string{"Asha", "Miguel"},
	}

	mock.ExpectExec("INSERT INTO intents").
		WithArgs(sqlmock.AnyArg(), input.Statement, input.Context, input.ExpectedOutcome, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	got, err := database.CreateIntent(context.Background(), db, input)
	if err != nil {
		t.Fatalf("CreateIntent returned error: %v", err)
	}

	if got.Statement != input.Statement {
		t.Errorf("expected statement %q, got %q", input.Statement, got.Statement)
	}

	if got.Context != input.Context {
		t.Errorf("expected context %q, got %q", input.Context, got.Context)
	}

	if got.ExpectedOutcome != input.ExpectedOutcome {
		t.Errorf("expected expected outcome %q, got %q", input.ExpectedOutcome, got.ExpectedOutcome)
	}

	if len(got.Collaborators) != len(input.Collaborators) {
		t.Fatalf("expected %d collaborators, got %d", len(input.Collaborators), len(got.Collaborators))
	}

	for i, collaborator := range input.Collaborators {
		if got.Collaborators[i] != collaborator {
			t.Errorf("unexpected collaborator at index %d: want %q got %q", i, collaborator, got.Collaborators[i])
		}
	}

	if got.ID == uuid.Nil {
		t.Error("expected non-zero UUID")
	}

	if time.Since(got.CreatedAt) > time.Minute {
		t.Error("expected CreatedAt to be recent")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
