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

func TestCreateGoalSuccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	input := GoalInput{
		Title:            "Boost release confidence",
		ClarityStatement: "Ensure Thursday release is risk-free",
		Constraints:      []string{"Keep production stable"},
		SuccessCriteria:  []string{"Zero Sev-1 incidents"},
	}

	mock.ExpectExec("INSERT INTO goals").
		WithArgs(sqlmock.AnyArg(), input.Title, input.ClarityStatement, `["Keep production stable"]`, `["Zero Sev-1 incidents"]`, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	goal, err := CreateGoal(context.Background(), db, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if goal.Title != input.Title {
		t.Fatalf("expected title %q got %q", input.Title, goal.Title)
	}

	if goal.ClarityStatement != input.ClarityStatement {
		t.Fatalf("expected clarity statement %q got %q", input.ClarityStatement, goal.ClarityStatement)
	}

	if goal.ID == uuid.Nil {
		t.Fatal("expected goal ID to be generated")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestGetGoalSuccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	id := uuid.New()
	createdAt := time.Now().UTC()
	updatedAt := createdAt.Add(time.Hour)

	rows := sqlmock.NewRows([]string{"id", "title", "clarity_statement", "constraints", "success_criteria", "created_at", "updated_at"}).
		AddRow(id, "Goal", "Clarity", `["Guardrail"]`, `["Outcome"]`, createdAt, updatedAt)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, title, clarity_statement, constraints, success_criteria, created_at, updated_at FROM goals WHERE id = $1")).
		WithArgs(id).
		WillReturnRows(rows)

	goal, err := GetGoal(context.Background(), db, id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if goal.ID != id {
		t.Fatalf("expected id %s got %s", id, goal.ID)
	}

	if len(goal.Constraints) != 1 || goal.Constraints[0] != "Guardrail" {
		t.Fatalf("expected constraints to be unmarshalled")
	}

	if len(goal.SuccessCriteria) != 1 || goal.SuccessCriteria[0] != "Outcome" {
		t.Fatalf("expected success criteria to be unmarshalled")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestGetGoalNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	id := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, title, clarity_statement, constraints, success_criteria, created_at, updated_at FROM goals WHERE id = $1")).
		WithArgs(id).
		WillReturnError(sql.ErrNoRows)

	if _, err := GetGoal(context.Background(), db, id); err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestUpdateGoalSuccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	id := uuid.New()
	createdAt := time.Now().UTC()
	updatedAt := createdAt.Add(time.Hour)

	input := GoalInput{
		Title:            "Refine onboarding",
		ClarityStatement: "Make onboarding consistent",
		Constraints:      []string{"Stay within budget"},
		SuccessCriteria:  []string{"Handbook updated"},
	}

	mock.ExpectQuery(regexp.QuoteMeta(`UPDATE goals
SET title = $1,
    clarity_statement = $2,
    constraints = $3,
    success_criteria = $4,
    updated_at = $5
WHERE id = $6
RETURNING id, title, clarity_statement, constraints, success_criteria, created_at, updated_at`)).
		WithArgs(input.Title, input.ClarityStatement, `["Stay within budget"]`, `["Handbook updated"]`, sqlmock.AnyArg(), id).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "clarity_statement", "constraints", "success_criteria", "created_at", "updated_at"}).
			AddRow(id, input.Title, input.ClarityStatement, `["Stay within budget"]`, `["Handbook updated"]`, createdAt, updatedAt))

	goal, err := UpdateGoal(context.Background(), db, id, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if goal.Title != input.Title {
		t.Fatalf("expected title %q got %q", input.Title, goal.Title)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestDeleteGoal(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	id := uuid.New()

	mock.ExpectExec("DELETE FROM goals").
		WithArgs(id).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := DeleteGoal(context.Background(), db, id); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestDeleteGoalNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	id := uuid.New()

	mock.ExpectExec("DELETE FROM goals").
		WithArgs(id).
		WillReturnResult(sqlmock.NewResult(0, 0))

	if err := DeleteGoal(context.Background(), db, id); err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestListGoalsWithFilters(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	now := time.Now().UTC()
	filters := GoalFilters{Query: "Clarity", CreatedAfter: &now}
	pagination := Pagination{Limit: 5, Offset: 5}

	id := uuid.New()
	createdAt := now
	updatedAt := now
	pattern := "%Clarity%"

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM goals WHERE (title ILIKE $1 OR clarity_statement ILIKE $2 OR success_criteria::text ILIKE $3) AND created_at >= $4")).
		WithArgs(pattern, pattern, pattern, now).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, title, clarity_statement, constraints, success_criteria, created_at, updated_at FROM goals WHERE (title ILIKE $1 OR clarity_statement ILIKE $2 OR success_criteria::text ILIKE $3) AND created_at >= $4 ORDER BY created_at DESC LIMIT $5 OFFSET $6")).
		WithArgs(pattern, pattern, pattern, now, pagination.Limit, pagination.Offset).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "clarity_statement", "constraints", "success_criteria", "created_at", "updated_at"}).
			AddRow(id, "Goal", "Clarity", `["Guardrail"]`, `["Outcome"]`, createdAt, updatedAt))

	result, err := ListGoals(context.Background(), db, filters, pagination)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalCount != 1 {
		t.Fatalf("expected total count 1 got %d", result.TotalCount)
	}

	if len(result.Goals) != 1 {
		t.Fatalf("expected 1 goal got %d", len(result.Goals))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}
