package api

import (
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

// SetupRoutes configures the API routes and middleware.
// It takes an Echo instance and a Handler as parameters.
// It sets up middlewares for logging, recovery, CORS, and rate limiting.
// It also defines the routes for health checks, URL shortening, redirection, and statistics retrieval.
func SetupRoutes(e *echo.Echo, handler *Handler) {
	e.Validator = &CustomValidator{
		validator: validator.New(),
	}

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		Timeout: 30 * time.Second,
	}))
	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(100)))

	e.GET("/health", handler.HealthCheck)
	e.GET("/:shortCode", handler.Redirect)

	api := e.Group("/api")
	{
		api.POST("/shorten", handler.CreateShortURL)
		api.GET("/stats/:shortCode", handler.GetStats)
	}
}
