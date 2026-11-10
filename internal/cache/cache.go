package cache

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/wuekevin/chatsphere/internal/config"
)

func redisTLSConfig() *tls.Config {
	pool, err := x509.SystemCertPool()
	if err != nil || pool == nil {
		pool = x509.NewCertPool()
	}
	return &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: pool}
}

// Connect creates and validates a Redis client from configuration.
func Connect(cfg config.CacheConfig) (*redis.Client, error) {
	opt := &redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	}
	if cfg.TLS {
		opt.TLSConfig = redisTLSConfig()
	}
	rdb := redis.NewClient(opt)

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return rdb, nil
}

// ConnectAddr opens a Redis connection to a raw address (used by the installer).
func ConnectAddr(addr, password string, db int, useTLS bool) (*redis.Client, error) {
	opt := &redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	}
	if useTLS {
		opt.TLSConfig = redisTLSConfig()
	}
	rdb := redis.NewClient(opt)
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}
	return rdb, nil
}
