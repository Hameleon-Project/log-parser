package main

import (
	"context"
	"log"
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

	db, err := storage.NewPostgresStorage(cfg.DBConn)
	if err != nil {
		log.Fatalf("Critical error: failed to connect to database: %v", err)
	}

	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}()

	repo := storage.NewPostgresRepo(db)
	svc := service.NewParserService(repo)
	pHandler := handler.NewParserHandler(svc)

	// настраиваем маршрутизацию
	mux := http.NewServeMux()

	mux.HandleFunc("/health", handler.HealthCheck)

	mux.HandleFunc("/parse", pHandler.Parse)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	// запуск сервера в отдельной горутине, чтобы не блокировал основной поток
	go func() {
		log.Printf("Starting Log-Parser server on port %s...", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)

	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	sign := <-stop
	log.Printf("Received signal: %v. Initiating graceful shutdown...", sign)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited properly")
}
