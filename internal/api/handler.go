package api

import (
	"net/http"

	"github.com/Elisandil/GoSnap/internal/domain"
	"github.com/Elisandil/GoSnap/internal/service"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

type Handler struct {
	service *service.ShortenerService
}

func NewHandler(service *service.ShortenerService) *Handler {
	return &Handler{
		service: service,
	}
}

// CreateShortURL handles the creation of a new short URL.
// @Summary Create Short URL
// @Description Create a new short URL from a long URL
// @Param request body domain.CreateURLRequest true "Create URL Request"
// @Accept json
// @Produce json
// @Success 201 {object} domain.CreateURLResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
func (h *Handler) CreateShortURL(c echo.Context) error {
	var request domain.CreateURLRequest

	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request payload",
		})
	}
	if err := c.Validate(&request); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Validation failed: " + err.Error(),
		})
	}

	response, err := h.service.CreateShortURL(c.Request().Context(), request.LongURL)
	if err != nil {
		log.Error().Err(err).Msg("error creating short URL")

		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create short URL",
		})
	}

	return c.JSON(http.StatusCreated, response)
}

// Redirect handles the redirection from a short URL to the original long URL.
// @Summary Redirect to Long URL
// @Description Redirect from a short URL to the original long URL
// @Param shortCode path string true "Short URL code"
// @Success 302
// @Failure 404 {object} map[string]string
func (h *Handler) Redirect(c echo.Context) error {
	shortCode := c.Param("shortCode")

	longURL, err := h.service.GetLongURL(c.Request().Context(), shortCode)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Short URL not found",
		})
	}

	return c.Redirect(http.StatusFound, longURL)
}

// GetStats handles retrieving statistics for a short URL.
// @Summary Get Short URL Stats
// @Description Retrieve statistics for a short URL
// @Param shortCode path string true "Short URL code"
// @Success 200 {object} domain.URLStats
// @Failure 404 {object} map[string]string
func (h *Handler) GetStats(c echo.Context) error {
	shortCode := c.Param("shortCode")

	stats, err := h.service.GetURLStats(c.Request().Context(), shortCode)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Short URL not found",
		})
	}

	return c.JSON(http.StatusOK, stats)
}

// HealthCheck handles the health check endpoint.
// @Summary Health Check
// @Description Check the health status of the service
// @Success 200 {object} map[string]string
func (h *Handler) HealthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status": "healthy",
	})
}
