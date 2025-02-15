package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"

	"github.com/basedalex/merch-shop/internal/config"
	"github.com/basedalex/merch-shop/internal/db"
	"github.com/basedalex/merch-shop/internal/middleware"
	"github.com/basedalex/merch-shop/internal/service"
	api "github.com/basedalex/merch-shop/internal/swagger"
	"github.com/go-chi/chi/v5"
	log "github.com/sirupsen/logrus"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cfg, err := config.Init("./config.dev.yaml")

	if err != nil {
		log.Error("error loading config ", err)
		return
	}

	database, err := db.NewPostgres(ctx, cfg.Database.DSN)
	if err != nil {
		log.Error(err)
		return
	}

	server := service.NewService(database)
	r := chi.NewRouter()
	r.Use(middleware.Authentication)
	api.HandlerFromMux(server, r)

	log.Println("Server listening on port 8080")
	err = http.ListenAndServe(":8080", r)
	if err != nil {
		log.Error(err)
		return
	}
}
