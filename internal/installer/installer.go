package installer

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/wuekevin/chatsphere/internal/cache"
	"github.com/wuekevin/chatsphere/internal/config"
	"github.com/wuekevin/chatsphere/internal/database"
	"github.com/wuekevin/chatsphere/internal/model"
	"github.com/wuekevin/chatsphere/web"
)

// Run starts an HTTP server that serves the installation wizard.
// It blocks until installation completes or fails, then returns.
func Run(cfgPath string) error {
	done := make(chan error, 1)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	subFS, err := fs.Sub(web.InstallerFS, "installer")
	if err != nil {
		return fmt.Errorf("embed installer: %w", err)
	}
	r.StaticFS("/setup", http.FS(subFS))
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/setup")
	})
	r.NoRoute(func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/setup")
	})

	r.POST("/setup/test-db", testDBHandler())
	r.POST("/setup/test-redis", testRedisHandler())
	r.POST("/api/setup/install", installHandler(cfgPath, done))

	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			done <- err
		}
	}()

	err = <-done

	shutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	srv.Shutdown(shutCtx)

	return err
}

// --- Test DB ---

type dbTestRequest struct {
	Host     string `json:"host" binding:"required"`
	Port     int    `json:"port" binding:"required"`
	Name     string `json:"name" binding:"required"`
	User     string `json:"user" binding:"required"`
	Password string `json:"password"`
}

func testDBHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dbTestRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		dsn := database.BuildDSN(req.Host, req.Port, req.User, req.Password, req.Name)
		db, err := database.ConnectDSN(dsn)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"ok": false, "error": err.Error()})
			return
		}

		sqlDB, _ := db.DB()
		sqlDB.Close()

		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// --- Test Redis ---

type redisTestRequest struct {
	Host     string `json:"host" binding:"required"`
	Port     int    `json:"port" binding:"required"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

func testRedisHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req redisTestRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		addr := fmt.Sprintf("%s:%d", req.Host, req.Port)
		rdb, err := cache.ConnectAddr(addr, req.Password, req.DB)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"ok": false, "error": err.Error()})
			return
		}
		rdb.Close()

		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// --- Install ---

type installRequest struct {
	// Server
	ServerHost string `json:"server_host" binding:"required"`
	ServerPort int    `json:"server_port" binding:"required"`
	Debug      bool   `json:"debug"`

	// Database
	DBHost     string `json:"db_host" binding:"required"`
	DBPort     int    `json:"db_port" binding:"required"`
	DBName     string `json:"db_name" binding:"required"`
	DBUser     string `json:"db_user" binding:"required"`
	DBPassword string `json:"db_password"`

	// Redis
	RedisHost     string `json:"redis_host" binding:"required"`
	RedisPort     int    `json:"redis_port" binding:"required"`
	RedisPassword string `json:"redis_password"`
	RedisDB       int    `json:"redis_db"`

	// Admin account
	AdminUsername string `json:"admin_username" binding:"required,min=3,max=30"`
	AdminPassword string `json:"admin_password" binding:"required,min=6"`
	AdminEmail    string `json:"admin_email"`

	// Site
	SiteName      string `json:"site_name" binding:"required"`
	Notice        string `json:"notice"`
	ChannelName   string `json:"channel_name" binding:"required"`
	ChannelSlug   string `json:"channel_slug" binding:"required"`
	TelegramURL   string `json:"telegram_url"`
	AllowAnonymous bool  `json:"allow_anonymous"`
}

func installHandler(cfgPath string, done chan<- error) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req installRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("X-Accel-Buffering", "no")

		events := make(chan string, 20)
		go func() {
			defer close(events)
			runInstall(req, cfgPath, done, events)
		}()

		c.Stream(func(w io.Writer) bool {
			event, ok := <-events
			if !ok {
				return false
			}
			fmt.Fprintf(w, "data: %s\n\n", event)
			return true
		})
	}
}

func runInstall(req installRequest, cfgPath string, done chan<- error, events chan<- string) {
	fail := func(msg string, err error) {
		b, _ := json.Marshal(map[string]string{"type": "error", "msg": msg})
		events <- string(b)
		done <- err
	}
	emit := func(typ, msg string) {
		b, _ := json.Marshal(map[string]string{"type": typ, "msg": msg})
		events <- string(b)
	}

	// 1. Connect to database
	emit("log", "Connecting to database…")
	dsn := database.BuildDSN(req.DBHost, req.DBPort, req.DBUser, req.DBPassword, req.DBName)
	db, err := database.ConnectDSN(dsn)
	if err != nil {
		fail(err.Error(), err)
		return
	}
	defer func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	}()
	emit("log", "Database connected ✓")

	// 2. Run migrations
	emit("log", "Running migrations…")
	if err := runMigrations(db); err != nil {
		fail("Migration failed: "+err.Error(), err)
		return
	}
	emit("log", "Migrations complete ✓")

	// 3. Create admin user
	emit("log", "Creating admin account…")
	hash, err := bcrypt.GenerateFromPassword([]byte(req.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		fail("Password hash failed", err)
		return
	}
	admin := &model.User{
		Username:     req.AdminUsername,
		PasswordHash: string(hash),
		Email:        req.AdminEmail,
		AvatarColor:  "#f5a623",
		Role:         model.RoleAdmin,
	}
	var existing model.User
	existingErr := db.Where("role = ?", model.RoleAdmin).First(&existing).Error
	if existingErr != nil && !errors.Is(existingErr, gorm.ErrRecordNotFound) {
		fail("Failed to check existing admin: "+existingErr.Error(), existingErr)
		return
	}
	if errors.Is(existingErr, gorm.ErrRecordNotFound) {
		if err := db.Create(admin).Error; err != nil {
			fail("Failed to create admin: "+err.Error(), err)
			return
		}
	}

	// 4. Create default channel
	emit("log", "Creating default channel…")
	defaultCh := &model.Channel{
		Name:             req.ChannelName,
		Slug:             req.ChannelSlug,
		IsDefault:        true,
		IsActive:         true,
		MaxMsgLength:     200,
		RateLimitSeconds: 3,
		AllowAnonymous:   req.AllowAnonymous,
		TelegramURL:      req.TelegramURL,
	}
	var existingCh model.Channel
	existingChErr := db.Where("slug = ?", req.ChannelSlug).First(&existingCh).Error
	if existingChErr != nil && !errors.Is(existingChErr, gorm.ErrRecordNotFound) {
		fail("Failed to check existing channel: "+existingChErr.Error(), existingChErr)
		return
	}
	if errors.Is(existingChErr, gorm.ErrRecordNotFound) {
		if err := db.Create(defaultCh).Error; err != nil {
			fail("Failed to create channel: "+err.Error(), err)
			return
		}
	}

	// 5. Write config
	emit("log", "Writing config.yaml…")
	secretKey := generateSecret()
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:           req.ServerHost,
			Port:           req.ServerPort,
			Debug:          req.Debug,
			SecretKey:      secretKey,
			AllowedOrigins: []string{"*"},
		},
		Database: config.DatabaseConfig{
			Host:            req.DBHost,
			Port:            req.DBPort,
			Name:            req.DBName,
			User:            req.DBUser,
			Password:        req.DBPassword,
			MaxOpenConns:    100,
			MaxIdleConns:    10,
			ConnMaxLifetime: 3600,
		},
		Cache: config.CacheConfig{
			Host:     req.RedisHost,
			Port:     req.RedisPort,
			Password: req.RedisPassword,
			DB:       req.RedisDB,
		},
		Chat: config.ChatConfig{
			SiteName:         req.SiteName,
			Notice:           req.Notice,
			MaxMessageLength: 200,
			MaxHistory:       500,
			RateLimit:        3,
			AllowAnonymous:   req.AllowAnonymous,
		},
		Admin: config.AdminConfig{
			Path: "/admin",
		},
	}
	if err := config.Save(cfgPath, cfg); err != nil {
		fail("Failed to write config: "+err.Error(), err)
		return
	}

	log.Println("[installer] Installation complete — config.yaml written")
	b, _ := json.Marshal(map[string]interface{}{"type": "done", "ok": true})
	events <- string(b)
	done <- nil
}

func runMigrations(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.User{},
		&model.Channel{},
		&model.Message{},
		&model.Report{},
		&model.DirectMessage{},
		&model.ReputationTier{},
	)
}

func generateSecret() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return hex.EncodeToString(b)
}
