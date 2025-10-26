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

type createGoalRequest struct {
	Title            string   `json:"title"`
	ClarityStatement string   `json:"clarityStatement"`
	Guardrails       []string `json:"guardrails"`
	DecisionRights   []string `json:"decisionRights"`
	Constraints      []string `json:"constraints"`
	SuccessCriteria  []string `json:"successCriteria"`
}

type goalResponse struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	ClarityStatement string   `json:"clarityStatement"`
	Guardrails       []string `json:"guardrails"`
	DecisionRights   []string `json:"decisionRights"`
	Constraints      []string `json:"constraints"`
	SuccessCriteria  []string `json:"successCriteria"`
	CreatedAt        string   `json:"createdAt"`
	UpdatedAt        string   `json:"updatedAt"`
}

type listGoalResponse struct {
	Items      []goalResponse     `json:"items"`
	Pagination paginationResponse `json:"pagination"`
}

type goalsHandler struct {
	logger *slog.Logger
	db     *sql.DB
}

// GoalsHandler routes CRUDL operations for goals.
func GoalsHandler(logger *slog.Logger, db *sql.DB) http.Handler {
	return &goalsHandler{logger: logger, db: db}
}

func (h *goalsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/api/goals":
		h.handleCreate(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/api/goals":
		h.handleList(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/goals/"):
		id := strings.TrimPrefix(r.URL.Path, "/api/goals/")
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
	default:
		h.methodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func validateGoalPayload(payload createGoalRequest) error {
	if strings.TrimSpace(payload.Title) == "" {
		return errors.New("title is required")
	}

	if strings.TrimSpace(payload.ClarityStatement) == "" {
		return errors.New("clarityStatement is required")
	}

	return nil
}

func normalizeGoalValues(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}

	cleaned := make([]string, 0, len(values))
	seen := make(map[string]struct{})
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
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

func (h *goalsHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "create goal invoked")

	var payload createGoalRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.logger.WarnContext(ctx, "invalid goal payload", "error", err)
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validateGoalPayload(payload); err != nil {
		h.logger.WarnContext(ctx, "goal validation failed", "error", err)
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	cleanedGuardrails := normalizeGoalValues(payload.Guardrails)
	cleanedDecisionRights := normalizeGoalValues(payload.DecisionRights)
	cleanedConstraints := normalizeGoalValues(payload.Constraints)
	cleanedSuccess := normalizeGoalValues(payload.SuccessCriteria)

	record, err := database.CreateGoal(ctx, h.db, database.GoalInput{
		Title:            strings.TrimSpace(payload.Title),
		ClarityStatement: strings.TrimSpace(payload.ClarityStatement),
		Guardrails:       cleanedGuardrails,
		DecisionRights:   cleanedDecisionRights,
		Constraints:      cleanedConstraints,
		SuccessCriteria:  cleanedSuccess,
	})
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to persist goal", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response := toGoalResponse(record)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.ErrorContext(ctx, "failed to encode response", "error", err)
	}
}

func (h *goalsHandler) handleList(w http.ResponseWriter, r *http.Request) {
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

	filters := database.GoalFilters{
		Query: strings.TrimSpace(r.URL.Query().Get("q")),
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

	result, err := database.ListGoals(ctx, h.db, filters, database.Pagination{Limit: pageSize, Offset: offset})
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to list goals", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	totalPages := 0
	if result.TotalCount > 0 && pageSize > 0 {
		totalPages = (result.TotalCount + pageSize - 1) / pageSize
	}

	responses := make([]goalResponse, 0, len(result.Goals))
	for _, goal := range result.Goals {
		responses = append(responses, toGoalResponse(goal))
	}

	payload := listGoalResponse{
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

func (h *goalsHandler) handleRetrieve(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	uuidValue, err := uuid.Parse(id)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid goal id")
		return
	}

	record, err := database.GetGoal(ctx, h.db, uuidValue)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "goal not found")
			return
		}
		h.logger.ErrorContext(ctx, "failed to retrieve goal", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(toGoalResponse(record)); err != nil {
		h.logger.ErrorContext(ctx, "failed to encode response", "error", err)
	}
}

func (h *goalsHandler) handleUpdate(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	uuidValue, err := uuid.Parse(id)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid goal id")
		return
	}

	var payload createGoalRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.logger.WarnContext(ctx, "invalid goal payload", "error", err)
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validateGoalPayload(payload); err != nil {
		h.logger.WarnContext(ctx, "goal validation failed", "error", err)
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	cleanedGuardrails := normalizeGoalValues(payload.Guardrails)
	cleanedDecisionRights := normalizeGoalValues(payload.DecisionRights)
	cleanedConstraints := normalizeGoalValues(payload.Constraints)
	cleanedSuccess := normalizeGoalValues(payload.SuccessCriteria)

	record, err := database.UpdateGoal(ctx, h.db, uuidValue, database.GoalInput{
		Title:            strings.TrimSpace(payload.Title),
		ClarityStatement: strings.TrimSpace(payload.ClarityStatement),
		Guardrails:       cleanedGuardrails,
		DecisionRights:   cleanedDecisionRights,
		Constraints:      cleanedConstraints,
		SuccessCriteria:  cleanedSuccess,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "goal not found")
			return
		}
		h.logger.ErrorContext(ctx, "failed to update goal", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(toGoalResponse(record)); err != nil {
		h.logger.ErrorContext(ctx, "failed to encode response", "error", err)
	}
}

func (h *goalsHandler) handleDelete(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	uuidValue, err := uuid.Parse(id)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid goal id")
		return
	}

	if err := database.DeleteGoal(ctx, h.db, uuidValue); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "goal not found")
			return
		}
		h.logger.ErrorContext(ctx, "failed to delete goal", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *goalsHandler) methodNotAllowed(w http.ResponseWriter, allowed ...string) {
	w.Header().Set("Allow", strings.Join(allowed, ", "))
	writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func toGoalResponse(goal database.Goal) goalResponse {
	return goalResponse{
		ID:               goal.ID.String(),
		Title:            goal.Title,
		ClarityStatement: goal.ClarityStatement,
		Guardrails:       goal.Guardrails,
		DecisionRights:   goal.DecisionRights,
		Constraints:      goal.Constraints,
		SuccessCriteria:  goal.SuccessCriteria,
		CreatedAt:        goal.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        goal.UpdatedAt.Format(time.RFC3339),
	}
}
