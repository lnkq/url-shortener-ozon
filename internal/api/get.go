package api

import (
	"errors"
	"log/slog"
	"net/http"

	"url-shortener/internal/storage"

	"github.com/go-chi/chi/v5"
)

type getResponse struct {
	URL string `json:"url"`
}

func handleGet(log *slog.Logger, getter URLGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		shortCode := chi.URLParam(r, "short_code")

		originalURL, err := getter.GetURL(r.Context(), shortCode)
		switch {
		case errors.Is(err, storage.ErrURLNotFound):
			writeError(w, http.StatusNotFound, "short url not found")
			return
		case err != nil:
			log.Error("get url", slog.String("error", err.Error()))
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		writeJSON(w, http.StatusOK, getResponse{URL: originalURL})
	}
}
