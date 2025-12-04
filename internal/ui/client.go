package ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Elisandil/go-snap/internal/domain"
)

type APIClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetBaseURL sets the base URL for the API client.
func (c *APIClient) SetBaseURL(baseURL string) {
	c.baseURL = baseURL
}

// GetBaseURL returns the current base URL of the API client.
func (c *APIClient) GetBaseURL() string {
	return c.baseURL
}

// CreateShortURL sends a request to create a shortened URL for the given long URL.
// It returns a CreateURLResponse containing the shortened URL or an error if the request fails.
// The longURL parameter is the original URL to be shortened.
func (c *APIClient) CreateShortURL(longURL string) (*domain.CreateURLResponse, error) {
	requestBody := domain.CreateURLRequest{
		LongURL: longURL,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	response, err := c.httpClient.Post(
		c.baseURL+"/api/shorten",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send POST request: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("failed to close response body: %v\n", err)
		}
	}(response.Body)

	if response.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(response.Body)

		return nil, fmt.Errorf("unexpected status code: %d, body: %s", response.StatusCode, string(body))
	}

	var result domain.CreateURLResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %w", err)
	}
	return &result, nil
}

// GetStats retrieves statistics for a given short code.
// It returns a StatsResponse containing the statistics data or an error if the request fails.
// The shortCode parameter is the unique identifier for the shortened URL.
func (c *APIClient) GetStats(shortCode string) (*domain.StatsResponse, error) {
	response, err := c.httpClient.Get(c.baseURL + "/api/stats/" + shortCode)
	if err != nil {
		return nil, fmt.Errorf("failed to send GET request: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("failed to close response body: %v\n", err)
		}
	}(response.Body)

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)

		return nil, fmt.Errorf("unexpected status code: %d, body: %s", response.StatusCode, string(body))
	}

	var result domain.StatsResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %w", err)
	}
	return &result, nil
}

// HealthCheck checks if the server is reachable and healthy.
// It returns an error if the server is not healthy or unreachable.
// A healthy server should respond with HTTP 200 OK status.
func (c *APIClient) HealthCheck() error {
	response, err := c.httpClient.Get(c.baseURL + "/health")
	if err != nil {
		return fmt.Errorf("error connecting to the server: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("failed to close response body: %v\n", err)
		}
	}(response.Body)

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned non-OK status: %d", response.StatusCode)
	}

	return nil
}
