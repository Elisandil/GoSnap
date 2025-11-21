package repo

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/Elisandil/GoSnap/internal/domain"
	"github.com/Elisandil/GoSnap/pkg/validator"
	"github.com/redis/go-redis/v9"
)

type RedisRepo struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisRepo(client *redis.Client, ttl time.Duration) *RedisRepo {
	return &RedisRepo{
		client: client,
		ttl:    ttl,
	}
}

// Set stores in cache the URL associated with the given short code in Redis with a TTL.
func (r *RedisRepo) Set(ctx context.Context, shortCode string, url *domain.URL) error {

	if !validator.IsValidShortCode(shortCode) {
		return ErrInvalidShortCode
	}

	data, err := json.Marshal(url)
	if err != nil {
		return err
	}

	return r.client.Set(ctx, shortCode, data, r.ttl).Err()
}

// Get retrieves from cache the URL associated with the given short code in Redis.
func (r *RedisRepo) Get(ctx context.Context, shortCode string) (*domain.URL, error) {

	if !validator.IsValidShortCode(shortCode) {
		return nil, ErrInvalidShortCode
	}

	data, err := r.client.Get(ctx, shortCode).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	var url domain.URL
	if err := json.Unmarshal(data, &url); err != nil {
		return nil, err
	}

	return &url, nil
}

// Delete removes from cache the URL associated with the given short code in Redis.
func (r *RedisRepo) Delete(ctx context.Context, shortCode string) error {

	if !validator.IsValidShortCode(shortCode) {
		return ErrInvalidShortCode
	}

	return r.client.Del(ctx, shortCode).Err()
}

// Exists checks if a short code exists in Redis.
func (r *RedisRepo) Exists(ctx context.Context, shortCode string) (bool, error) {

	if !validator.IsValidShortCode(shortCode) {
		return false, ErrInvalidShortCode
	}

	result, err := r.client.Exists(ctx, shortCode).Result()
	if err != nil {
		return false, err
	}

	return result != 0, nil
}
