package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Elisandil/GoSnap/internal/domain"
	"github.com/Elisandil/GoSnap/internal/repo"
	"github.com/Elisandil/GoSnap/internal/shortid"
)

// ------------------------------------------------------------------------------------------
//                                        MOCKS
// ------------------------------------------------------------------------------------------

type mockPostgresRepo struct {
	nextID              int64
	createFunc          func(ctx context.Context, id int64, shortCode, longURL string) (*domain.URL, error)
	getByShortCodeFunc  func(ctx context.Context, shortCode string) (*domain.URL, error)
	incrementClicksFunc func(ctx context.Context, shortCode string) error
	getNextIDFunc       func(ctx context.Context) (int64, error)
}

func (m *mockPostgresRepo) Create(ctx context.Context, id int64, shortCode, longURL string) (*domain.URL, error) {

	if m.createFunc != nil {
		return m.createFunc(ctx, id, shortCode, longURL)
	}

	return &domain.URL{
		ID:        1,
		ShortCode: shortCode,
		LongURL:   longURL,
		CreatedAt: time.Now(),
		Clicks:    0,
	}, nil
}

func (m *mockPostgresRepo) GetByShortCode(ctx context.Context, shortCode string) (*domain.URL, error) {

	if m.getByShortCodeFunc != nil {
		return m.getByShortCodeFunc(ctx, shortCode)
	}

	return &domain.URL{
		ID:        1,
		ShortCode: shortCode,
		LongURL:   "https://example.com",
		CreatedAt: time.Now(),
		Clicks:    0,
	}, nil
}

func (m *mockPostgresRepo) IncrementClicksCounter(ctx context.Context, shortCode string) error {

	if m.incrementClicksFunc != nil {
		return m.incrementClicksFunc(ctx, shortCode)
	}

	return nil
}

func (m *mockPostgresRepo) GetNextID(ctx context.Context) (int64, error) {

	if m.getNextIDFunc != nil {
		return m.getNextIDFunc(ctx)
	}
	m.nextID++

	return m.nextID, nil
}

type mockRedisRepo struct {
	setFunc    func(ctx context.Context, shortCode string, url *domain.URL) error
	getFunc    func(ctx context.Context, shortCode string) (*domain.URL, error)
	deleteFunc func(ctx context.Context, shortCode string) error
	existsFunc func(ctx context.Context, shortCode string) (bool, error)
}

func (m *mockRedisRepo) Set(ctx context.Context, shortCode string, url *domain.URL) error {

	if m.setFunc != nil {
		return m.setFunc(ctx, shortCode, url)
	}

	return nil
}

func (m *mockRedisRepo) Get(ctx context.Context, shortCode string) (*domain.URL, error) {

	if m.getFunc != nil {
		return m.getFunc(ctx, shortCode)
	}

	return nil, repo.ErrNotFound
}

func (m *mockRedisRepo) Delete(ctx context.Context, shortCode string) error {

	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, shortCode)
	}

	return nil
}

func (m *mockRedisRepo) Exists(ctx context.Context, shortCode string) (bool, error) {

	if m.existsFunc != nil {
		return m.existsFunc(ctx, shortCode)
	}

	return false, nil
}

// ------------------------------------------------------------------------------------------
//                                  TABLE-DRIVEN TESTS
// ------------------------------------------------------------------------------------------

func TestShortenerService_CreateShortURL(t *testing.T) {
	tests := []struct {
		name          string
		longURL       string
		mockPg        *mockPostgresRepo
		mockRedis     *mockRedisRepo
		expectedError bool
	}{
		{
			name:    "successful creation",
			longURL: "https://www.google.com",
			mockPg: &mockPostgresRepo{
				nextID: 0,
			},
			mockRedis:     &mockRedisRepo{},
			expectedError: false,
		},
		{
			name:          "url not normalized",
			longURL:       "www.google.com",
			mockPg:        &mockPostgresRepo{},
			mockRedis:     &mockRedisRepo{},
			expectedError: false,
		},
		{
			name:          "invalid url",
			longURL:       "not a url",
			mockPg:        &mockPostgresRepo{},
			mockRedis:     &mockRedisRepo{},
			expectedError: true,
		},
		{
			name:    "database error",
			longURL: "https://example.com",
			mockPg: &mockPostgresRepo{
				getNextIDFunc: func(ctx context.Context) (int64, error) {
					return 0, errors.New("database error")
				},
			},
			mockRedis:     &mockRedisRepo{},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := shortid.NewGenerator()
			service := NewShortenerService(tt.mockPg, tt.mockRedis, generator, "http://localhost:8080")

			result, err := service.CreateShortURL(context.Background(), tt.longURL)

			if tt.expectedError {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Errorf("expected result but got nil")
				return
			}

			if result.ShortCode == "" {
				t.Errorf("expected short code but got empty string")
			}
			if result.ShortURL == "" {
				t.Errorf("expected short URL but got empty string")
			}
			if result.LongURL == "" {
				t.Errorf("expected long URL but got empty string")
			}
		})
	}
}

func TestShortenerService_CreateShortURL_Collisions(t *testing.T) {
	tests := []struct {
		name          string
		collisions    int
		expectedError bool
	}{
		{
			name:          "success after 1 collision",
			collisions:    1,
			expectedError: false,
		},
		{
			name:          "success after 2 collisions",
			collisions:    2,
			expectedError: false,
		},
		{
			name:          "fail after max retries",
			collisions:    3,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attempts := 0
			mockPg := &mockPostgresRepo{
				nextID: 0,
				createFunc: func(ctx context.Context, id int64, shortCode, longURL string) (*domain.URL, error) {
					attempts++
					if attempts <= tt.collisions {
						return nil, repo.ErrAlreadyExists
					}
					return &domain.URL{
						ID:        int64(attempts),
						ShortCode: shortCode,
						LongURL:   longURL,
						CreatedAt: time.Now(),
					}, nil
				},
			}
			mockRedis := &mockRedisRepo{}
			generator := shortid.NewGenerator()
			service := NewShortenerService(mockPg, mockRedis, generator, "http://localhost:8080")

			result, err := service.CreateShortURL(context.Background(), "https://example.com")

			if tt.expectedError {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Errorf("expected result but got nil")
			}
		})
	}
}

func TestShortenerService_GetLongURL(t *testing.T) {
	tests := []struct {
		name          string
		shortCode     string
		mockPg        *mockPostgresRepo
		mockRedis     *mockRedisRepo
		expectedURL   string
		expectedError string
	}{
		{
			name:      "cache hit",
			shortCode: "abc123",
			mockRedis: &mockRedisRepo{
				getFunc: func(ctx context.Context, shortCode string) (*domain.URL, error) {
					return &domain.URL{LongURL: "https://example.com"}, nil
				},
			},
			mockPg:      &mockPostgresRepo{},
			expectedURL: "https://example.com",
		},
		{
			name:      "cache miss - db hit",
			shortCode: "xyz789",
			mockRedis: &mockRedisRepo{
				getFunc: func(ctx context.Context, shortCode string) (*domain.URL, error) {
					return nil, repo.ErrNotFound
				},
			},
			mockPg: &mockPostgresRepo{
				getByShortCodeFunc: func(ctx context.Context, shortCode string) (*domain.URL, error) {
					return &domain.URL{LongURL: "https://example.com"}, nil
				},
			},
			expectedURL: "https://example.com",
		},
		{
			name:          "invalid short code format",
			shortCode:     "invalid@code!",
			mockRedis:     &mockRedisRepo{},
			mockPg:        &mockPostgresRepo{},
			expectedError: "invalid short code format",
		},
		{
			name:      "short code not found in db",
			shortCode: "notfound",
			mockRedis: &mockRedisRepo{
				getFunc: func(ctx context.Context, shortCode string) (*domain.URL, error) {
					return nil, repo.ErrNotFound
				},
			},
			mockPg: &mockPostgresRepo{
				getByShortCodeFunc: func(ctx context.Context, shortCode string) (*domain.URL, error) {
					return nil, repo.ErrNotFound
				},
			},
			expectedError: "short URL not found",
		},
		{
			name:      "database error",
			shortCode: "abc123",
			mockRedis: &mockRedisRepo{
				getFunc: func(ctx context.Context, shortCode string) (*domain.URL, error) {
					return nil, repo.ErrNotFound
				},
			},
			mockPg: &mockPostgresRepo{
				getByShortCodeFunc: func(ctx context.Context, shortCode string) (*domain.URL, error) {
					return nil, errors.New("db connection error")
				},
			},
			expectedError: "error retrieving long URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := shortid.NewGenerator()
			service := NewShortenerService(tt.mockPg, tt.mockRedis, generator, "http://localhost:8080")

			longURL, err := service.GetLongURL(context.Background(), tt.shortCode)

			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("expected error containing '%s', got nil", tt.expectedError)
					return
				}
				if !contains(err.Error(), tt.expectedError) {
					t.Errorf("expected error '%s', got '%s'", tt.expectedError, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if longURL != tt.expectedURL {
				t.Errorf("expected '%s', got '%s'", tt.expectedURL, longURL)
			}
		})
	}
}

func TestNewShortenerService_GetLongURL_CacheMiss(t *testing.T) {
	mockPg := &mockPostgresRepo{
		getByShortCodeFunc: func(ctx context.Context, shortCode string) (*domain.URL, error) {
			return &domain.URL{
				ID:        1,
				ShortCode: shortCode,
				LongURL:   "https://example.com",
				CreatedAt: time.Now(),
				Clicks:    5,
			}, nil
		},
	}
	mockRedis := &mockRedisRepo{
		getFunc: func(ctx context.Context, shortCode string) (*domain.URL, error) {
			return nil, repo.ErrNotFound
		},
	}
	generator := shortid.NewGenerator()
	service := NewShortenerService(mockPg, mockRedis, generator, "http://localhost:8080")

	longURL, err := service.GetLongURL(context.Background(), "xyz789")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if longURL != "https://example.com" {
		t.Errorf("expected long URL 'https://example.com', got '%s'", longURL)
	}
}

func TestShortenerService_GetURLStats(t *testing.T) {
	tests := []struct {
		name          string
		shortCode     string
		mockPg        *mockPostgresRepo
		expectedStats *domain.StatsResponse
		expectedError string
	}{
		{
			name:      "success",
			shortCode: "abc123",
			mockPg: &mockPostgresRepo{
				getByShortCodeFunc: func(ctx context.Context, shortCode string) (*domain.URL, error) {
					return &domain.URL{
						ShortCode: shortCode,
						LongURL:   "https://example.com",
						Clicks:    42,
						CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					}, nil
				},
			},
			expectedStats: &domain.StatsResponse{
				ShortCode: "abc123",
				LongURL:   "https://example.com",
				Clicks:    42,
				CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name:          "invalid short code",
			shortCode:     "invalid@",
			mockPg:        &mockPostgresRepo{},
			expectedError: "invalid short code format",
		},
		{
			name:      "not found",
			shortCode: "notfound",
			mockPg: &mockPostgresRepo{
				getByShortCodeFunc: func(ctx context.Context, shortCode string) (*domain.URL, error) {
					return nil, repo.ErrNotFound
				},
			},
			expectedError: "short URL not found",
		},
		{
			name:      "database error",
			shortCode: "abc123",
			mockPg: &mockPostgresRepo{
				getByShortCodeFunc: func(ctx context.Context, shortCode string) (*domain.URL, error) {
					return nil, errors.New("db error")
				},
			},
			expectedError: "error retrieving URL stats",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := shortid.NewGenerator()
			service := NewShortenerService(tt.mockPg, &mockRedisRepo{}, generator, "http://localhost:8080")

			stats, err := service.GetURLStats(context.Background(), tt.shortCode)

			if tt.expectedError != "" {
				if err == nil || !contains(err.Error(), tt.expectedError) {
					t.Errorf("expected error '%s', got %v", tt.expectedError, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if stats.Clicks != tt.expectedStats.Clicks {
				t.Errorf("expected %d clicks, got %d", tt.expectedStats.Clicks, stats.Clicks)
			}
		})
	}
}

// ------------------------------------------------------------------------------------------
//                                        HELPERS
// ------------------------------------------------------------------------------------------

func contains(s, substr string) bool {

	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
