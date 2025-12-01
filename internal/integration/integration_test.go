package integration

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/Elisandil/GoSnap/internal/repo"
	"github.com/Elisandil/GoSnap/internal/service"
	"github.com/Elisandil/GoSnap/internal/shortid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// ------------------------------------------------------------------------------------------
//                                    TEST SETUP
// ------------------------------------------------------------------------------------------

var (
	testPgPool      *pgxpool.Pool
	testRedisClient *redis.Client
	testService     *service.ShortenerService
)

// setupTestEnvironment initializes the test database and Redis connections
func setupTestEnvironment(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	pgConnStr := os.Getenv("TEST_POSTGRES_URL")
	if pgConnStr == "" {
		pgConnStr = "postgres://postgres:postgres@localhost:5433/urlshortener_test?sslmode=disable"
	}

	var err error
	testPgPool, err = pgxpool.New(context.Background(), pgConnStr)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	if err := testPgPool.Ping(context.Background()); err != nil {
		t.Fatalf("Failed to ping test database: %v", err)
	}

	redisAddr := os.Getenv("TEST_REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6380"
	}

	testRedisClient = redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: os.Getenv("TEST_REDIS_PASSWORD"),
		DB:       1, // Use DB 1 for tests
	})

	if err := testRedisClient.Ping(context.Background()).Err(); err != nil {
		t.Fatalf("Failed to connect to test Redis: %v", err)
	}

	pgRepo := repo.NewPostgresRepo(testPgPool)
	redisRepo := repo.NewRedisRepo(testRedisClient, 1*time.Hour)
	generator := shortid.NewGenerator()
	testService = service.NewShortenerService(pgRepo, redisRepo, generator, "http://localhost:8080")
}

// teardownTestEnvironment cleans up test resources
func teardownTestEnvironment(t *testing.T) {
	t.Helper()

	if testPgPool != nil {
		testPgPool.Close()
	}

	if testRedisClient != nil {
		err := testRedisClient.Close()
		if err != nil {
			t.Logf("Warning: Failed to close test Redis client: %v", err)
		}
	}
}

// cleanupTestData removes all test data from database and Redis
func cleanupTestData(t *testing.T) {
	t.Helper()

	ctx := context.Background()

	_, err := testPgPool.Exec(ctx, "TRUNCATE TABLE urls RESTART IDENTITY CASCADE")
	if err != nil {
		t.Logf("Warning: Failed to clean test database: %v", err)
	}

	err = testRedisClient.FlushDB(ctx).Err()
	if err != nil {
		t.Logf("Warning: Failed to clean test Redis: %v", err)
	}
}

// ------------------------------------------------------------------------------------------
//                              INTEGRATION TESTS
// ------------------------------------------------------------------------------------------

func TestIntegration_CreateAndRetrieveURL(t *testing.T) {
	setupTestEnvironment(t)
	defer teardownTestEnvironment(t)
	cleanupTestData(t)

	ctx := context.Background()
	longURL := "https://example.com/test/integration"

	result, err := testService.CreateShortURL(ctx, longURL)
	if err != nil {
		t.Fatalf("Failed to create short URL: %v", err)
	}
	if result.ShortCode == "" {
		t.Error("Expected non-empty short code")
	}
	if result.ShortURL == "" {
		t.Error("Expected non-empty short URL")
	}
	if result.LongURL != longURL {
		t.Errorf("Expected long URL '%s', got '%s'", longURL, result.LongURL)
	}

	retrievedURL, err := testService.GetLongURL(ctx, result.ShortCode)
	if err != nil {
		t.Fatalf("Failed to retrieve long URL: %v", err)
	}
	if retrievedURL != longURL {
		t.Errorf("Expected retrieved URL '%s', got '%s'", longURL, retrievedURL)
	}

	stats, err := testService.GetURLStats(ctx, result.ShortCode)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}
	if stats.ShortCode != result.ShortCode {
		t.Errorf("Expected short code '%s', got '%s'", result.ShortCode, stats.ShortCode)
	}
	if stats.LongURL != longURL {
		t.Errorf("Expected long URL '%s', got '%s'", longURL, stats.LongURL)
	}
}

func TestIntegration_CreateDuplicateShortCode(t *testing.T) {
	setupTestEnvironment(t)
	defer teardownTestEnvironment(t)
	cleanupTestData(t)

	ctx := context.Background()

	result1, err := testService.CreateShortURL(ctx, "https://example.com/first")
	if err != nil {
		t.Fatalf("Failed to create first URL: %v", err)
	}

	result2, err := testService.CreateShortURL(ctx, "https://example.com/second")
	if err != nil {
		t.Fatalf("Failed to create second URL: %v", err)
	}
	if result1.ShortCode == result2.ShortCode {
		t.Error("Expected different short codes for different URLs")
	}

	url1, err := testService.GetLongURL(ctx, result1.ShortCode)
	if err != nil || url1 != "https://example.com/first" {
		t.Errorf("Failed to retrieve first URL: %v", err)
	}

	url2, err := testService.GetLongURL(ctx, result2.ShortCode)
	if err != nil || url2 != "https://example.com/second" {
		t.Errorf("Failed to retrieve second URL: %v", err)
	}
}

func TestIntegration_RedisCache_HitAndMiss(t *testing.T) {
	setupTestEnvironment(t)
	defer teardownTestEnvironment(t)
	cleanupTestData(t)

	ctx := context.Background()
	longURL := "https://example.com/cache-test"

	result, err := testService.CreateShortURL(ctx, longURL)
	if err != nil {
		t.Fatalf("Failed to create URL: %v", err)
	}

	start1 := time.Now()
	url1, err := testService.GetLongURL(ctx, result.ShortCode)
	duration1 := time.Since(start1)
	if err != nil {
		t.Fatalf("Failed to retrieve URL (cache hit): %v", err)
	}
	if url1 != longURL {
		t.Errorf("Expected URL '%s', got '%s'", longURL, url1)
	}

	err = testRedisClient.Del(ctx, result.ShortCode).Err()
	if err != nil {
		t.Fatalf("Failed to clear cache: %v", err)
	}

	start2 := time.Now()
	url2, err := testService.GetLongURL(ctx, result.ShortCode)
	duration2 := time.Since(start2)
	if err != nil {
		t.Fatalf("Failed to retrieve URL (cache miss): %v", err)
	}
	if url2 != longURL {
		t.Errorf("Expected URL '%s', got '%s'", longURL, url2)
	}

	t.Logf("Cache hit duration: %v, Cache miss duration: %v", duration1, duration2)
}

func TestIntegration_ClickCounterIncrement(t *testing.T) {
	setupTestEnvironment(t)
	defer teardownTestEnvironment(t)
	cleanupTestData(t)

	ctx := context.Background()
	longURL := "https://example.com/click-test"

	result, err := testService.CreateShortURL(ctx, longURL)
	if err != nil {
		t.Fatalf("Failed to create URL: %v", err)
	}

	stats, err := testService.GetURLStats(ctx, result.ShortCode)
	if err != nil {
		t.Fatalf("Failed to get initial stats: %v", err)
	}
	if stats.Clicks != 0 {
		t.Errorf("Expected 0 initial clicks, got %d", stats.Clicks)
	}

	numAccesses := 5
	for i := 0; i < numAccesses; i++ {
		_, err := testService.GetLongURL(ctx, result.ShortCode)
		if err != nil {
			t.Fatalf("Failed to access URL (iteration %d): %v", i, err)
		}
	}
	time.Sleep(2 * time.Second)

	stats, err = testService.GetURLStats(ctx, result.ShortCode)
	if err != nil {
		t.Fatalf("Failed to get final stats: %v", err)
	}
	if stats.Clicks != int64(numAccesses) {
		t.Errorf("Expected %d clicks, got %d", numAccesses, stats.Clicks)
	}
}

func TestIntegration_ConcurrentRequests(t *testing.T) {
	setupTestEnvironment(t)
	defer teardownTestEnvironment(t)
	cleanupTestData(t)

	ctx := context.Background()
	numGoroutines := 50
	numRequestsPerGoroutine := 10

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*numRequestsPerGoroutine)

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numRequestsPerGoroutine; j++ {
				longURL := fmt.Sprintf("https://example.com/concurrent/%d/%d", id, j)
				_, err := testService.CreateShortURL(ctx, longURL)
				if err != nil {
					errors <- fmt.Errorf("goroutine %d, request %d: %w", id, j, err)
				}
			}
		}(i)
	}
	wg.Wait()
	close(errors)

	errorCount := 0
	for err := range errors {
		t.Errorf("Concurrent request error: %v", err)
		errorCount++
	}
	if errorCount > 0 {
		t.Fatalf("Failed with %d errors out of %d total requests", errorCount, numGoroutines*numRequestsPerGoroutine)
	}

	expectedCount := numGoroutines * numRequestsPerGoroutine
	var actualCount int64
	err := testPgPool.QueryRow(ctx, "SELECT COUNT(*) FROM urls").Scan(&actualCount)
	if err != nil {
		t.Fatalf("Failed to count URLs: %v", err)
	}
	if int(actualCount) != expectedCount {
		t.Errorf("Expected %d URLs, got %d", expectedCount, actualCount)
	}
}

func TestIntegration_MultipleURLs(t *testing.T) {
	setupTestEnvironment(t)
	defer teardownTestEnvironment(t)
	cleanupTestData(t)

	ctx := context.Background()
	numURLs := 100
	urls := make(map[string]string) // shortCode -> longURL

	for i := 0; i < numURLs; i++ {
		longURL := fmt.Sprintf("https://example.com/test/%d", i)
		result, err := testService.CreateShortURL(ctx, longURL)
		if err != nil {
			t.Fatalf("Failed to create URL %d: %v", i, err)
		}
		urls[result.ShortCode] = longURL
	}

	for shortCode, expectedLongURL := range urls {
		actualLongURL, err := testService.GetLongURL(ctx, shortCode)
		if err != nil {
			t.Errorf("Failed to retrieve URL for short code '%s': %v", shortCode, err)
			continue
		}
		if actualLongURL != expectedLongURL {
			t.Errorf("Short code '%s': expected URL '%s', got '%s'", shortCode, expectedLongURL, actualLongURL)
		}
	}

	for shortCode, expectedLongURL := range urls {
		stats, err := testService.GetURLStats(ctx, shortCode)
		if err != nil {
			t.Errorf("Failed to get stats for short code '%s': %v", shortCode, err)
			continue
		}
		if stats.ShortCode != shortCode {
			t.Errorf("Expected short code '%s', got '%s'", shortCode, stats.ShortCode)
		}
		if stats.LongURL != expectedLongURL {
			t.Errorf("Expected long URL '%s', got '%s'", expectedLongURL, stats.LongURL)
		}
	}
}

func TestIntegration_URLNormalization(t *testing.T) {
	setupTestEnvironment(t)
	defer teardownTestEnvironment(t)
	cleanupTestData(t)

	ctx := context.Background()

	tests := []struct {
		name        string
		input       string
		expectedURL string
		shouldFail  bool
	}{
		{
			name:        "url with https",
			input:       "https://example.com",
			expectedURL: "https://example.com",
			shouldFail:  false,
		},
		{
			name:        "url without scheme",
			input:       "example.com",
			expectedURL: "https://example.com",
			shouldFail:  false,
		},
		{
			name:        "url with spaces",
			input:       "  https://example.com  ",
			expectedURL: "https://example.com",
			shouldFail:  false,
		},
		{
			name:        "invalid url",
			input:       "not a url",
			expectedURL: "",
			shouldFail:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := testService.CreateShortURL(ctx, tt.input)

			if tt.shouldFail {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if result.LongURL != tt.expectedURL {
				t.Errorf("Expected normalized URL '%s', got '%s'", tt.expectedURL, result.LongURL)
			}

			retrievedURL, err := testService.GetLongURL(ctx, result.ShortCode)
			if err != nil {
				t.Fatalf("Failed to retrieve URL: %v", err)
			}
			if retrievedURL != tt.expectedURL {
				t.Errorf("Expected retrieved URL '%s', got '%s'", tt.expectedURL, retrievedURL)
			}
		})
	}
}

func TestIntegration_InvalidShortCode(t *testing.T) {
	setupTestEnvironment(t)
	defer teardownTestEnvironment(t)
	cleanupTestData(t)

	ctx := context.Background()

	invalidCodes := []string{
		"invalid@code",
		"code with spaces",
		"",
		"toolongcode123456789",
		"código-español",
		"emoji😊code",
	}

	for _, code := range invalidCodes {
		t.Run(fmt.Sprintf("invalid_code_%s", code), func(t *testing.T) {
			_, err := testService.GetLongURL(ctx, code)
			if err == nil {
				t.Errorf("Expected error for invalid short code '%s', got none", code)
			}

			_, err = testService.GetURLStats(ctx, code)
			if err == nil {
				t.Errorf("Expected error for invalid short code '%s' in stats, got none", code)
			}
		})
	}
}

func TestIntegration_NonExistentShortCode(t *testing.T) {
	setupTestEnvironment(t)
	defer teardownTestEnvironment(t)
	cleanupTestData(t)

	ctx := context.Background()

	_, err := testService.GetLongURL(ctx, "notfound")
	if err == nil {
		t.Error("Expected error for non-existent short code, got none")
	}

	_, err = testService.GetURLStats(ctx, "notfound")
	if err == nil {
		t.Error("Expected error for non-existent short code in stats, got none")
	}
}

// ------------------------------------------------------------------------------------------
//                                    BENCHMARK TESTS
// ------------------------------------------------------------------------------------------

func BenchmarkIntegration_CreateShortURL(b *testing.B) {
	setupTestEnvironment(&testing.T{})
	defer teardownTestEnvironment(&testing.T{})
	cleanupTestData(&testing.T{})

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		longURL := fmt.Sprintf("https://example.com/benchmark/%d", i)
		_, err := testService.CreateShortURL(ctx, longURL)
		if err != nil {
			b.Fatalf("Failed to create URL: %v", err)
		}
	}
}

func BenchmarkIntegration_GetLongURL_CacheHit(b *testing.B) {
	setupTestEnvironment(&testing.T{})
	defer teardownTestEnvironment(&testing.T{})
	cleanupTestData(&testing.T{})

	ctx := context.Background()

	result, err := testService.CreateShortURL(ctx, "https://example.com/benchmark")
	if err != nil {
		b.Fatalf("Failed to create URL: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := testService.GetLongURL(ctx, result.ShortCode)
		if err != nil {
			b.Fatalf("Failed to get URL: %v", err)
		}
	}
}

func BenchmarkIntegration_GetLongURL_CacheMiss(b *testing.B) {
	setupTestEnvironment(&testing.T{})
	defer teardownTestEnvironment(&testing.T{})
	cleanupTestData(&testing.T{})

	ctx := context.Background()

	result, err := testService.CreateShortURL(ctx, "https://example.com/benchmark")
	if err != nil {
		b.Fatalf("Failed to create URL: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		testRedisClient.Del(ctx, result.ShortCode)

		_, err := testService.GetLongURL(ctx, result.ShortCode)
		if err != nil {
			b.Fatalf("Failed to get URL: %v", err)
		}
	}
}
