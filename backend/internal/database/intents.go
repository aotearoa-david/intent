package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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

// IntentFilters capture optional filtering criteria when querying intents.
type IntentFilters struct {
	Query         string
	Collaborator  string
	CreatedAfter  *time.Time
	CreatedBefore *time.Time
}

// Pagination captures offset-based pagination inputs.
type Pagination struct {
	Limit  int
	Offset int
}

// IntentListResult represents the outcome of a list query.
type IntentListResult struct {
	Intents    []Intent
	TotalCount int
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

// GetIntent retrieves a single intent by identifier.
func GetIntent(ctx context.Context, db *sql.DB, id uuid.UUID) (Intent, error) {
	if db == nil {
		return Intent{}, errors.New("database handle is nil")
	}

	const query = `
SELECT id, statement, context, expected_outcome, collaborators, created_at
FROM intents
WHERE id = $1
`

	var (
		intent  Intent
		rawJSON []byte
	)

	err := db.QueryRowContext(ctx, query, id).Scan(&intent.ID, &intent.Statement, &intent.Context, &intent.ExpectedOutcome, &rawJSON, &intent.CreatedAt)
	if err != nil {
		return Intent{}, err
	}

	if len(rawJSON) > 0 {
		if err := json.Unmarshal(rawJSON, &intent.Collaborators); err != nil {
			return Intent{}, err
		}
	}

	return intent, nil
}

// UpdateIntent updates an existing intent and returns the persisted entity.
func UpdateIntent(ctx context.Context, db *sql.DB, id uuid.UUID, input IntentInput) (Intent, error) {
	if db == nil {
		return Intent{}, errors.New("database handle is nil")
	}

	collaboratorJSON, err := json.Marshal(input.Collaborators)
	if err != nil {
		return Intent{}, err
	}

	const query = `
UPDATE intents
SET statement = $1,
    context = $2,
    expected_outcome = $3,
    collaborators = $4
WHERE id = $5
RETURNING id, statement, context, expected_outcome, collaborators, created_at
`

	var (
		intent  Intent
		rawJSON []byte
	)

	err = db.QueryRowContext(ctx, query, input.Statement, input.Context, input.ExpectedOutcome, string(collaboratorJSON), id).Scan(
		&intent.ID,
		&intent.Statement,
		&intent.Context,
		&intent.ExpectedOutcome,
		&rawJSON,
		&intent.CreatedAt,
	)
	if err != nil {
		return Intent{}, err
	}

	if len(rawJSON) > 0 {
		if err := json.Unmarshal(rawJSON, &intent.Collaborators); err != nil {
			return Intent{}, err
		}
	}

	return intent, nil
}

// DeleteIntent removes an intent by identifier.
func DeleteIntent(ctx context.Context, db *sql.DB, id uuid.UUID) error {
	if db == nil {
		return errors.New("database handle is nil")
	}

	const query = `DELETE FROM intents WHERE id = $1`

	result, err := db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// ListIntents returns intents applying optional filters and pagination.
func ListIntents(ctx context.Context, db *sql.DB, filters IntentFilters, pagination Pagination) (IntentListResult, error) {
	if db == nil {
		return IntentListResult{}, errors.New("database handle is nil")
	}

	var (
		conditions []string
		args       []any
		param      = 1
	)

	if strings.TrimSpace(filters.Query) != "" {
		pattern := fmt.Sprintf("%%%s%%", filters.Query)
		conditions = append(conditions, fmt.Sprintf("(statement ILIKE $%d OR context ILIKE $%d OR expected_outcome ILIKE $%d)", param, param+1, param+2))
		args = append(args, pattern, pattern, pattern)
		param += 3
	}

	if strings.TrimSpace(filters.Collaborator) != "" {
		conditions = append(conditions, fmt.Sprintf("EXISTS (SELECT 1 FROM jsonb_array_elements_text(collaborators) AS c WHERE LOWER(c) = LOWER($%d))", param))
		args = append(args, filters.Collaborator)
		param++
	}

	if filters.CreatedAfter != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", param))
		args = append(args, *filters.CreatedAfter)
		param++
	}

	if filters.CreatedBefore != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", param))
		args = append(args, *filters.CreatedBefore)
		param++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := "SELECT COUNT(*) FROM intents" + whereClause

	var total int
	if err := db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return IntentListResult{}, err
	}

	listQuery := "SELECT id, statement, context, expected_outcome, collaborators, created_at FROM intents" + whereClause + " ORDER BY created_at DESC"
	listArgs := append([]any{}, args...)

	if pagination.Limit > 0 {
		listQuery += fmt.Sprintf(" LIMIT $%d", param)
		listArgs = append(listArgs, pagination.Limit)
		param++
	}

	if pagination.Offset > 0 {
		listQuery += fmt.Sprintf(" OFFSET $%d", param)
		listArgs = append(listArgs, pagination.Offset)
	}

	rows, err := db.QueryContext(ctx, listQuery, listArgs...)
	if err != nil {
		return IntentListResult{}, err
	}
	defer rows.Close()

	intents := make([]Intent, 0)
	for rows.Next() {
		var (
			intent  Intent
			rawJSON []byte
		)

		if err := rows.Scan(&intent.ID, &intent.Statement, &intent.Context, &intent.ExpectedOutcome, &rawJSON, &intent.CreatedAt); err != nil {
			return IntentListResult{}, err
		}

		if len(rawJSON) > 0 {
			if err := json.Unmarshal(rawJSON, &intent.Collaborators); err != nil {
				return IntentListResult{}, err
			}
		}

		intents = append(intents, intent)
	}

	if err := rows.Err(); err != nil {
		return IntentListResult{}, err
	}

	return IntentListResult{Intents: intents, TotalCount: total}, nil
}
