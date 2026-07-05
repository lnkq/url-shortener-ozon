package memory_test

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"testing"

	"url-shortener/internal/storage"
	"url-shortener/internal/storage/memory"
)

func TestSaveAndGetURL(t *testing.T) {
	ctx := context.Background()
	s := memory.New()

	code, created, err := s.SaveURL(ctx, "https://example.com")
	if err != nil {
		t.Fatalf("SaveURL: %v", err)
	}
	if !created {
		t.Errorf("first save should create the url")
	}

	got, err := s.GetURL(ctx, code)
	if err != nil {
		t.Fatalf("GetURL: %v", err)
	}
	if got != "https://example.com" {
		t.Errorf("got %q, want %q", got, "https://example.com")
	}
}

func TestSaveURL_SameURLIsIdempotent(t *testing.T) {
	ctx := context.Background()
	s := memory.New()

	code1, created1, _ := s.SaveURL(ctx, "https://example.com/page")
	if !created1 {
		t.Errorf("first save should create the url")
	}

	code2, created2, _ := s.SaveURL(ctx, "https://example.com/page")
	if created2 {
		t.Errorf("second save should not create a new code")
	}
	if code1 != code2 {
		t.Errorf("same url gave different codes: %q vs %q", code1, code2)
	}
}

func TestSaveURL_DifferentURLsGetDifferentCodes(t *testing.T) {
	ctx := context.Background()
	s := memory.New()

	codeA, _, _ := s.SaveURL(ctx, "https://a.example.com")
	codeB, _, _ := s.SaveURL(ctx, "https://b.example.com")

	if codeA == codeB {
		t.Errorf("different urls got the same code %q", codeA)
	}
}

func TestGetURL_NotFound(t *testing.T) {
	ctx := context.Background()
	s := memory.New()

	_, err := s.GetURL(ctx, "doesNotExist")
	if !errors.Is(err, storage.ErrURLNotFound) {
		t.Errorf("got %v, want storage.ErrURLNotFound", err)
	}
}

func TestConcurrentSaveSameURL(t *testing.T) {
	ctx := context.Background()
	s := memory.New()

	const goroutines = 200
	codes := make([]string, goroutines)

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer wg.Done()
			code, _, err := s.SaveURL(ctx, "https://example.com")
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

func TestConcurrentSaveDistinctURLs(t *testing.T) {
	ctx := context.Background()
	s := memory.New()

	const goroutines = 200
	codes := make([]string, goroutines)

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer wg.Done()
			url := "https://distinct.example.com/" + strconv.Itoa(i)
			code, _, err := s.SaveURL(ctx, url)
			if err != nil {
				t.Errorf("SaveURL: %v", err)
				return
			}
			codes[i] = code
		}()
	}
	wg.Wait()

	seen := make(map[string]bool, goroutines)
	for _, code := range codes {
		if seen[code] {
			t.Errorf("duplicate code %q", code)
		}
		seen[code] = true
	}
}
