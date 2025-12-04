package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/Elisandil/go-snap/internal/api"
	"github.com/Elisandil/go-snap/internal/repo"
	"github.com/Elisandil/go-snap/internal/service"
	"github.com/Elisandil/go-snap/internal/shortid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {

	if err := godotenv.Load(); err != nil {

		if err := godotenv.Load("../../.env"); err != nil {
			log.Warn().Msg("no .env file found, relying on environment variables")
		}
	}

	logLevel := getEnv("LOG_LEVEL")
	logFormat := getEnv("LOG_FORMAT")

	setupLogger(logLevel, logFormat)

	if err := validateConfig(); err != nil {
		log.Fatal().Err(err).Msg("configuration validation failed")
	}

	// Connect to Postgres
	pgPool, err := connectPostgres()
	if err != nil {
		log.Fatal().Err(err).Msg("error connecting to Postgres")
	}
	defer pgPool.Close()

	// Connect to Redis
	redisClient := connectRedis()
	defer func(redisClient *redis.Client) {
		err := redisClient.Close()
		if err != nil {
			log.Error().Err(err).Msg("error closing Redis client")
		}
	}(redisClient)

	// Test connections
	ctx := context.Background()
	if err := pgPool.Ping(ctx); err != nil {
		log.Fatal().Err(err).Msg("error pinging Postgres")
	}
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatal().Err(err).Msg("error pinging Redis")
	}

	log.Info().Msg("Successfully connected to Postgres and Redis")

	// Initialize repositories, services, and handlers
	pgRepo := repo.NewPostgresRepo(pgPool)
	redisRepo := repo.NewRedisRepo(redisClient, 24*time.Hour)
	generator := shortid.NewGenerator()
	baseURL := getEnv("SERVER_BASE_URL")
	shortenerService := service.NewShortenerService(pgRepo, redisRepo, generator, baseURL)
	handler := api.NewHandler(shortenerService)

	// Setup and start the Echo server
	e := echo.New()
	e.HideBanner = true
	api.SetupRoutes(e, handler)

	port := getEnv("SERVER_PORT")
	go func() {
		log.Info().Str("port", port).Str("base_url", baseURL).Msg("starting the server")
		if err := e.Start(":" + port); err != nil {
			log.Error().Err(err).Msg("server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	log.Info().Msg("shutting down the server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("error during server shutdown")
	}

	log.Info().Msg("server stopped gracefully")
}

// ---------------------------------------------------------------------------------------
//                                    PRIVATE FUNCTIONS
// ---------------------------------------------------------------------------------------

// getEnv retrieves the value of the environment variable named by the key.
func getEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatal().Msgf("environment variable %s is not set", key)
	}
	return value
}

// getEnvAsInt retrieves the value of the environment variable named by the key and converts it to an integer.
func getEnvAsInt(key string) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		log.Fatal().Msgf("environment variable %s is not set", key)
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		log.Fatal().Msgf("environment variable %s must be an integer", key)
	}
	return value
}

// validateConfig checks for the presence of required environment variables.
func validateConfig() error {
	requiredKeys := []string{
		"POSTGRES_HOST",
		"POSTGRES_PORT",
		"POSTGRES_USER",
		"POSTGRES_PASSWORD",
		"POSTGRES_DATABASE",
		"REDIS_HOST",
		"REDIS_PORT",
	}

	for _, key := range requiredKeys {
		if os.Getenv(key) == "" {
			return fmt.Errorf("missing required environment variable: %s", key)
		}
	}

	return nil
}

// setupLogger configures the global logger based on environment variables.
func setupLogger(level, format string) {
	switch level {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	if format == "console" {
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: time.RFC3339,
		})
	} else {
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	}
}

// connectPostgres establishes a connection to the Postgres database.
func connectPostgres() (*pgxpool.Pool, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		getEnv("POSTGRES_HOST"),
		getEnv("POSTGRES_PORT"),
		getEnv("POSTGRES_USER"),
		getEnv("POSTGRES_PASSWORD"),
		getEnv("POSTGRES_DATABASE"),
	)

	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, err
	}
	config.MaxConns = int32(getEnvAsInt("POSTGRES_MAX_CONNECTIONS"))
	config.MinConns = int32(getEnvAsInt("POSTGRES_MIN_CONNECTIONS"))

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}

	return pool, nil
}

// connectRedis establishes a connection to the Redis server.
func connectRedis() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", getEnv("REDIS_HOST"), getEnv("REDIS_PORT")),
		Password: getEnv("REDIS_PASSWORD"),
		DB:       getEnvAsInt("REDIS_DB"),
		PoolSize: getEnvAsInt("REDIS_POOL_SIZE"),
	})
}
