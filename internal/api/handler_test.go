package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Elisandil/go-snap/internal/domain"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

// ------------------------------------------------------------------------------------------
//                                        MOCK SERVICE
// ------------------------------------------------------------------------------------------

type mockShortenerService struct {
	createFunc   func(ctx context.Context, longURL string) (*domain.CreateURLResponse, error)
	getLongFunc  func(ctx context.Context, shortCode string) (string, error)
	getStatsFunc func(ctx context.Context, shortCode string) (*domain.StatsResponse, error)
}

func (m *mockShortenerService) CreateShortURL(ctx context.Context, longURL string) (*domain.CreateURLResponse, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, longURL)
	}
	return &domain.CreateURLResponse{
		ShortCode: "abc123",
		ShortURL:  "http://localhost:8080/abc123",
		LongURL:   longURL,
	}, nil
}

func (m *mockShortenerService) GetLongURL(ctx context.Context, shortCode string) (string, error) {
	if m.getLongFunc != nil {
		return m.getLongFunc(ctx, shortCode)
	}
	return "https://example.com", nil
}

func (m *mockShortenerService) GetURLStats(ctx context.Context, shortCode string) (*domain.StatsResponse, error) {
	if m.getStatsFunc != nil {
		return m.getStatsFunc(ctx, shortCode)
	}
	return &domain.StatsResponse{
		ShortCode: shortCode,
		LongURL:   "https://example.com",
		Clicks:    42,
		CreatedAt: time.Now(),
	}, nil
}

// ------------------------------------------------------------------------------------------
//                              TESTS: CreateShortURL
// ------------------------------------------------------------------------------------------

func TestHandler_CreateShortURL_Success(t *testing.T) {
	mockService := &mockShortenerService{
		createFunc: func(ctx context.Context, longURL string) (*domain.CreateURLResponse, error) {
			return &domain.CreateURLResponse{
				ShortCode: "abc123",
				ShortURL:  "http://localhost:8080/abc123",
				LongURL:   longURL,
			}, nil
		},
	}

	handler := NewHandler(mockService)
	e := setupEcho()

	reqBody := `{"long_url": "https://example.com"}`
	rec, c := testRequest(t, e, http.MethodPost, "/api/shorten", reqBody)

	handleRequest(t, handler.CreateShortURL, c)
	assertStatusCode(t, rec, http.StatusCreated)

	var response domain.CreateURLResponse
	assertJSONResponse(t, rec, &response)

	if response.ShortCode != "abc123" {
		t.Errorf("expected short_code 'abc123', got '%s'", response.ShortCode)
	}
	if response.ShortURL != "http://localhost:8080/abc123" {
		t.Errorf("expected short_url 'http://localhost:8080/abc123', got '%s'", response.ShortURL)
	}
	if response.LongURL != "https://example.com" {
		t.Errorf("expected long_url 'https://example.com', got '%s'", response.LongURL)
	}
}

func TestHandler_CreateShortURL_InvalidJSON(t *testing.T) {
	mockService := &mockShortenerService{}
	handler := NewHandler(mockService)
	e := setupEcho()

	reqBody := `{"long_url": invalid json}`
	rec, c := testRequest(t, e, http.MethodPost, "/api/shorten", reqBody)

	handleRequest(t, handler.CreateShortURL, c)
	assertStatusCode(t, rec, http.StatusBadRequest)
	assertErrorResponse(t, rec, "")
}

func TestHandler_CreateShortURL_ValidationError_MissingURL(t *testing.T) {
	mockService := &mockShortenerService{}
	handler := NewHandler(mockService)
	e := setupEcho()

	reqBody := `{"long_url": ""}`
	rec, c := testRequest(t, e, http.MethodPost, "/api/shorten", reqBody)

	handleRequest(t, handler.CreateShortURL, c)
	assertStatusCode(t, rec, http.StatusBadRequest)
	assertErrorResponse(t, rec, "Validation failed")
}

func TestHandler_CreateShortURL_ValidationError_InvalidURL(t *testing.T) {
	mockService := &mockShortenerService{}
	handler := NewHandler(mockService)
	e := setupEcho()

	reqBody := `{"long_url": "not-a-valid-url"}`
	rec, c := testRequest(t, e, http.MethodPost, "/api/shorten", reqBody)

	handleRequest(t, handler.CreateShortURL, c)
	assertStatusCode(t, rec, http.StatusBadRequest)
}

func TestHandler_CreateShortURL_ServiceError(t *testing.T) {
	mockService := &mockShortenerService{
		createFunc: func(ctx context.Context, longURL string) (*domain.CreateURLResponse, error) {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, "database connection failed")
		},
	}

	handler := NewHandler(mockService)
	e := setupEcho()

	reqBody := `{"long_url": "https://example.com"}`
	rec, c := testRequest(t, e, http.MethodPost, "/api/shorten", reqBody)

	handleRequest(t, handler.CreateShortURL, c)
	assertStatusCode(t, rec, http.StatusInternalServerError)
	assertErrorResponse(t, rec, "Failed to create short URL")
}

// ------------------------------------------------------------------------------------------
//                                    TESTS: Redirect
// ------------------------------------------------------------------------------------------

func TestHandler_Redirect_Success(t *testing.T) {
	mockService := &mockShortenerService{
		getLongFunc: func(ctx context.Context, shortCode string) (string, error) {
			return "https://example.com", nil
		},
	}

	handler := NewHandler(mockService)
	e := setupEcho()

	rec, c := testRequestWithParam(t, e, http.MethodGet, "/abc123", "shortCode", "abc123")

	handleRequest(t, handler.Redirect, c)
	assertStatusCode(t, rec, http.StatusFound)
	assertHeader(t, rec, "Location", "https://example.com")
}

func TestHandler_Redirect_NotFound(t *testing.T) {
	mockService := &mockShortenerService{
		getLongFunc: func(ctx context.Context, shortCode string) (string, error) {
			return "", echo.NewHTTPError(http.StatusNotFound, "short URL not found")
		},
	}

	handler := NewHandler(mockService)
	e := setupEcho()

	rec, c := testRequestWithParam(t, e, http.MethodGet, "/notfound", "shortCode", "notfound")

	handleRequest(t, handler.Redirect, c)
	assertStatusCode(t, rec, http.StatusNotFound)
	assertErrorResponse(t, rec, "Short URL not found")
}

func TestHandler_Redirect_InvalidShortCode(t *testing.T) {
	mockService := &mockShortenerService{
		getLongFunc: func(ctx context.Context, shortCode string) (string, error) {
			return "", echo.NewHTTPError(http.StatusBadRequest, "invalid short code format")
		},
	}

	handler := NewHandler(mockService)
	e := setupEcho()

	rec, c := testRequestWithParam(t, e, http.MethodGet, "/invalid@code!", "shortCode", "invalid@code!")

	handleRequest(t, handler.Redirect, c)
	assertStatusCode(t, rec, http.StatusNotFound)
}

func TestHandler_Redirect_EmptyShortCode(t *testing.T) {
	mockService := &mockShortenerService{
		getLongFunc: func(ctx context.Context, shortCode string) (string, error) {
			return "", echo.NewHTTPError(http.StatusBadRequest, "invalid short code format")
		},
	}

	handler := NewHandler(mockService)
	e := setupEcho()

	rec, c := testRequestWithParam(t, e, http.MethodGet, "/", "shortCode", "")

	handleRequest(t, handler.Redirect, c)
	assertStatusCode(t, rec, http.StatusNotFound)
}

// ------------------------------------------------------------------------------------------
//                                    TESTS: GetStats
// ------------------------------------------------------------------------------------------

func TestHandler_GetStats_Success(t *testing.T) {
	expectedTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	mockService := &mockShortenerService{
		getStatsFunc: func(ctx context.Context, shortCode string) (*domain.StatsResponse, error) {
			return &domain.StatsResponse{
				ShortCode: shortCode,
				LongURL:   "https://example.com",
				Clicks:    42,
				CreatedAt: expectedTime,
			}, nil
		},
	}

	handler := NewHandler(mockService)
	e := setupEcho()

	rec, c := testRequestWithParam(t, e, http.MethodGet, "/api/stats/abc123", "shortCode", "abc123")
	c.SetPath("/api/stats/:shortCode")

	handleRequest(t, handler.GetStats, c)
	assertStatusCode(t, rec, http.StatusOK)

	var response domain.StatsResponse
	assertJSONResponse(t, rec, &response)

	if response.ShortCode != "abc123" {
		t.Errorf("expected short_code 'abc123', got '%s'", response.ShortCode)
	}
	if response.LongURL != "https://example.com" {
		t.Errorf("expected long_url 'https://example.com', got '%s'", response.LongURL)
	}
	if response.Clicks != 42 {
		t.Errorf("expected clicks 42, got %d", response.Clicks)
	}
}

func TestHandler_GetStats_NotFound(t *testing.T) {
	mockService := &mockShortenerService{
		getStatsFunc: func(ctx context.Context, shortCode string) (*domain.StatsResponse, error) {
			return nil, echo.NewHTTPError(http.StatusNotFound, "short URL not found")
		},
	}

	handler := NewHandler(mockService)
	e := setupEcho()

	rec, c := testRequestWithParam(t, e, http.MethodGet, "/api/stats/notfound", "shortCode", "notfound")
	c.SetPath("/api/stats/:shortCode")

	handleRequest(t, handler.GetStats, c)
	assertStatusCode(t, rec, http.StatusNotFound)
	assertErrorResponse(t, rec, "Short URL not found")
}

func TestHandler_GetStats_InvalidShortCode(t *testing.T) {
	mockService := &mockShortenerService{
		getStatsFunc: func(ctx context.Context, shortCode string) (*domain.StatsResponse, error) {
			return nil, echo.NewHTTPError(http.StatusBadRequest, "invalid short code format")
		},
	}

	handler := NewHandler(mockService)
	e := setupEcho()

	rec, c := testRequestWithParam(t, e, http.MethodGet, "/api/stats/invalid@code!", "shortCode", "invalid@code!")
	c.SetPath("/api/stats/:shortCode")

	handleRequest(t, handler.GetStats, c)
	assertStatusCode(t, rec, http.StatusNotFound)
}

// ------------------------------------------------------------------------------------------
//                                 TESTS: HealthCheck
// ------------------------------------------------------------------------------------------

func TestHandler_HealthCheck_Success(t *testing.T) {
	mockService := &mockShortenerService{}
	handler := NewHandler(mockService)
	e := setupEcho()

	rec, c := testRequest(t, e, http.MethodGet, "/health", "")

	handleRequest(t, handler.HealthCheck, c)
	assertStatusCode(t, rec, http.StatusOK)

	var response map[string]string
	assertJSONResponse(t, rec, &response)

	if response["status"] != "healthy" {
		t.Errorf("expected status 'healthy', got '%s'", response["status"])
	}
}

// ------------------------------------------------------------------------------------------
//                              TESTS: Table-Driven
// ------------------------------------------------------------------------------------------

func TestHandler_CreateShortURL_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    string
		mockService    *mockShortenerService
		expectedStatus int
		expectedError  string
	}{
		{
			name:        "valid request",
			requestBody: `{"long_url": "https://example.com"}`,
			mockService: &mockShortenerService{
				createFunc: func(ctx context.Context, longURL string) (*domain.CreateURLResponse, error) {
					return &domain.CreateURLResponse{
						ShortCode: "abc123",
						ShortURL:  "http://localhost:8080/abc123",
						LongURL:   longURL,
					}, nil
				},
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "empty request body",
			requestBody:    `""`,
			mockService:    &mockShortenerService{},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request payload",
		},
		{
			name:           "invalid json",
			requestBody:    `{"long_url": }`,
			mockService:    &mockShortenerService{},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request payload",
		},
		{
			name:           "empty json object",
			requestBody:    `{}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Validation failed",
		},
		{
			name:           "malformed json",
			requestBody:    `{not valid json}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request payload",
		},
		{
			name:           "missing long_url field",
			requestBody:    `{"url": "https://example.com"}`,
			mockService:    &mockShortenerService{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(tt.mockService)
			e := setupEcho()

			rec, c := testRequest(t, e, http.MethodPost, "/api/shorten", tt.requestBody)

			handleRequest(t, handler.CreateShortURL, c)
			assertStatusCode(t, rec, tt.expectedStatus)

			if tt.expectedError != "" {
				assertErrorResponse(t, rec, tt.expectedError)
			}
		})
	}
}

// ------------------------------------------------------------------------------------------
//                                    HELPER FUNCTIONS
// ------------------------------------------------------------------------------------------

func setupEcho() *echo.Echo {
	e := echo.New()
	e.Validator = &CustomValidator{
		validator: validator.New(),
	}
	return e
}

// testRequest executes an HTTP request and returns the recorder
func testRequest(t *testing.T, e *echo.Echo, method, path, body string) (*httptest.ResponseRecorder, echo.Context) {
	t.Helper()

	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	return rec, c
}

// testRequestWithParam executes an HTTP request with path parameters
func testRequestWithParam(t *testing.T, e *echo.Echo, method, path, paramName, paramValue string) (*httptest.ResponseRecorder, echo.Context) {
	t.Helper()

	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/:shortCode")
	c.SetParamNames(paramName)
	c.SetParamValues(paramValue)

	return rec, c
}

// assertStatusCode checks if the response status code matches expected
func assertStatusCode(t *testing.T, rec *httptest.ResponseRecorder, expected int) {
	t.Helper()
	if rec.Code != expected {
		t.Errorf("expected status %d, got %d", expected, rec.Code)
	}
}

// assertJSONResponse unmarshal and returns the JSON response
func assertJSONResponse(t *testing.T, rec *httptest.ResponseRecorder, v interface{}) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), v); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
}

// assertErrorResponse checks if the response contains an error message
func assertErrorResponse(t *testing.T, rec *httptest.ResponseRecorder, expectedMsg string) {
	t.Helper()
	var response map[string]string
	assertJSONResponse(t, rec, &response)

	if errMsg, exists := response["error"]; exists {
		if expectedMsg != "" && !strings.Contains(errMsg, expectedMsg) {
			t.Errorf("expected error containing '%s', got '%s'", expectedMsg, errMsg)
		}
	} else {
		t.Error("expected error field in response")
	}
}

// assertHeader checks if a header matches the expected value
func assertHeader(t *testing.T, rec *httptest.ResponseRecorder, header, expected string) {
	t.Helper()
	actual := rec.Header().Get(header)
	if actual != expected {
		t.Errorf("expected header %s='%s', got '%s'", header, expected, actual)
	}
}

// handleRequest executes a handler and returns an error if any
func handleRequest(t *testing.T, handler func(echo.Context) error, c echo.Context) {
	t.Helper()
	if err := handler(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
