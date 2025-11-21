package domain

import "time"

// URL Represents a shortened URL entity
type URL struct {
	ID        int64     `json:"id"`
	ShortCode string    `json:"short_code"`
	LongURL   string    `json:"long_url"`
	CreatedAt time.Time `json:"created_at"`
	Clicks    int64     `json:"clicks"`
}

// CreateURLRequest Represents the request payload for creating a shortened URL
type CreateURLRequest struct {
	LongURL string `json:"long_url" validate:"required,url"`
}

// CreateURLResponse Represents the response payload after creating a shortened URL
type CreateURLResponse struct {
	ShortCode string `json:"short_code"`
	ShortURL  string `json:"short_url"`
	LongURL   string `json:"long_url"`
}

// StatsResponse Represents the response payload for URL statistics
type StatsResponse struct {
	ShortCode string    `json:"short_code"`
	LongURL   string    `json:"long_url"`
	Clicks    int64     `json:"clicks"`
	CreatedAt time.Time `json:"created_at"`
}
