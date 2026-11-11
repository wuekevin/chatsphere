package database

import (
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	mysqldrv "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/wuekevin/chatsphere/internal/config"
	"github.com/wuekevin/chatsphere/internal/model"
)

var registerTLSOnce sync.Once

func registerTiDBTLS() {
	registerTLSOnce.Do(func() {
		_ = mysqldrv.RegisterTLSConfig("tidb", &tls.Config{MinVersion: tls.VersionTLS12})
	})
}

// Connect opens a MySQL connection and runs auto-migrations.
func Connect(cfg config.DatabaseConfig) (*gorm.DB, error) {
	tlsParam := "false"
	if cfg.TLS {
		registerTiDBTLS()
		tlsParam = "tidb"
	}
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=UTC&tls=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, tlsParam,
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)

	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return db, nil
}

// ConnectDSN opens a connection with a raw DSN (used by the installer to test).
func ConnectDSN(dsn string) (*gorm.DB, error) {
	return gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
}

// BuildDSN constructs a MySQL DSN from individual fields.
// Set useTLS=true for TiDB Cloud and other managed databases that require SSL/TLS.
func BuildDSN(host string, port int, user, password, name string, useTLS bool) string {
	tlsParam := "false"
	if useTLS {
		registerTiDBTLS()
		tlsParam = "tidb"
	}
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=UTC&tls=%s",
		user, password, host, port, name, tlsParam,
	)
}

func migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.User{},
		&model.Channel{},
		&model.Message{},
		&model.Report{},
		&model.BlockedWord{},
		&model.DirectMessage{},
		&model.EmbedConfig{},
	)
}
