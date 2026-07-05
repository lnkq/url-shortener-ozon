package memory

import (
	"context"
	"sync"
	"sync/atomic"

	"url-shortener/internal/shortcode"
	"url-shortener/internal/storage"
)

type Storage struct {
	mu            sync.RWMutex
	byShortCode   map[string]string
	byOriginalURL map[string]string
	nextID        atomic.Uint64
}

func New() *Storage {
	return &Storage{
		byShortCode:   make(map[string]string),
		byOriginalURL: make(map[string]string),
	}
}

func (s *Storage) SaveURL(_ context.Context, originalURL string) (code string, created bool, err error) {
	s.mu.RLock()
	if existing, ok := s.byOriginalURL[originalURL]; ok {
		s.mu.RUnlock()
		return existing, false, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.byOriginalURL[originalURL]; ok {
		return existing, false, nil
	}

	id := s.nextID.Add(1)
	code = shortcode.Encode(id)

	s.byShortCode[code] = originalURL
	s.byOriginalURL[originalURL] = code

	return code, true, nil
}

func (s *Storage) GetURL(_ context.Context, shortCode string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	originalURL, ok := s.byShortCode[shortCode]
	if !ok {
		return "", storage.ErrURLNotFound
	}
	return originalURL, nil
}
