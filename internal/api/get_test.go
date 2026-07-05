package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"url-shortener/internal/storage"

	"github.com/go-chi/chi/v5"
)

type stubGetter struct {
	url string
	err error
}

func (s *stubGetter) GetURL(_ context.Context, _ string) (string, error) {
	return s.url, s.err
}

func mount(handler http.HandlerFunc) http.Handler {
	r := chi.NewRouter()
	r.Get("/url/{short_code}", handler)
	return r
}

func TestHandleGet_Found(t *testing.T) {
	handler := handleGet(discardLogger(), &stubGetter{url: "https://example.com/target"})
	router := mount(handler)

	req := httptest.NewRequest(http.MethodGet, "/url/AbC123XyZ_", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp getResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if resp.URL != "https://example.com/target" {
		t.Errorf("url got %q, want %q", resp.URL, "https://example.com/target")
	}
}

func TestHandleGet_NotFound(t *testing.T) {
	handler := handleGet(discardLogger(), &stubGetter{err: storage.ErrURLNotFound})
	router := mount(handler)

	req := httptest.NewRequest(http.MethodGet, "/url/doesNotExist", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestHandleGet_StorageError(t *testing.T) {
	handler := handleGet(discardLogger(), &stubGetter{err: errors.New("db is down")})
	router := mount(handler)

	req := httptest.NewRequest(http.MethodGet, "/url/AbC123XyZ_", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}
