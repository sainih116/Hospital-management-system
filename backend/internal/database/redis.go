package database

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"hms-backend/internal/config"
)

// ConnectRedis establishes a connection to Redis when the cache layer is
// enabled via configuration. Callers should treat a nil client as "no cache
// available" rather than an error condition.
func ConnectRedis(ctx context.Context, cfg config.RedisConfig) (*redis.Client, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis: ping: %w", err)
	}

	return client, nil
}

// DisconnectRedis closes the Redis client connection if one is present.
func DisconnectRedis(client *redis.Client) error {
	if client == nil {
		return nil
	}
	return client.Close()
}
