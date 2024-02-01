package routers

import (
	"database/sql"
	"github.com/go-chi/chi/v5"
	"github.com/lev-stas/metricsmonitor.git/internal/handlers"
	"github.com/lev-stas/metricsmonitor.git/internal/metricsstorage"
	"net/http"
	"strings"
)

func RootRouter(storage *metricsstorage.MemStorage, fileWriter *metricsstorage.FileWriter, db *sql.DB) http.Handler {
	r := chi.NewRouter()

	checkTypeMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			updatePath := "/update/"
			if strings.HasPrefix(r.URL.Path, updatePath) {
				parts := strings.Split(strings.TrimPrefix(r.URL.Path, updatePath), "/")
				if len(parts) > 1 && (parts[0] != "gauge" && parts[0] != "counter") {
					http.Error(w, "Invalid metric type", http.StatusBadRequest)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}

	r.Use(checkTypeMiddleware)

	r.Get("/", handlers.RootHandler(storage))
	r.Get("/ping", handlers.PingHandler(db))
	r.Post("/value/", handlers.ValueHandlerJSON(storage))
	r.Get("/value/{metricsType}/{metricsName}", handlers.ValueHandler(storage))
	r.Post("/update/", handlers.HandleUpdateJSON(storage, fileWriter))
	r.Post("/update/gauge/{metricsName}/{metricsValue}", handlers.HandleGaugeUpdate(storage, fileWriter))
	r.Post("/update/counter/{metricsName}/{metricsValue}", handlers.HandleCounterUpdate(storage, fileWriter))

	return r
}
