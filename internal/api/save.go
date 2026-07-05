package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
)

type saveRequest struct {
	URL string `json:"url" validate:"required,url"`
}

type saveResponse struct {
	ShortCode string `json:"short_code"`
	ShortURL  string `json:"short_url"`
}

var validate = validator.New()

func handleSave(log *slog.Logger, baseURL string, saver URLSaver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req saveRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if err := validate.Struct(req); err != nil {
			writeError(w, http.StatusBadRequest, "url must be a valid absolute URL")
			return
		}

		shortCode, created, err := saver.SaveURL(r.Context(), req.URL)
		if err != nil {
			log.Error("save url", slog.String("error", err.Error()))
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		status := http.StatusOK
		if created {
			status = http.StatusCreated
		}

		writeJSON(w, status, saveResponse{
			ShortCode: shortCode,
			ShortURL:  strings.TrimRight(baseURL, "/") + "/url/" + shortCode,
		})
	}
}
