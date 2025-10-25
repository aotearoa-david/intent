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
	"github.com/google/uuid"
)

type createIntentRequest struct {
	Statement       string   `json:"statement"`
	Context         string   `json:"context"`
	ExpectedOutcome string   `json:"expectedOutcome"`
	Collaborators   []string `json:"collaborators"`
}

type intentResponse struct {
	ID              string   `json:"id"`
	Statement       string   `json:"statement"`
	Context         string   `json:"context"`
	ExpectedOutcome string   `json:"expectedOutcome"`
	Collaborators   []string `json:"collaborators"`
	CreatedAt       string   `json:"createdAt"`
}

type listIntentResponse struct {
	Items      []intentResponse   `json:"items"`
	Pagination paginationResponse `json:"pagination"`
}

const (
	defaultPageSize = 20
	maxPageSize     = 100
)

type intentsHandler struct {
	logger *slog.Logger
	db     *sql.DB
}

// CreateIntentHandler handles POST /api/intents requests. Maintained for
// backwards compatibility with existing integration tests.
func CreateIntentHandler(logger *slog.Logger, db *sql.DB) http.HandlerFunc {
	handler := &intentsHandler{logger: logger, db: db}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		handler.handleCreate(w, r)
	}
}

// IntentsHandler routes CRUDL operations for intents.
func IntentsHandler(logger *slog.Logger, db *sql.DB) http.Handler {
	return &intentsHandler{logger: logger, db: db}
}

func (h *intentsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/api/intents":
		h.handleCreate(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/api/intents":
		h.handleList(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/intents/"):
		id := strings.TrimPrefix(r.URL.Path, "/api/intents/")
		if id == "" {
			http.NotFound(w, r)
			return
		}

		switch r.Method {
		case http.MethodGet:
			h.handleRetrieve(w, r, id)
		case http.MethodPut:
			h.handleUpdate(w, r, id)
		case http.MethodDelete:
			h.handleDelete(w, r, id)
		default:
			h.methodNotAllowed(w, http.MethodGet, http.MethodPut, http.MethodDelete)
		}
	case r.Method == http.MethodGet:
		h.handleList(w, r)
	default:
		h.methodNotAllowed(w, http.MethodGet, http.MethodPost)
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

func (h *intentsHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "create intent invoked")

	var payload createIntentRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.logger.WarnContext(ctx, "invalid intent payload", "error", err)
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validateIntentPayload(payload); err != nil {
		h.logger.WarnContext(ctx, "intent validation failed", "error", err)
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	cleanedCollaborators := normalizeCollaborators(payload.Collaborators)

	record, err := database.CreateIntent(ctx, h.db, database.IntentInput{
		Statement:       strings.TrimSpace(payload.Statement),
		Context:         strings.TrimSpace(payload.Context),
		ExpectedOutcome: strings.TrimSpace(payload.ExpectedOutcome),
		Collaborators:   cleanedCollaborators,
	})
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to persist intent", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response := toIntentResponse(record)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.ErrorContext(ctx, "failed to encode response", "error", err)
	}
}

func (h *intentsHandler) handleList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	page := parsePositiveInt(r.URL.Query().Get("page"), 1)
	if page < 1 {
		page = 1
	}

	pageSize := parsePositiveInt(r.URL.Query().Get("pageSize"), defaultPageSize)
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	filters := database.IntentFilters{
		Query:        strings.TrimSpace(r.URL.Query().Get("q")),
		Collaborator: strings.TrimSpace(r.URL.Query().Get("collaborator")),
	}

	if value := strings.TrimSpace(r.URL.Query().Get("createdAfter")); value != "" {
		ts, err := time.Parse(time.RFC3339, value)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "createdAfter must be RFC3339 timestamp")
			return
		}
		filters.CreatedAfter = &ts
	}

	if value := strings.TrimSpace(r.URL.Query().Get("createdBefore")); value != "" {
		ts, err := time.Parse(time.RFC3339, value)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "createdBefore must be RFC3339 timestamp")
			return
		}
		filters.CreatedBefore = &ts
	}

	offset := (page - 1) * pageSize

	result, err := database.ListIntents(ctx, h.db, filters, database.Pagination{Limit: pageSize, Offset: offset})
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to list intents", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	totalPages := 0
	if result.TotalCount > 0 && pageSize > 0 {
		totalPages = (result.TotalCount + pageSize - 1) / pageSize
	}

	responses := make([]intentResponse, 0, len(result.Intents))
	for _, intent := range result.Intents {
		responses = append(responses, toIntentResponse(intent))
	}

	payload := listIntentResponse{
		Items: responses,
		Pagination: paginationResponse{
			Page:       page,
			PageSize:   pageSize,
			TotalItems: result.TotalCount,
			TotalPages: totalPages,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		h.logger.ErrorContext(ctx, "failed to encode response", "error", err)
	}
}

func (h *intentsHandler) handleRetrieve(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	uuidValue, err := uuid.Parse(id)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid intent id")
		return
	}

	record, err := database.GetIntent(ctx, h.db, uuidValue)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "intent not found")
			return
		}
		h.logger.ErrorContext(ctx, "failed to retrieve intent", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(toIntentResponse(record)); err != nil {
		h.logger.ErrorContext(ctx, "failed to encode response", "error", err)
	}
}

func (h *intentsHandler) handleUpdate(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	uuidValue, err := uuid.Parse(id)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid intent id")
		return
	}

	var payload createIntentRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.logger.WarnContext(ctx, "invalid intent payload", "error", err)
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validateIntentPayload(payload); err != nil {
		h.logger.WarnContext(ctx, "intent validation failed", "error", err)
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	cleanedCollaborators := normalizeCollaborators(payload.Collaborators)

	record, err := database.UpdateIntent(ctx, h.db, uuidValue, database.IntentInput{
		Statement:       strings.TrimSpace(payload.Statement),
		Context:         strings.TrimSpace(payload.Context),
		ExpectedOutcome: strings.TrimSpace(payload.ExpectedOutcome),
		Collaborators:   cleanedCollaborators,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "intent not found")
			return
		}
		h.logger.ErrorContext(ctx, "failed to update intent", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(toIntentResponse(record)); err != nil {
		h.logger.ErrorContext(ctx, "failed to encode response", "error", err)
	}
}

func (h *intentsHandler) handleDelete(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	uuidValue, err := uuid.Parse(id)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid intent id")
		return
	}

	if err := database.DeleteIntent(ctx, h.db, uuidValue); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "intent not found")
			return
		}
		h.logger.ErrorContext(ctx, "failed to delete intent", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *intentsHandler) methodNotAllowed(w http.ResponseWriter, allowed ...string) {
	w.Header().Set("Allow", strings.Join(allowed, ", "))
	writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func toIntentResponse(intent database.Intent) intentResponse {
	return intentResponse{
		ID:              intent.ID.String(),
		Statement:       intent.Statement,
		Context:         intent.Context,
		ExpectedOutcome: intent.ExpectedOutcome,
		Collaborators:   intent.Collaborators,
		CreatedAt:       intent.CreatedAt.Format(time.RFC3339),
	}
}
