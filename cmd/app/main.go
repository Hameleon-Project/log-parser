package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"log-parser/internal/config"
	"log-parser/internal/handler"
	"log-parser/internal/service"
	"log-parser/internal/storage"

	_ "github.com/lib/pq"
)

func main() {
	cfg := config.Load()

	logHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel})
	slog.SetDefault(slog.New(logHandler))

	db, err := storage.NewPostgresStorage(cfg.DBConn)
	if err != nil {
		slog.Error("db_connect_failed", "err", err)
		os.Exit(1)
	}
	defer func() { _ = db.Close() }()

	repo := storage.NewPostgresRepo(db)
	svc := service.NewParserService(repo)
	pHandler := handler.NewParserHandler(svc)

	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/parse/", pHandler.Parse)
	mux.HandleFunc("GET /api/v1/topology/{log_id}", pHandler.GetTopology)
	mux.HandleFunc("GET /api/v1/node/{node_id}", pHandler.GetNode)
	mux.HandleFunc("GET /api/v1/port/{node_id}", pHandler.GetPorts)
	mux.HandleFunc("GET /api/v1/log/{log_id}", pHandler.GetLogMeta)

	mux.HandleFunc("GET /health", handler.HealthCheck)

	wrappedMux := handler.LoggingMiddleware(mux)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      wrappedMux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
	}

	go func() {
		slog.Info("server_listen", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server_failed", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	sig := <-stop
	slog.Info("shutdown_signal", "signal", sig.String())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("shutdown_error", "err", err)
	}
	slog.Info("server_stopped")
}
