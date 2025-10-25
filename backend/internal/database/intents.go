package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Intent represents a submitted intent from an engineer.
type Intent struct {
	ID              uuid.UUID
	Statement       string
	Context         string
	ExpectedOutcome string
	Collaborators   []string
	CreatedAt       time.Time
}

// IntentInput captures the minimal fields required to create an intent.
type IntentInput struct {
	Statement       string
	Context         string
	ExpectedOutcome string
	Collaborators   []string
}

// CreateIntent persists a new intent record and returns the stored entity.
func CreateIntent(ctx context.Context, db *sql.DB, input IntentInput) (Intent, error) {
	if db == nil {
		return Intent{}, errors.New("database handle is nil")
	}

	collaboratorJSON, err := json.Marshal(input.Collaborators)
	if err != nil {
		return Intent{}, err
	}

	now := time.Now().UTC()
	id := uuid.New()

	const query = `
INSERT INTO intents (id, statement, context, expected_outcome, collaborators, created_at)
VALUES ($1, $2, $3, $4, $5, $6)
`

	if _, err := db.ExecContext(ctx, query, id, input.Statement, input.Context, input.ExpectedOutcome, string(collaboratorJSON), now); err != nil {
		return Intent{}, err
	}

	return Intent{
		ID:              id,
		Statement:       input.Statement,
		Context:         input.Context,
		ExpectedOutcome: input.ExpectedOutcome,
		Collaborators:   input.Collaborators,
		CreatedAt:       now,
	}, nil
}
