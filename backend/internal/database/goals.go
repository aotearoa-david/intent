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

// Goal represents a chapter-level objective that guides intents.
type Goal struct {
	ID               uuid.UUID
	Title            string
	ClarityStatement string
	Constraints      []string
	SuccessCriteria  []string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// GoalInput captures the fields required to create or update a goal.
type GoalInput struct {
	Title            string
	ClarityStatement string
	Constraints      []string
	SuccessCriteria  []string
}

// GoalFilters captures optional filters applied when querying goals.
type GoalFilters struct {
	Query         string
	CreatedAfter  *time.Time
	CreatedBefore *time.Time
}

// GoalListResult represents the outcome of listing goals.
type GoalListResult struct {
	Goals      []Goal
	TotalCount int
}

// CreateGoal persists a new goal and returns the stored entity.
func CreateGoal(ctx context.Context, db *sql.DB, input GoalInput) (Goal, error) {
	if db == nil {
		return Goal{}, errors.New("database handle is nil")
	}

	constraintsJSON, err := json.Marshal(input.Constraints)
	if err != nil {
		return Goal{}, err
	}

	successJSON, err := json.Marshal(input.SuccessCriteria)
	if err != nil {
		return Goal{}, err
	}

	now := time.Now().UTC()
	id := uuid.New()

	const query = `
INSERT INTO goals (id, title, clarity_statement, constraints, success_criteria, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
`

	if _, err := db.ExecContext(ctx, query, id, input.Title, input.ClarityStatement, string(constraintsJSON), string(successJSON), now, now); err != nil {
		return Goal{}, err
	}

	return Goal{
		ID:               id,
		Title:            input.Title,
		ClarityStatement: input.ClarityStatement,
		Constraints:      input.Constraints,
		SuccessCriteria:  input.SuccessCriteria,
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
}

// GetGoal retrieves a goal by identifier.
func GetGoal(ctx context.Context, db *sql.DB, id uuid.UUID) (Goal, error) {
	if db == nil {
		return Goal{}, errors.New("database handle is nil")
	}

	const query = `
SELECT id, title, clarity_statement, constraints, success_criteria, created_at, updated_at
FROM goals
WHERE id = $1
`

	var (
		goal            Goal
		constraintsJSON []byte
		successCriteria []byte
	)

	err := db.QueryRowContext(ctx, query, id).Scan(
		&goal.ID,
		&goal.Title,
		&goal.ClarityStatement,
		&constraintsJSON,
		&successCriteria,
		&goal.CreatedAt,
		&goal.UpdatedAt,
	)
	if err != nil {
		return Goal{}, err
	}

	if len(constraintsJSON) > 0 {
		if err := json.Unmarshal(constraintsJSON, &goal.Constraints); err != nil {
			return Goal{}, err
		}
	}

	if len(successCriteria) > 0 {
		if err := json.Unmarshal(successCriteria, &goal.SuccessCriteria); err != nil {
			return Goal{}, err
		}
	}

	return goal, nil
}

// UpdateGoal updates an existing goal and returns the persisted entity.
func UpdateGoal(ctx context.Context, db *sql.DB, id uuid.UUID, input GoalInput) (Goal, error) {
	if db == nil {
		return Goal{}, errors.New("database handle is nil")
	}

	constraintsJSON, err := json.Marshal(input.Constraints)
	if err != nil {
		return Goal{}, err
	}

	successJSON, err := json.Marshal(input.SuccessCriteria)
	if err != nil {
		return Goal{}, err
	}

	now := time.Now().UTC()

	const query = `
UPDATE goals
SET title = $1,
    clarity_statement = $2,
    constraints = $3,
    success_criteria = $4,
    updated_at = $5
WHERE id = $6
RETURNING id, title, clarity_statement, constraints, success_criteria, created_at, updated_at
`

	var (
		goal           Goal
		rawConstraints []byte
		rawSuccess     []byte
	)

	err = db.QueryRowContext(ctx, query, input.Title, input.ClarityStatement, string(constraintsJSON), string(successJSON), now, id).Scan(
		&goal.ID,
		&goal.Title,
		&goal.ClarityStatement,
		&rawConstraints,
		&rawSuccess,
		&goal.CreatedAt,
		&goal.UpdatedAt,
	)
	if err != nil {
		return Goal{}, err
	}

	if len(rawConstraints) > 0 {
		if err := json.Unmarshal(rawConstraints, &goal.Constraints); err != nil {
			return Goal{}, err
		}
	}

	if len(rawSuccess) > 0 {
		if err := json.Unmarshal(rawSuccess, &goal.SuccessCriteria); err != nil {
			return Goal{}, err
		}
	}

	return goal, nil
}

// DeleteGoal removes a goal by identifier.
func DeleteGoal(ctx context.Context, db *sql.DB, id uuid.UUID) error {
	if db == nil {
		return errors.New("database handle is nil")
	}

	const query = `DELETE FROM goals WHERE id = $1`

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

// ListGoals returns goals applying optional filters and pagination.
func ListGoals(ctx context.Context, db *sql.DB, filters GoalFilters, pagination Pagination) (GoalListResult, error) {
	if db == nil {
		return GoalListResult{}, errors.New("database handle is nil")
	}

	var (
		conditions []string
		args       []any
		param      = 1
	)

	if strings.TrimSpace(filters.Query) != "" {
		pattern := fmt.Sprintf("%%%s%%", filters.Query)
		conditions = append(conditions, fmt.Sprintf("(title ILIKE $%d OR clarity_statement ILIKE $%d OR success_criteria::text ILIKE $%d)", param, param+1, param+2))
		args = append(args, pattern, pattern, pattern)
		param += 3
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

	countQuery := "SELECT COUNT(*) FROM goals" + whereClause

	var total int
	if err := db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return GoalListResult{}, err
	}

	listQuery := "SELECT id, title, clarity_statement, constraints, success_criteria, created_at, updated_at FROM goals" + whereClause + " ORDER BY created_at DESC"
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
		return GoalListResult{}, err
	}
	defer rows.Close()

	goals := make([]Goal, 0)
	for rows.Next() {
		var (
			goal           Goal
			rawConstraints []byte
			rawSuccess     []byte
		)

		if err := rows.Scan(
			&goal.ID,
			&goal.Title,
			&goal.ClarityStatement,
			&rawConstraints,
			&rawSuccess,
			&goal.CreatedAt,
			&goal.UpdatedAt,
		); err != nil {
			return GoalListResult{}, err
		}

		if len(rawConstraints) > 0 {
			if err := json.Unmarshal(rawConstraints, &goal.Constraints); err != nil {
				return GoalListResult{}, err
			}
		}

		if len(rawSuccess) > 0 {
			if err := json.Unmarshal(rawSuccess, &goal.SuccessCriteria); err != nil {
				return GoalListResult{}, err
			}
		}

		goals = append(goals, goal)
	}

	if err := rows.Err(); err != nil {
		return GoalListResult{}, err
	}

	return GoalListResult{Goals: goals, TotalCount: total}, nil
}
