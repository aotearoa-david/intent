package handlers

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/example/intent/backend/internal/database"
)

// HelloHandler responds with a hello world payload enriched by a Postgres query.
func HelloHandler(logger *slog.Logger, db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger.InfoContext(ctx, "hello endpoint invoked")

		greeting, err := database.FetchGreeting(ctx, db)
		if err != nil {
			logger.ErrorContext(ctx, "failed to fetch greeting", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": greeting})
	}
}
