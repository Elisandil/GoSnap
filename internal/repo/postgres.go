package repo

import (
	"context"
	"errors"
	"time"

	"github.com/Elisandil/GoSnap/internal/domain"
	"github.com/Elisandil/GoSnap/pkg/validator"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound         = errors.New("url not found")
	ErrAlreadyExists    = errors.New("short code for URL already exists")
	ErrInvalidShortCode = errors.New("invalid short code: must be between 1 and 10 characters")
)

// PostgresRepo is a repository that uses PostgreSQL as the backend.
type PostgresRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresRepo creates a new PostgresRepo with the given pgxpool.Pool.
func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{
		pool: pool,
	}
}

// Create inserts a new URL mapping into the database.
func (r *PostgresRepo) Create(ctx context.Context, shortCode, longURL string) (*domain.URL, error) {

	if !validator.IsValidShortCode(shortCode) {
		return nil, ErrInvalidShortCode
	}
	query := `INSERT INTO urls (short_code, long_url, created_at, clicks) 
				VALUES ($1, $2, $3, 0) 
				RETURNING id, short_code, long_url, created_at, clicks`

	var url domain.URL
	err := r.pool.QueryRow(ctx, query, shortCode, longURL, time.Now()).
		Scan(&url.ID, &url.ShortCode, &url.LongURL, &url.CreatedAt, &url.Clicks)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrAlreadyExists
		}
		return nil, err
	}

	return &url, nil
}

// GetByShortCode retrieves a URL mapping by its short code.
func (r *PostgresRepo) GetByShortCode(ctx context.Context, shortCode string) (*domain.URL, error) {

	if !validator.IsValidShortCode(shortCode) {
		return nil, ErrInvalidShortCode
	}
	query := `SELECT id, short_code, long_url, created_at, clicks
				FROM urls
				WHERE short_code = $1`

	var url domain.URL
	err := r.pool.QueryRow(ctx, query, shortCode).
		Scan(&url.ID, &url.ShortCode, &url.LongURL, &url.CreatedAt, &url.Clicks)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &url, nil
}

// IncrementClicksCounter increments the click counter for a given short code.
func (r *PostgresRepo) IncrementClicksCounter(ctx context.Context, shortCode string) error {

	if !validator.IsValidShortCode(shortCode) {
		return ErrInvalidShortCode
	}
	query := `UPDATE urls 
				SET clicks = clicks + 1 
				WHERE short_code = $1`

	result, err := r.pool.Exec(ctx, query, shortCode)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// GetNextID retrieves the next value from the URL ID sequence.
func (r *PostgresRepo) GetNextID(ctx context.Context) (int64, error) {
	query := `SELECT nextval('urls_id_seq')`

	var id int64
	err := r.pool.QueryRow(ctx, query).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}
