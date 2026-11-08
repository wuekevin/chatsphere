package installer

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
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
)

// Run starts an HTTP server that serves the installation wizard.
// It blocks until installation completes or fails, then returns.
func Run(cfgPath string) error {
	done := make(chan error, 1)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	r.Static("/setup", "./web/installer")
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/setup")
	})

	r.POST("/api/setup/test-db", testDBHandler())
	r.POST("/api/setup/test-redis", testRedisHandler())
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

	err := <-done

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

		// 1. Connect to database
		dsn := database.BuildDSN(req.DBHost, req.DBPort, req.DBUser, req.DBPassword, req.DBName)
		db, err := database.ConnectDSN(dsn)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"ok": false, "step": "database", "error": err.Error()})
			return
		}

		// 2. Run migrations
		if err := runMigrations(db); err != nil {
			c.JSON(http.StatusOK, gin.H{"ok": false, "step": "migrations", "error": err.Error()})
			return
		}

		// 3. Create admin user
		hash, err := bcrypt.GenerateFromPassword([]byte(req.AdminPassword), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "password hash failed"})
			return
		}

		admin := &model.User{
			Username:     req.AdminUsername,
			PasswordHash: string(hash),
			Email:        req.AdminEmail,
			AvatarColor:  "#22c55e",
			Role:         model.RoleAdmin,
		}

		// If an admin already exists (re-install), skip creation
		var existing model.User
		if db.Where("role = ?", model.RoleAdmin).First(&existing).Error == gorm.ErrRecordNotFound {
			if err := db.Create(admin).Error; err != nil {
				c.JSON(http.StatusOK, gin.H{"ok": false, "step": "admin_user", "error": err.Error()})
				return
			}
		}

		// 4. Create default channel
		ch := &model.Channel{
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
		if db.Where("slug = ?", req.ChannelSlug).First(&existingCh).Error == gorm.ErrRecordNotFound {
			db.Create(ch)
		}

		// 5. Build and save config
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
			c.JSON(http.StatusOK, gin.H{"ok": false, "step": "config", "error": err.Error()})
			return
		}

		log.Println("[installer] Installation complete — config.yaml written")
		c.JSON(http.StatusOK, gin.H{"ok": true})

		// Signal the main goroutine that installation is finished
		go func() { done <- nil }()
	}
}

func runMigrations(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.User{},
		&model.Channel{},
		&model.Message{},
		&model.Report{},
	)
}

func generateSecret() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
