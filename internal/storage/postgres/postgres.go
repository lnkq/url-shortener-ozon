package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"url-shortener/internal/shortcode"
	"url-shortener/internal/storage"

	_ "github.com/lib/pq"
)

const (
	readyAttempts = 10
	readyInterval = time.Second
)

type Storage struct {
	db *sql.DB
}

func New(ctx context.Context, dsn string) (*Storage, error) {
	const op = "postgres.New"

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("%s: open: %w", op, err)
	}

	db.SetConnMaxLifetime(5 * time.Minute)

	if err := waitReady(ctx, db); err != nil {
		db.Close()
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if err := migrate(ctx, db); err != nil {
		db.Close()
		return nil, fmt.Errorf("%s: migrate: %w", op, err)
	}

	return &Storage{db: db}, nil
}

func waitReady(ctx context.Context, db *sql.DB) error {
	ticker := time.NewTicker(readyInterval)
	defer ticker.Stop()

	var err error
	for attempt := 0; attempt < readyAttempts; attempt++ {
		if err = db.PingContext(ctx); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
	return fmt.Errorf("postgres unreachable after %d attempts: %w", readyAttempts, err)
}

func migrate(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE SEQUENCE IF NOT EXISTS urls_id_seq;

		CREATE TABLE IF NOT EXISTS urls (
			id           BIGINT PRIMARY KEY DEFAULT nextval('urls_id_seq'),
			original_url TEXT NOT NULL UNIQUE,
			short_code   VARCHAR(10) NOT NULL UNIQUE,
			created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
		);
	`)
	return err
}

func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) SaveURL(ctx context.Context, originalURL string) (code string, created bool, err error) {
	const op = "postgres.SaveURL"

	var id int64
	if err := s.db.QueryRowContext(ctx, `SELECT nextval('urls_id_seq')`).Scan(&id); err != nil {
		return "", false, fmt.Errorf("%s: next id: %w", op, err)
	}

	candidate := shortcode.Encode(uint64(id))

	var saved string
	err = s.db.QueryRowContext(ctx, `
		INSERT INTO urls (id, original_url, short_code)
		VALUES ($1, $2, $3)
		ON CONFLICT (original_url) DO NOTHING
		RETURNING short_code
	`, id, originalURL, candidate).Scan(&saved)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		if err := s.db.QueryRowContext(ctx, `
			SELECT short_code FROM urls WHERE original_url = $1
		`, originalURL).Scan(&saved); err != nil {
			return "", false, fmt.Errorf("%s: lookup: %w", op, err)
		}
		return saved, false, nil
	case err != nil:
		return "", false, fmt.Errorf("%s: insert: %w", op, err)
	default:
		return saved, true, nil
	}
}

func (s *Storage) GetURL(ctx context.Context, shortCode string) (string, error) {
	const op = "postgres.GetURL"

	var originalURL string
	err := s.db.QueryRowContext(ctx, `
		SELECT original_url FROM urls WHERE short_code = $1
	`, shortCode).Scan(&originalURL)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return "", storage.ErrURLNotFound
	case err != nil:
		return "", fmt.Errorf("%s: %w", op, err)
	default:
		return originalURL, nil
	}
}
