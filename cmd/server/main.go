package main

import (
	"database/sql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/lev-stas/metricsmonitor.git/internal/configs"
	"github.com/lev-stas/metricsmonitor.git/internal/gzipper"
	"github.com/lev-stas/metricsmonitor.git/internal/logger"
	"github.com/lev-stas/metricsmonitor.git/internal/metricsstorage"
	"github.com/lev-stas/metricsmonitor.git/internal/preloadworkers"
	"github.com/lev-stas/metricsmonitor.git/internal/routers"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

var storage *metricsstorage.MemStorage

func main() {
	if err := logger.LogInit(configs.ServerParams.LogLevel); err != nil {
		log.Fatalln(err)
	}
	configs.GetServerConfigs()
	storage = metricsstorage.NewMemStorage()

	writer, err := metricsstorage.NewFileWriter(&configs.ServerParams)
	if err != nil {
		logger.Log.Errorw("Error during creating New File Writer")
	}

	db, err := sql.Open("pgx", configs.ServerParams.DBConnect)
	if err != nil {
		logger.Log.Debugw("Can't connect to DB", "error", err)
	}
	defer db.Close()

	r := routers.RootRouter(storage, writer, db)
	if configs.ServerParams.InitLoad() {
		if err := preloadworkers.RestoreMetricsFromFile(storage); err != nil {
			logger.Log.Fatal("Error during reading metrics from file")
		}
	}

	if !configs.ServerParams.SyncSave() {
		preloadworkers.WriteMetricsTicker(storage, writer)
	}

	gshdwn := make(chan os.Signal, 1)

	signal.Notify(gshdwn, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := http.ListenAndServe(configs.ServerParams.Host, gzipper.GzipMiddleware(logger.RequestResponseLogger(r))); err != nil {
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	<-gshdwn

	if err = metricsstorage.SaveMetricsToFile(writer, storage); err != nil {
		logger.Log.Fatalw("Error saving during graceful shutdown", "error", err)
	}
	logger.Log.Infow("Server stopped gracefully")

	//log.Fatalln(http.ListenAndServe(configs.ServerParams.Host, gzipper.GzipMiddleware(logger.RequestResponseLogger(r))))
}
