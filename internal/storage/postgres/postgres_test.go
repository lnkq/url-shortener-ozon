package postgres_test

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"

	"url-shortener/internal/storage"
	"url-shortener/internal/storage/postgres"
)

func newTestStorage(t *testing.T) *postgres.Storage {
	t.Helper()

	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN not set")
	}

	st, err := postgres.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("postgres.New: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	return st
}

func TestSaveAndGetURL(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	url := "https://example.com/postgres-test-" + t.Name()

	code, created, err := st.SaveURL(ctx, url)
	if err != nil {
		t.Fatalf("SaveURL: %v", err)
	}
	if !created {
		t.Errorf("first save should create the url")
	}

	got, err := st.GetURL(ctx, code)
	if err != nil {
		t.Fatalf("GetURL: %v", err)
	}
	if got != url {
		t.Errorf("got %q, want %q", got, url)
	}
}

func TestSaveURL_SameURLIsIdempotent(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	url := "https://example.com/idempotent-test-" + t.Name()

	code1, _, err := st.SaveURL(ctx, url)
	if err != nil {
		t.Fatalf("first SaveURL: %v", err)
	}

	code2, created2, err := st.SaveURL(ctx, url)
	if err != nil {
		t.Fatalf("second SaveURL: %v", err)
	}
	if created2 {
		t.Errorf("second save should not create a new code")
	}
	if code1 != code2 {
		t.Errorf("same url gave different codes: %q vs %q", code1, code2)
	}
}

func TestGetURL_NotFound(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	_, err := st.GetURL(ctx, "0000000000")
	if !errors.Is(err, storage.ErrURLNotFound) {
		t.Errorf("got %v, want storage.ErrURLNotFound", err)
	}
}

func TestConcurrentSaveSameURL(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	url := "https://example.com/concurrent-test-" + t.Name()
	const goroutines = 50
	codes := make([]string, goroutines)

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer wg.Done()
			code, _, err := st.SaveURL(ctx, url)
			if err != nil {
				t.Errorf("SaveURL: %v", err)
				return
			}
			codes[i] = code
		}()
	}
	wg.Wait()

	first := codes[0]
	for i, code := range codes {
		if code != first {
			t.Errorf("goroutine %d got %q, want %q", i, code, first)
		}
	}
}
