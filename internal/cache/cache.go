package cache

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/wuekevin/chatsphere/internal/config"
)

// Connect creates and validates a Redis client from configuration.
func Connect(cfg config.CacheConfig) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return rdb, nil
}

// ConnectAddr opens a Redis connection to a raw address (used by the installer).
func ConnectAddr(addr, password string, db int) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}
	return rdb, nil
}
