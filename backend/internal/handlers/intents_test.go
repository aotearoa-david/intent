package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
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

// testLogger creates a slog.Logger that discards output during tests.
func testLogger(t *testing.T) *slog.Logger {
	t.Helper()
	handler := slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError})
	return slog.New(handler)
}
