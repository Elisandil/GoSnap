package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Elisandil/GoSnap/internal/domain"
	"github.com/Elisandil/GoSnap/internal/repo"
	"github.com/Elisandil/GoSnap/internal/shortid"
	"github.com/Elisandil/GoSnap/pkg/validator"
	"github.com/rs/zerolog/log"
)

// ----------------------------------------------------------------------------------------
//                                    INTERFACES
// ----------------------------------------------------------------------------------------

type PostgresRepository interface {
	Create(ctx context.Context, id int64, shortCode, longURL string) (*domain.URL, error)
	GetByShortCode(ctx context.Context, shortCode string) (*domain.URL, error)
	IncrementClicksCounter(ctx context.Context, shortCode string) error
	GetNextID(ctx context.Context) (int64, error)
}

type RedisRepository interface {
	Get(ctx context.Context, shortCode string) (*domain.URL, error)
	Set(ctx context.Context, shortCode string, url *domain.URL) error
	Delete(ctx context.Context, shortCode string) error
	Exists(ctx context.Context, shortCode string) (bool, error)
}

// ----------------------------------------------------------------------------------------
//                                    SERVICE
// ----------------------------------------------------------------------------------------

type ShortenerService struct {
	pgRepo       PostgresRepository
	redisRepo    RedisRepository
	generator    *shortid.Generator
	baseURL      string
	maxRetries   int
	clickWorkers chan struct{}
}

func NewShortenerService(pgRepo PostgresRepository,
	redisRepo RedisRepository,
	generator *shortid.Generator,
	baseURL string) *ShortenerService {

	return &ShortenerService{
		pgRepo:       pgRepo,
		redisRepo:    redisRepo,
		generator:    generator,
		baseURL:      baseURL,
		maxRetries:   3,
		clickWorkers: make(chan struct{}, 100),
	}
}

// CreateShortURL creates a short URL for the given long URL.
func (s *ShortenerService) CreateShortURL(ctx context.Context, longURL string) (*domain.CreateURLResponse, error) {
	return s.createShortURLWithRetries(ctx, longURL, 0, s.maxRetries)
}

// GetLongURL retrieves the long URL associated with the given short code.
// It first checks the Redis cache for the short code.
// If the short code is found in the cache, it returns the long URL and increments the click counter asynchronously.
// If the short code is not found in the cache, it queries the Postgres database.
// If the short code is not found in the database, it returns an error.
// If there is an error retrieving the URL from the database, it returns an error.
// On success, it returns the long URL.
func (s *ShortenerService) GetLongURL(ctx context.Context, shortCode string) (string, error) {

	if !validator.IsValidShortCode(shortCode) {
		return "", fmt.Errorf("invalid short code format")
	}

	url, err := s.redisRepo.Get(ctx, shortCode)
	if err == nil {
		log.Debug().Str("short_code", shortCode).Msg("cache hit")
		s.incrementClicksAsync(shortCode)

		return url.LongURL, nil
	}

	log.Debug().Str("short_code", shortCode).Msg("cache miss, querying from Postgres")
	url, err = s.pgRepo.GetByShortCode(ctx, shortCode)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return "", fmt.Errorf("short URL not found")
		}
		log.Error().Err(err).Str("short_code", shortCode).Msg("error retrieving URL from the database")

		return "", fmt.Errorf("error retrieving long URL")
	}

	if err := s.redisRepo.Set(ctx, shortCode, url); err != nil {
		log.Warn().Err(err).Str("short_code", shortCode).Msg("error caching URL with Redis")
	}
	s.incrementClicksAsync(shortCode)

	return url.LongURL, nil
}

// GetURLStats retrieves statistics for the given short code.
// It queries the Postgres database for the URL associated with the short code.
// If the short code is not found, it returns an error.
// If there is an error retrieving the URL from the database, it returns an error.
// On success, it returns a StatsResponse containing the short code, long URL, click count, and creation date.
func (s *ShortenerService) GetURLStats(ctx context.Context, shortCode string) (*domain.StatsResponse, error) {

	if !validator.IsValidShortCode(shortCode) {
		return nil, fmt.Errorf("invalid short code format")
	}

	url, err := s.pgRepo.GetByShortCode(ctx, shortCode)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, fmt.Errorf("short URL not found")
		}
		return nil, fmt.Errorf("error retrieving URL stats")
	}

	return &domain.StatsResponse{
		ShortCode: url.ShortCode,
		LongURL:   url.LongURL,
		Clicks:    url.Clicks,
		CreatedAt: url.CreatedAt,
	}, nil
}

// ----------------------------------------------------------------------------------------
//                                    PRIVATE METHODS
// ----------------------------------------------------------------------------------------

// createShortURLWithRetries attempts to create a short URL for the given long URL.
// It retries the operation up to maxRetries times in case of collisions.
// If the long URL is invalid, it returns an error.
// If a collision occurs, it logs a warning and retries.
// If the maximum number of retries is reached, it returns an error.
// On success, it returns a CreateURLResponse containing the short code, short URL, and long URL.
func (s *ShortenerService) createShortURLWithRetries(ctx context.Context,
	longURL string,
	attempt, maxRetries int) (*domain.CreateURLResponse, error) {

	if attempt >= maxRetries {
		return nil, fmt.Errorf("max retries reached for creating short URL")
	}

	longURL = validator.NormalizeURL(longURL)
	if !validator.IsValidURL(longURL) {
		return nil, fmt.Errorf("invalid URL: %s", longURL)
	}

	id, err := s.pgRepo.GetNextID(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error getting next ID")
		return nil, fmt.Errorf("error generating short code")
	}

	shortCode := s.generator.Encode(id)

	url, err := s.pgRepo.Create(ctx, id, shortCode, longURL)
	if err != nil {
		if errors.Is(err, repo.ErrAlreadyExists) {
			log.Warn().Str("short_code", shortCode).Msg("collision detected, retrying")

			return s.createShortURLWithRetries(ctx, longURL, attempt+1, maxRetries)
		}
		log.Error().Err(err).Msg("error inserting URL into the database")

		return nil, fmt.Errorf("error creating short URL")
	}

	if err := s.redisRepo.Set(ctx, shortCode, url); err != nil {
		log.Warn().Err(err).Str("short_code", shortCode).Msg("error caching URL with Redis")
	}

	return &domain.CreateURLResponse{
		ShortCode: shortCode,
		ShortURL:  fmt.Sprintf("%s/%s", s.baseURL, shortCode),
		LongURL:   longURL,
	}, nil
}

// incrementClicksAsync increments the click counter for the given short code asynchronously.
// It runs the increment operation in a separate goroutine with a timeout context.
// If there is an error incrementing the counter, it logs a warning.
func (s *ShortenerService) incrementClicksAsync(shortCode string) {

	select {
	case s.clickWorkers <- struct{}{}:
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := s.pgRepo.IncrementClicksCounter(ctx, shortCode); err != nil {
				log.Warn().Err(err).Str("short_code", shortCode).Msg("error incrementing clicks counter in " +
					"background")
			}
		}()
	default:
		log.Warn().Msg("click workers limit reached, skipping increment")
	}
}
