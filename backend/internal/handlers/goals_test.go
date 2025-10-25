package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

func TestGoalsHandlerCreateSuccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	logger := testLogger(t)

	payload := map[string]any{
		"title":            "Improve release readiness",
		"clarityStatement": "Thursday release keeps slipping due to missing evidence",
		"constraints":      []string{"Protect member focus time", " protect member focus time "},
		"successCriteria":  []string{"Checklist published"},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	mock.ExpectExec("INSERT INTO goals").
		WithArgs(sqlmock.AnyArg(), payload["title"], payload["clarityStatement"], `["Protect member focus time"]`, `["Checklist published"]`, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	req := httptest.NewRequest(http.MethodPost, "/api/goals", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	handler := GoalsHandler(logger, db)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status %d got %d", http.StatusCreated, rr.Code)
	}

	var response struct {
		ID               string   `json:"id"`
		Title            string   `json:"title"`
		ClarityStatement string   `json:"clarityStatement"`
		Constraints      []string `json:"constraints"`
		SuccessCriteria  []string `json:"successCriteria"`
		CreatedAt        string   `json:"createdAt"`
		UpdatedAt        string   `json:"updatedAt"`
	}

	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}

	if response.ID == "" {
		t.Fatal("expected response to include an ID")
	}

	if len(response.Constraints) != 1 || response.Constraints[0] != "Protect member focus time" {
		t.Fatalf("expected constraints to be normalized")
	}

	if len(response.SuccessCriteria) != 1 || response.SuccessCriteria[0] != "Checklist published" {
		t.Fatalf("expected success criteria to round-trip")
	}

	if _, err := time.Parse(time.RFC3339, response.CreatedAt); err != nil {
		t.Fatalf("expected createdAt to be RFC3339: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestGoalsHandlerCreateValidationError(t *testing.T) {
	db := &sql.DB{}
	logger := testLogger(t)

	payload := map[string]any{
		"title":            "",
		"clarityStatement": "",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/goals", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	handler := GoalsHandler(logger, db)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestGoalsHandlerListSuccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	logger := testLogger(t)

	pattern := "%focus%"
	createdAt := time.Now().UTC()
	updatedAt := createdAt
	id := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM goals WHERE (title ILIKE $1 OR clarity_statement ILIKE $2 OR success_criteria::text ILIKE $3)")).
		WithArgs(pattern, pattern, pattern).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, title, clarity_statement, constraints, success_criteria, created_at, updated_at FROM goals WHERE (title ILIKE $1 OR clarity_statement ILIKE $2 OR success_criteria::text ILIKE $3) ORDER BY created_at DESC LIMIT $4")).
		WithArgs(pattern, pattern, pattern, 20).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "clarity_statement", "constraints", "success_criteria", "created_at", "updated_at"}).
			AddRow(id, "Goal", "Clarity", `["Guardrail"]`, `["Outcome"]`, createdAt, updatedAt))

	req := httptest.NewRequest(http.MethodGet, "/api/goals?q=focus", nil)
	rr := httptest.NewRecorder()

	handler := GoalsHandler(logger, db)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d got %d", http.StatusOK, rr.Code)
	}

	var payload struct {
		Items []struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"items"`
		Pagination struct {
			Page     int `json:"page"`
			PageSize int `json:"pageSize"`
		} `json:"pagination"`
	}

	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	if len(payload.Items) != 1 {
		t.Fatalf("expected 1 item got %d", len(payload.Items))
	}

	if payload.Pagination.Page != 1 || payload.Pagination.PageSize != 20 {
		t.Fatalf("unexpected pagination metadata: %+v", payload.Pagination)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestGoalsHandlerRetrieveNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	logger := testLogger(t)
	id := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, title, clarity_statement, constraints, success_criteria, created_at, updated_at FROM goals WHERE id = $1")).
		WithArgs(id).
		WillReturnError(sql.ErrNoRows)

	req := httptest.NewRequest(http.MethodGet, "/api/goals/"+id.String(), nil)
	rr := httptest.NewRecorder()

	handler := GoalsHandler(logger, db)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status %d got %d", http.StatusNotFound, rr.Code)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestGoalsHandlerUpdateSuccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	logger := testLogger(t)
	id := uuid.New()
	createdAt := time.Now().UTC().Add(-time.Hour)
	updatedAt := time.Now().UTC()

	payload := map[string]any{
		"title":            "Updated goal",
		"clarityStatement": "Updated clarity",
		"constraints":      []string{"Guardrail"},
		"successCriteria":  []string{"Outcome"},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	mock.ExpectQuery(regexp.QuoteMeta(`UPDATE goals
SET title = $1,
    clarity_statement = $2,
    constraints = $3,
    success_criteria = $4,
    updated_at = $5
WHERE id = $6
RETURNING id, title, clarity_statement, constraints, success_criteria, created_at, updated_at`)).
		WithArgs(payload["title"], payload["clarityStatement"], `["Guardrail"]`, `["Outcome"]`, sqlmock.AnyArg(), id).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "clarity_statement", "constraints", "success_criteria", "created_at", "updated_at"}).
			AddRow(id, payload["title"], payload["clarityStatement"], `["Guardrail"]`, `["Outcome"]`, createdAt, updatedAt))

	req := httptest.NewRequest(http.MethodPut, "/api/goals/"+id.String(), bytes.NewReader(body))
	rr := httptest.NewRecorder()

	handler := GoalsHandler(logger, db)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d got %d", http.StatusOK, rr.Code)
	}

	var response struct {
		ID string `json:"id"`
	}

	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	if response.ID != id.String() {
		t.Fatalf("expected id %s got %s", id, response.ID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestGoalsHandlerDeleteSuccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	logger := testLogger(t)
	id := uuid.New()

	mock.ExpectExec("DELETE FROM goals").
		WithArgs(id).
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := httptest.NewRequest(http.MethodDelete, "/api/goals/"+id.String(), nil)
	rr := httptest.NewRecorder()

	handler := GoalsHandler(logger, db)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status %d got %d", http.StatusNoContent, rr.Code)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestNormalizeGoalValues(t *testing.T) {
	input := []string{" focus ", "FOCUS", "", "Guard"}
	got := normalizeGoalValues(input)
	want := []string{"focus", "Guard"}

	if len(got) != len(want) {
		t.Fatalf("expected %d values got %d", len(want), len(got))
	}

	if got[0] != want[0] {
		t.Fatalf("unexpected first value: %q", got[0])
	}

	if got[1] != want[1] {
		t.Fatalf("unexpected second value: %q", got[1])
	}
}
