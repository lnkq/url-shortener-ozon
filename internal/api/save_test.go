package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type stubSaver struct {
	shortCode string
	created   bool
	err       error
	gotURL    string
}

func (s *stubSaver) SaveURL(_ context.Context, originalURL string) (string, bool, error) {
	s.gotURL = originalURL
	return s.shortCode, s.created, s.err
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func doSave(handler http.HandlerFunc, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/url", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestHandleSave_NewURL_Returns201(t *testing.T) {
	saver := &stubSaver{shortCode: "AbC123XyZ_", created: true}
	handler := handleSave(discardLogger(), "http://localhost:8080", saver)

	rec := doSave(handler, `{"url":"https://example.com"}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("got status %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var resp saveResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if resp.ShortCode != "AbC123XyZ_" {
		t.Errorf("short code got %q, want %q", resp.ShortCode, "AbC123XyZ_")
	}
	if want := "http://localhost:8080/url/AbC123XyZ_"; resp.ShortURL != want {
		t.Errorf("short url got %q, want %q", resp.ShortURL, want)
	}
	if saver.gotURL != "https://example.com" {
		t.Errorf("save url got %q, want %q", saver.gotURL, "https://example.com")
	}
}

func TestHandleSave_ExistingURL_Returns200(t *testing.T) {
	saver := &stubSaver{shortCode: "AbC123XyZ_", created: false}
	handler := handleSave(discardLogger(), "http://localhost:8080", saver)

	rec := doSave(handler, `{"url":"https://example.com"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHandleSave_InvalidJSON_Returns400(t *testing.T) {
	handler := handleSave(discardLogger(), "http://localhost:8080", &stubSaver{})

	rec := doSave(handler, `not json`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleSave_MissingURL_Returns400(t *testing.T) {
	handler := handleSave(discardLogger(), "http://localhost:8080", &stubSaver{})

	rec := doSave(handler, `{}`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleSave_InvalidURL_Returns400(t *testing.T) {
	handler := handleSave(discardLogger(), "http://localhost:8080", &stubSaver{})

	rec := doSave(handler, `{"url":"not-a-url"}`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleSave_StorageError_Returns500(t *testing.T) {
	handler := handleSave(discardLogger(), "http://localhost:8080", &stubSaver{err: errors.New("db is down")})

	rec := doSave(handler, `{"url":"https://example.com"}`)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}
