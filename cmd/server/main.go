package server

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/Elisandil/GoSnap/internal/api"
	"github.com/Elisandil/GoSnap/internal/repo"
	"github.com/Elisandil/GoSnap/internal/service"
	"github.com/Elisandil/GoSnap/internal/shortid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out: os.Stderr,
	})

	// Load configuration
	if err := loadConfig(); err != nil {
		log.Fatal().Err(err).Msg("error loading config")
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
	baseURL := viper.GetString("server.base_url")
	shortenerService := service.NewShortenerService(pgRepo, redisRepo, generator, baseURL)
	handler := api.NewHandler(shortenerService)

	// Setup and start the Echo server
	e := echo.New()
	e.HideBanner = true
	api.SetupRoutes(e, handler)

	port := viper.GetString("server.port")
	go func() {
		log.Info().Str("port", port).Msg("starting the server")
		if err := e.Start(":" + port); err != nil {
			log.Error().Err(err).Msg("error starting the server")
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

// loadConfig loads the configuration from file and environment variables.
func loadConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")

	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.base_url", "http://localhost")
	viper.SetDefault("postgres.host", "localhost")
	viper.SetDefault("postgres.port", "5432")
	viper.SetDefault("postgres.user", "postgres")
	viper.SetDefault("postgres.password", "postgres")
	viper.SetDefault("postgres.database", "url_shortener")
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", "6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			log.Warn().Msg("Config file not found; using default values and environment variables")
			return nil
		}
		return err
	}

	return nil
}

// connectPostgres establishes a connection pool to the PostgreSQL database.
// It returns the connection pool or an error if the connection fails.
func connectPostgres() (*pgxpool.Pool, error) {
	connectionString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbanme=%s sslmode=disable",
		viper.GetString("postgres.host"),
		viper.GetString("postgres.port"),
		viper.GetString("postgres.user"),
		viper.GetString("postgres.password"),
		viper.GetString("postgres.database"),
	)

	config, err := pgxpool.ParseConfig(connectionString)
	if err != nil {
		return nil, err
	}
	config.MaxConns = 25
	config.MinConns = 1

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}

	return pool, nil
}

// connectRedis establishes a connection to the Redis server.
// It returns the Redis client.
func connectRedis() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", viper.GetString("redis.host"), viper.GetString("redis.port")),
		Password: viper.GetString("redis.password"),
		DB:       viper.GetInt("redis.db"),
	})
}
