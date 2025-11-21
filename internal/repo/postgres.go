package repo

import (
	"context"
	"errors"
	"time"

	"github.com/Elisandil/GoSnap/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound      = errors.New("url not found")
	ErrAlreadyExists = errors.New("short code for URL already exists")
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

// Insert inserts a new URL mapping into the database.
func (r *PostgresRepo) Insert(ctx context.Context, shortCode, longURL string) (*domain.URL, error) {
	query := `INSERT INTO urls (short_code, long_url, created_at, clicks) 
				VALUES ($1, $2, $3, 0) 
				RETURNING id, short_code, long_url, created_at, clicks`

	var url domain.URL
	err := r.pool.QueryRow(ctx, query, shortCode, longURL, time.Now()).
		Scan(&url.ID, &url.ShortCode, &url.LongURL, &url.CreatedAt, &url.Clicks)

	if err != nil {
		// Check for unique constraint violation
		if err.Error() == "ERROR: duplicate key value violates unique constraint "+
			"\"urls_short_code_key\" (SQLSTATE 23505)" {

			return nil, ErrAlreadyExists
		}
		return nil, err
	}

	return &url, nil
}

// GetByShortCode retrieves a URL mapping by its short code.
func (r *PostgresRepo) GetByShortCode(ctx context.Context, shortCode string) (*domain.URL, error) {
	query := `SELECT id, short_code, long_url, created_At, clicks
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
	query := `UPDATE urls 
				SET clicks = clicks + 1 
				WHERE short_code = $1`

	_, err := r.pool.Exec(ctx, query, shortCode)

	return err
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
