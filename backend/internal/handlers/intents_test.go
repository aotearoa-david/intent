package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

func TestCreateIntentHandlerSuccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	logger := testLogger(t)

	payload := map[string]any{
		"statement":       "I intend to simplify our onboarding.",
		"context":         "New joiners are confused by the handbook.",
		"expectedOutcome": "A concise guide published in Confluence.",
		"collaborators":   []string{"Jamie", "Ana"},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	mock.ExpectExec("INSERT INTO intents").
		WithArgs(sqlmock.AnyArg(), payload["statement"], payload["context"], payload["expectedOutcome"], sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	req := httptest.NewRequest(http.MethodPost, "/api/intents", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	handler := CreateIntentHandler(logger, db)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rr.Code)
	}

	var response struct {
		ID              string   `json:"id"`
		Statement       string   `json:"statement"`
		Context         string   `json:"context"`
		ExpectedOutcome string   `json:"expectedOutcome"`
		Collaborators   []string `json:"collaborators"`
		CreatedAt       string   `json:"createdAt"`
	}

	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}

	if response.ID == "" {
		t.Error("expected response to include an ID")
	} else if _, err := uuid.Parse(response.ID); err != nil {
		t.Errorf("expected response ID to be a UUID: %v", err)
	}

	if response.Statement != payload["statement"] {
		t.Errorf("unexpected statement. got %q want %q", response.Statement, payload["statement"])
	}

	if response.Context != payload["context"] {
		t.Errorf("unexpected context. got %q want %q", response.Context, payload["context"])
	}

	if response.ExpectedOutcome != payload["expectedOutcome"] {
		t.Errorf("unexpected expectedOutcome. got %q want %q", response.ExpectedOutcome, payload["expectedOutcome"])
	}

	if len(response.Collaborators) != 2 {
		t.Fatalf("expected 2 collaborators got %d", len(response.Collaborators))
	}

	if _, err := time.Parse(time.RFC3339, response.CreatedAt); err != nil {
		t.Errorf("expected createdAt to be RFC3339: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestCreateIntentHandlerValidationError(t *testing.T) {
	db := &sql.DB{}
	logger := testLogger(t)

	payload := map[string]any{
		"statement":       "",
		"context":         "context",
		"expectedOutcome": "outcome",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/intents", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	handler := CreateIntentHandler(logger, db)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestCreateIntentHandlerMethodNotAllowed(t *testing.T) {
	db := &sql.DB{}
	logger := testLogger(t)

	req := httptest.NewRequest(http.MethodGet, "/api/intents", nil)
	rr := httptest.NewRecorder()

	handler := CreateIntentHandler(logger, db)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
	}

	if got := rr.Header().Get("Allow"); got != http.MethodPost {
		t.Fatalf("expected Allow header %q, got %q", http.MethodPost, got)
	}
}

func TestNormalizeCollaborators(t *testing.T) {
	input := []string{"  Jamie  ", "Ana", "JAMIE", ""}
	got := normalizeCollaborators(input)
	want := []string{"Jamie", "Ana"}

	if len(got) != len(want) {
		t.Fatalf("expected %d collaborators, got %d", len(want), len(got))
	}

	for i, collaborator := range want {
		if got[i] != collaborator {
			t.Errorf("unexpected collaborator at index %d. want %q got %q", i, collaborator, got[i])
		}
	}
}

func TestIntentsHandlerListSuccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	logger := testLogger(t)

	pattern := "%swarm%"
	createdAt := time.Now().UTC()
	id := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM intents WHERE (statement ILIKE $1 OR context ILIKE $2 OR expected_outcome ILIKE $3) AND EXISTS (SELECT 1 FROM jsonb_array_elements_text(collaborators) AS c WHERE LOWER(c) = LOWER($4))")).
		WithArgs(pattern, pattern, pattern, "Jamie").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, statement, context, expected_outcome, collaborators, created_at FROM intents WHERE (statement ILIKE $1 OR context ILIKE $2 OR expected_outcome ILIKE $3) AND EXISTS (SELECT 1 FROM jsonb_array_elements_text(collaborators) AS c WHERE LOWER(c) = LOWER($4)) ORDER BY created_at DESC LIMIT $5 OFFSET $6")).
		WithArgs(pattern, pattern, pattern, "Jamie", 5, 5).
		WillReturnRows(sqlmock.NewRows([]string{"id", "statement", "context", "expected_outcome", "collaborators", "created_at"}).
			AddRow(id, "statement", "context", "outcome", `["Jamie"]`, createdAt))

	req := httptest.NewRequest(http.MethodGet, "/api/intents?page=2&pageSize=5&q=swarm&collaborator=Jamie", nil)
	rr := httptest.NewRecorder()

	handler := IntentsHandler(logger, db)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var payload struct {
		Items []struct {
			ID            string   `json:"id"`
			Statement     string   `json:"statement"`
			Collaborators []string `json:"collaborators"`
		} `json:"items"`
		Pagination struct {
			Page       int `json:"page"`
			PageSize   int `json:"pageSize"`
			TotalItems int `json:"totalItems"`
			TotalPages int `json:"totalPages"`
		} `json:"pagination"`
	}

	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	if payload.Pagination.Page != 2 || payload.Pagination.PageSize != 5 {
		t.Fatalf("unexpected pagination: %+v", payload.Pagination)
	}

	if len(payload.Items) != 1 {
		t.Fatalf("expected 1 item got %d", len(payload.Items))
	}

	if payload.Items[0].ID == "" {
		t.Fatalf("expected ID in list item")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestIntentsHandlerRetrieveNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	logger := testLogger(t)
	id := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, statement, context, expected_outcome, collaborators, created_at FROM intents WHERE id = $1")).
		WithArgs(id).
		WillReturnError(sql.ErrNoRows)

	req := httptest.NewRequest(http.MethodGet, "/api/intents/"+id.String(), nil)
	rr := httptest.NewRecorder()

	handler := IntentsHandler(logger, db)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 got %d", rr.Code)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestIntentsHandlerUpdateSuccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	logger := testLogger(t)
	id := uuid.New()
	createdAt := time.Now().UTC()

	payload := map[string]any{
		"statement":       "Updated statement",
		"context":         "Context",
		"expectedOutcome": "Outcome",
		"collaborators":   []string{"Jamie"},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	mock.ExpectQuery(regexp.QuoteMeta(`UPDATE intents
SET statement = $1,
    context = $2,
    expected_outcome = $3,
    collaborators = $4
WHERE id = $5
RETURNING id, statement, context, expected_outcome, collaborators, created_at`)).
		WithArgs(payload["statement"], payload["context"], payload["expectedOutcome"], `["Jamie"]`, id).
		WillReturnRows(sqlmock.NewRows([]string{"id", "statement", "context", "expected_outcome", "collaborators", "created_at"}).
			AddRow(id, payload["statement"], payload["context"], payload["expectedOutcome"], `["Jamie"]`, createdAt))

	req := httptest.NewRequest(http.MethodPut, "/api/intents/"+id.String(), bytes.NewReader(body))
	rr := httptest.NewRecorder()

	handler := IntentsHandler(logger, db)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200 got %d", rr.Code)
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

func TestIntentsHandlerDeleteSuccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	logger := testLogger(t)
	id := uuid.New()

	mock.ExpectExec("DELETE FROM intents").
		WithArgs(id).
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := httptest.NewRequest(http.MethodDelete, "/api/intents/"+id.String(), nil)
	rr := httptest.NewRecorder()

	handler := IntentsHandler(logger, db)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status 204 got %d", rr.Code)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

// testLogger creates a slog.Logger that discards output during tests.
func testLogger(t *testing.T) *slog.Logger {
	t.Helper()
	handler := slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError})
	return slog.New(handler)
}
