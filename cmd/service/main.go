package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/basedalex/merch-shop/internal/config"
	"github.com/basedalex/merch-shop/internal/db"
	"github.com/basedalex/merch-shop/internal/middleware"
	"github.com/basedalex/merch-shop/internal/service"
	api "github.com/basedalex/merch-shop/internal/swagger"
	"github.com/go-chi/chi/v5"
	log "github.com/sirupsen/logrus"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cfg, err := config.Init("./config.dev.yaml")
	if err != nil {
		log.Fatal("Error loading config: ", err)
		return
	}

	maxAttempts := 10
	delay := 1 * time.Second

	var database *db.Postgres

	for i := 0; i < maxAttempts; i++ {
		database, err = db.NewPostgres(ctx, cfg)
		if err == nil {
			log.Info("connected to database")
			break
		}
		log.Warnf("database not ready, attempt %d/%d: %v", i+1, maxAttempts, err)
		time.Sleep(delay)
		err = nil
	}

	if err != nil {
		log.Fatal("Error connecting to database: ", err)
		return
	}

	server := service.NewService(database)
	r := chi.NewRouter()
	r.Use(middleware.Authentication)
	api.HandlerFromMux(server, r)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	go func() {
		log.Println("Server listening on port 8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("ListenAndServe Error: ", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	log.Println("Shutting down server...")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal("Server Shutdown Error: ", err)
	}
	log.Println("Server gracefully stopped")
}