package testutil

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	mysqldrv "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/wuekevin/chatsphere/internal/model"
)

var registerTLSOnce sync.Once

func registerTiDBTLS() {
	registerTLSOnce.Do(func() {
		_ = mysqldrv.RegisterTLSConfig("tidb", &tls.Config{
			MinVersion: tls.VersionTLS12,
		})
	})
}

// MustDB opens a TiDB connection for tests.
// Reads TEST_TIDB_DSN from env, falls back to component parts:
//
//	TEST_TIDB_HOST, TEST_TIDB_PORT, TEST_TIDB_USER, TEST_TIDB_PASS, TEST_TIDB_NAME
func MustDB(t *testing.T) *gorm.DB {
	t.Helper()
	registerTiDBTLS()
	dsn := os.Getenv("TEST_TIDB_DSN")
	dbName := "chatsphere_test"
	if dsn == "" {
		host := getenv("TEST_TIDB_HOST", "127.0.0.1")
		port := getenv("TEST_TIDB_PORT", "4000")
		user := getenv("TEST_TIDB_USER", "root")
		pass := os.Getenv("TEST_TIDB_PASS")
		dbName = getenv("TEST_TIDB_NAME", "chatsphere_test")
		// Connect to default db first to create the test db if needed.
		rootDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/mysql?parseTime=true&tls=tidb", user, pass, host, port)
		if root, rerr := gorm.Open(mysql.Open(rootDSN), &gorm.Config{Logger: logger.Discard}); rerr == nil {
			root.Exec("CREATE DATABASE IF NOT EXISTS `" + dbName + "` CHARACTER SET utf8mb4")
			sqlDB, _ := root.DB()
			sqlDB.Close()
		}
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&tls=tidb", user, pass, host, port, dbName)
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Skipf("TiDB not available (%v) — set TEST_TIDB_DSN or TEST_TIDB_* env vars to run integration tests", err)
	}
	_ = dbName
	if err := db.AutoMigrate(
		&model.User{}, &model.Channel{}, &model.Message{},
		&model.Report{}, &model.DirectMessage{}, &model.ReputationTier{},
		&model.BlockedWord{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// Clean before test so stale data from prior runs can't interfere.
	cleanDB(db)
	t.Cleanup(func() { cleanDB(db) })
	return db
}

// DB returns a shared *gorm.DB for a test, reusing any already opened within the
// same test via context. Callers that need the same connection (e.g. a test that
// seeds data and then calls buildServer) should pass the db they received from
// MustDB into the builder rather than calling MustDB again.
func DB(t *testing.T) *gorm.DB { return MustDB(t) }

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
	// Delete in reverse dependency order; no TRUNCATE so we avoid FK check issues.
	tables := []string{
		"direct_messages", "reports", "messages",
		"blocked_words", "reputation_tiers", "channels", "users",
	}
	for _, tbl := range tables {
		db.Exec("DELETE FROM `" + tbl + "`")
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
