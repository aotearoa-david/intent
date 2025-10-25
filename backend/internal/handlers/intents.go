package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/example/intent/backend/internal/database"
)

type createIntentRequest struct {
	Statement       string   `json:"statement"`
	Context         string   `json:"context"`
	ExpectedOutcome string   `json:"expectedOutcome"`
	Collaborators   []string `json:"collaborators"`
}

type createIntentResponse struct {
	ID              string   `json:"id"`
	Statement       string   `json:"statement"`
	Context         string   `json:"context"`
	ExpectedOutcome string   `json:"expectedOutcome"`
	Collaborators   []string `json:"collaborators"`
	CreatedAt       string   `json:"createdAt"`
}

// CreateIntentHandler handles POST /api/intents requests.
func CreateIntentHandler(logger *slog.Logger, db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger.InfoContext(ctx, "create intent invoked")

		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		var payload createIntentRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			logger.WarnContext(ctx, "invalid intent payload", "error", err)
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if err := validateIntentPayload(payload); err != nil {
			logger.WarnContext(ctx, "intent validation failed", "error", err)
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}

		cleanedCollaborators := normalizeCollaborators(payload.Collaborators)

		record, err := database.CreateIntent(ctx, db, database.IntentInput{
			Statement:       strings.TrimSpace(payload.Statement),
			Context:         strings.TrimSpace(payload.Context),
			ExpectedOutcome: strings.TrimSpace(payload.ExpectedOutcome),
			Collaborators:   cleanedCollaborators,
		})
		if err != nil {
			logger.ErrorContext(ctx, "failed to persist intent", "error", err)
			writeJSONError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		response := createIntentResponse{
			ID:              record.ID.String(),
			Statement:       record.Statement,
			Context:         record.Context,
			ExpectedOutcome: record.ExpectedOutcome,
			Collaborators:   record.Collaborators,
			CreatedAt:       record.CreatedAt.Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			logger.ErrorContext(ctx, "failed to encode response", "error", err)
		}
	}
}

func validateIntentPayload(payload createIntentRequest) error {
	if strings.TrimSpace(payload.Statement) == "" {
		return errors.New("statement is required")
	}

	if strings.TrimSpace(payload.Context) == "" {
		return errors.New("context is required")
	}

	if strings.TrimSpace(payload.ExpectedOutcome) == "" {
		return errors.New("expectedOutcome is required")
	}

	return nil
}

func normalizeCollaborators(collaborators []string) []string {
	if len(collaborators) == 0 {
		return []string{}
	}

	cleaned := make([]string, 0, len(collaborators))
	seen := make(map[string]struct{})
	for _, collaborator := range collaborators {
		trimmed := strings.TrimSpace(collaborator)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		cleaned = append(cleaned, trimmed)
	}
	return cleaned
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
