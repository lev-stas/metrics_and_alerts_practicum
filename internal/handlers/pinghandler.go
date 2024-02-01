package handlers

import (
	"database/sql"
	"github.com/lev-stas/metricsmonitor.git/internal/logger"
	"net/http"
)

func PingHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method is not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := db.Ping(); err != nil {
			logger.Log.Errorw("Error during connecting to database", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
