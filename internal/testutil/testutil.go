package testutil

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/wuekevin/chatsphere/internal/model"
)

// MustDB opens a TiDB connection for tests.
// Reads TEST_TIDB_DSN from env, falls back to component parts:
//   TEST_TIDB_HOST, TEST_TIDB_PORT, TEST_TIDB_USER, TEST_TIDB_PASS, TEST_TIDB_NAME
func MustDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_TIDB_DSN")
	if dsn == "" {
		host := getenv("TEST_TIDB_HOST", "127.0.0.1")
		port := getenv("TEST_TIDB_PORT", "4000")
		user := getenv("TEST_TIDB_USER", "root")
		pass := os.Getenv("TEST_TIDB_PASS")
		name := getenv("TEST_TIDB_NAME", "chatsphere_test")
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&tls=tidb", user, pass, host, port, name)
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Skipf("TiDB not available (%v) — set TEST_TIDB_DSN or TEST_TIDB_* env vars to run integration tests", err)
	}
	if err := db.AutoMigrate(
		&model.User{}, &model.Channel{}, &model.Message{},
		&model.Report{}, &model.DirectMessage{}, &model.ReputationTier{},
		&model.BlockedWord{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { cleanDB(db) })
	return db
}

// MustRedis opens an Upstash/Redis client for tests.
// Reads TEST_REDIS_ADDR and TEST_REDIS_PASS from env.
// Skips the test if TEST_REDIS_ADDR is unset.
func MustRedis(t *testing.T) *redis.Client {
	t.Helper()
	addr := os.Getenv("TEST_REDIS_ADDR")
	pass := os.Getenv("TEST_REDIS_PASS")
	if addr == "" {
		t.Skip("TEST_REDIS_ADDR not set — skipping (set TEST_REDIS_ADDR and TEST_REDIS_PASS to run)")
	}
	useTLS := strings.Contains(addr, "upstash.io")
	opt := &redis.Options{Addr: addr, Password: pass}
	if useTLS {
		opt.TLSConfig = tlsConfig()
	}
	rdb := redis.NewClient(opt)
	t.Cleanup(func() { rdb.FlushDB(context.Background()) })
	return rdb
}

func cleanDB(db *gorm.DB) {
	tables := []string{
		"direct_messages", "reports", "messages",
		"blocked_words", "reputation_tiers", "channels", "users",
	}
	db.Exec("SET FOREIGN_KEY_CHECKS=0")
	for _, tbl := range tables {
		db.Exec("TRUNCATE TABLE " + tbl)
	}
	db.Exec("SET FOREIGN_KEY_CHECKS=1")
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
