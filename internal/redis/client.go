package redis

import (
	"context"
	"fmt"

	"github.com/mossy-p/webrtc-signaling/config"
	"github.com/redis/go-redis/v9"
)

var client *redis.Client
var ctx = context.Background()

// Connect initializes the Redis client
func Connect(cfg config.RedisConfig) error {
	client = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return nil
}

// Close closes the Redis connection
func Close() error {
	if client != nil {
		return client.Close()
	}
	return nil
}

// GetClient returns the Redis client instance
func GetClient() *redis.Client {
	return client
}

// GetContext returns the context for Redis operations
func GetContext() context.Context {
	return ctx
}
