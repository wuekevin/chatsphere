package handler

import (
	"bytes"
	"fmt"
	"io/fs"
	"net/http"
	"regexp"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/tdewolff/minify/v2"
	jsminify "github.com/tdewolff/minify/v2/js"
	"gorm.io/gorm"

	"github.com/wuekevin/chatsphere/internal/config"
	"github.com/wuekevin/chatsphere/internal/middleware"
	"github.com/wuekevin/chatsphere/internal/model"
	"github.com/wuekevin/chatsphere/web"
)

// RegisterAdmin wires admin API routes (all require admin role).
func RegisterAdmin(r *gin.Engine, db *gorm.DB, rdb *redis.Client, cfg *config.Config) {
	admin := r.Group(cfg.Admin.Path+"/api", middleware.AdminOnly(cfg.Server.SecretKey))

	// Dashboard stats
	admin.GET("/stats", statsHandler(db))

	// Channels
	admin.GET("/channels", adminListChannels(db))
	admin.POST("/channels", adminCreateChannel(db))
	admin.PUT("/channels/:id", adminUpdateChannel(db))
	admin.DELETE("/channels/:id", adminDeleteChannel(db))

	// Messages
	admin.GET("/messages", adminListMessages(db))
	admin.DELETE("/messages/:id", adminDeleteMessage(db))

	// Users
	admin.GET("/users", adminListUsers(db))
	admin.PUT("/users/:id/ban", adminBanUser(db))
	admin.PUT("/users/:id/unban", adminUnbanUser(db))
	admin.PUT("/users/:id/reputation", adminSetReputation(db))

	// Reports
	admin.GET("/reports", adminListReports(db))
	admin.PUT("/reports/:id/resolve", adminResolveReport(db))

	// Blocked words (content moderation)
	admin.GET("/blocked-words", adminListBlockedWords(db))
	admin.POST("/blocked-words", adminCreateBlockedWord(db))
	admin.DELETE("/blocked-words/:id", adminDeleteBlockedWord(db))

	// Embed config
	admin.GET("/embed-config", getEmbedConfig(db))
	admin.PUT("/embed-config", saveEmbedConfig(db))
}

// RegisterWidget serves the admin SPA and the embeddable widget files.
func RegisterWidget(r *gin.Engine, db *gorm.DB, cfg *config.Config) {
	// Admin SPA — single index.html embedded in binary.
	// Use exact routes only: a wildcard under cfg.Admin.Path conflicts with
	// the /api sub-routes registered by RegisterAdmin in gin's radix tree.
	adminHTML, _ := web.AdminHTML.ReadFile("admin/index.html")
	serveAdmin := func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", adminHTML)
	}
	r.GET(cfg.Admin.Path, serveAdmin)
	r.GET(cfg.Admin.Path+"/", serveAdmin)

	// Dynamic widget JS (minified + config-baked). Must register before StaticFS
	// so this exact route takes priority over the static file handler.
	r.GET("/widget/chatsphere.js", serveWidgetJS(db))

	// Widget assets — embedded in binary so the production binary is self-contained.
	widgetSub, _ := fs.Sub(web.WidgetFS, "widget")
	r.StaticFS("/widget", http.FS(widgetSub))
}

// --- Stats ---

func statsHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var userCount, messageCount, channelCount, reportCount int64
		db.Model(&model.User{}).Count(&userCount)
		db.Model(&model.Message{}).Where("is_deleted = false").Count(&messageCount)
		db.Model(&model.Channel{}).Where("is_active = true").Count(&channelCount)
		db.Model(&model.Report{}).Where("status = 'pending'").Count(&reportCount)

		c.JSON(http.StatusOK, gin.H{
			"users":           userCount,
			"messages":        messageCount,
			"channels":        channelCount,
			"pending_reports": reportCount,
		})
	}
}

// --- Channels ---

func adminListChannels(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var channels []model.Channel
		db.Order("created_at DESC").Find(&channels)
		c.JSON(http.StatusOK, gin.H{"channels": channels})
	}
}

type channelRequest struct {
	Name             string `json:"name" binding:"required"`
	Slug             string `json:"slug" binding:"required"`
	Description      string `json:"description"`
	IsDefault        bool   `json:"is_default"`
	IsActive         bool   `json:"is_active"`
	MaxMsgLength     int    `json:"max_msg_length"`
	RateLimitSeconds int    `json:"rate_limit_seconds"`
	AllowAnonymous   bool   `json:"allow_anonymous"`
	WelcomeMessage   string `json:"welcome_message"`
	TelegramURL      string `json:"telegram_url"`
	AccentColor      string `json:"accent_color"`
}

func adminCreateChannel(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req channelRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ch := model.Channel{
			Name:             req.Name,
			Slug:             req.Slug,
			Description:      req.Description,
			IsDefault:        req.IsDefault,
			IsActive:         req.IsActive,
			MaxMsgLength:     req.MaxMsgLength,
			RateLimitSeconds: req.RateLimitSeconds,
			AllowAnonymous:   req.AllowAnonymous,
			WelcomeMessage:   req.WelcomeMessage,
			TelegramURL:      req.TelegramURL,
			AccentColor:      req.AccentColor,
		}

		if err := db.Create(&ch).Error; err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": "slug already exists"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"channel": ch})
	}
}

func adminUpdateChannel(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		var ch model.Channel
		if err := db.First(&ch, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		var req channelRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := db.Model(&ch).Updates(model.Channel{
			Name:             req.Name,
			Slug:             req.Slug,
			Description:      req.Description,
			IsDefault:        req.IsDefault,
			IsActive:         req.IsActive,
			MaxMsgLength:     req.MaxMsgLength,
			RateLimitSeconds: req.RateLimitSeconds,
			AllowAnonymous:   req.AllowAnonymous,
			WelcomeMessage:   req.WelcomeMessage,
			TelegramURL:      req.TelegramURL,
			AccentColor:      req.AccentColor,
		}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
			return
		}
		db.First(&ch, id)

		c.JSON(http.StatusOK, gin.H{"channel": ch})
	}
}

func adminDeleteChannel(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		db.Model(&model.Channel{}).Where("id = ?", id).Update("is_active", false)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// --- Messages ---

func adminListMessages(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		channelID := c.Query("channel_id")
		keyword := c.Query("q")
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		if page < 1 {
			page = 1
		}
		limit := 50
		offset := (page - 1) * limit

		query := db.Where("is_deleted = false").Order("created_at DESC").Limit(limit).Offset(offset)
		if channelID != "" {
			query = query.Where("channel_id = ?", channelID)
		}
		if keyword != "" {
			query = query.Where("content LIKE ?", "%"+keyword+"%")
		}

		var messages []model.Message
		query.Find(&messages)
		c.JSON(http.StatusOK, gin.H{"messages": messages})
	}
}

func adminDeleteMessage(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		db.Model(&model.Message{}).Where("id = ?", id).Update("is_deleted", true)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// --- Users ---

func adminListUsers(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		if page < 1 {
			page = 1
		}
		limit := 50
		offset := (page - 1) * limit

		var users []model.User
		db.Order("created_at DESC").Limit(limit).Offset(offset).Find(&users)
		c.JSON(http.StatusOK, gin.H{"users": users})
	}
}

func adminBanUser(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

		// Prevent banning another admin
		claims := middleware.GetClaims(c)
		if claims != nil && claims.UserID == id {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot ban yourself"})
			return
		}

		db.Model(&model.User{}).Where("id = ? AND role != 'admin'", id).Update("is_banned", true)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func adminUnbanUser(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		db.Model(&model.User{}).Where("id = ?", id).Update("is_banned", false)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func adminSetReputation(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		var body struct {
			Value int `json:"value"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		db.Model(&model.User{}).Where("id = ?", id).Update("reputation", body.Value)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// --- Reports ---

func adminListReports(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := c.DefaultQuery("status", "pending")
		var reports []model.Report
		db.Where("status = ?", status).Order("created_at DESC").Limit(100).Find(&reports)
		c.JSON(http.StatusOK, gin.H{"reports": reports})
	}
}

func adminResolveReport(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		var body struct {
			Status string `json:"status"` // "resolved" or "dismissed"
		}
		c.ShouldBindJSON(&body)
		if body.Status == "" {
			body.Status = "resolved"
		}
		db.Model(&model.Report{}).Where("id = ?", id).Update("status", body.Status)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// --- Blocked Words ---

func adminListBlockedWords(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var words []model.BlockedWord
		q := db.Where("is_active = true")
		if ch := c.Query("channel_id"); ch != "" {
			q = q.Where("channel_id = ? OR channel_id IS NULL", ch)
		}
		q.Order("created_at DESC").Find(&words)
		c.JSON(http.StatusOK, gin.H{"blocked_words": words})
	}
}

func adminCreateBlockedWord(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			ChannelID *uint64 `json:"channel_id"`
			Pattern   string  `json:"pattern" binding:"required"`
			IsRegex   bool    `json:"is_regex"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		word := model.BlockedWord{
			ChannelID: body.ChannelID,
			Pattern:   body.Pattern,
			IsRegex:   body.IsRegex,
			IsActive:  true,
		}
		if err := db.Create(&word).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"blocked_word": word})
	}
}

func adminDeleteBlockedWord(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		db.Model(&model.BlockedWord{}).Where("id = ?", id).Update("is_active", false)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// --- Embed Config ---

func getEmbedConfig(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var cfg model.EmbedConfig
		db.FirstOrCreate(&cfg, model.EmbedConfig{ID: 1})
		c.JSON(http.StatusOK, gin.H{"config": cfg})
	}
}

func saveEmbedConfig(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body model.EmbedConfig
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		body.ID = 1
		if err := db.Save(&body).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "save failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"config": body})
	}
}

var jsMinifier = func() *minify.M {
	m := minify.New()
	m.AddFuncRegexp(regexp.MustCompile("^application/javascript$"), jsminify.Minify)
	return m
}()

func serveWidgetJS(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, err := web.WidgetFS.ReadFile("widget/chatsphere.js")
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		var cfg model.EmbedConfig
		db.FirstOrCreate(&cfg, model.EmbedConfig{ID: 1})

		serverURL := cfg.ServerURL
		if serverURL == "" {
			scheme := "http"
			if c.Request.TLS != nil {
				scheme = "https"
			}
			serverURL = fmt.Sprintf("%s://%s", scheme, c.Request.Host)
		}

		snippet := fmt.Sprintf(
			"\n;(function(){ChatSphere.init({server:%q,channel:%q,position:%q,offset:%d,title:%q,notice:%q,accent:%q,lang:%q});}());\n",
			serverURL,
			cfg.Channel,
			cfg.Position,
			cfg.Offset,
			cfg.Title,
			cfg.Notice,
			cfg.AccentColor,
			cfg.Lang,
		)

		combined := append(raw, []byte(snippet)...)

		var buf bytes.Buffer
		if err := jsMinifier.Minify("application/javascript", &buf, bytes.NewReader(combined)); err != nil {
			c.Data(http.StatusOK, "application/javascript; charset=utf-8", combined)
			return
		}

		c.Header("Cache-Control", "public, max-age=300")
		c.Data(http.StatusOK, "application/javascript; charset=utf-8", buf.Bytes())
	}
}
