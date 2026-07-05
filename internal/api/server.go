package api

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type URLSaver interface {
	SaveURL(ctx context.Context, originalURL string) (shortCode string, created bool, err error)
}

type URLGetter interface {
	GetURL(ctx context.Context, shortCode string) (originalURL string, err error)
}

type Store interface {
	URLSaver
	URLGetter
}

func NewServer(log *slog.Logger, baseURL string, store Store) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(requestLogger(log))

	r.Post("/url", handleSave(log, baseURL, store))
	r.Get("/url/{short_code}", handleGet(log, store))

	return r
}
