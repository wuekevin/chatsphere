package handler

import (
	"bytes"
	"fmt"
	"io/fs"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/tdewolff/minify/v2"
	jsminify "github.com/tdewolff/minify/v2/js"
	"gorm.io/gorm"

	"github.com/wuekevin/chatsphere/internal/bot"
	"github.com/wuekevin/chatsphere/internal/config"
	"github.com/wuekevin/chatsphere/internal/middleware"
	"github.com/wuekevin/chatsphere/internal/model"
	"github.com/wuekevin/chatsphere/internal/ws"
	"github.com/wuekevin/chatsphere/web"
)

// RegisterAdmin wires admin API routes (all require admin role).
func RegisterAdmin(r *gin.Engine, db *gorm.DB, rdb *redis.Client, cfg *config.Config, scheduler *bot.Scheduler, hubReg *ws.HubRegistry) {
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
	admin.DELETE("/messages/:id", adminDeleteMessage(db, rdb, hubReg))

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
	admin.POST("/blocked-words", adminCreateBlockedWord(db, hubReg))
	admin.DELETE("/blocked-words/:id", adminDeleteBlockedWord(db, hubReg))

	admin.GET("/allowed-domains", adminListAllowedDomains(db))
	admin.POST("/allowed-domains", adminCreateAllowedDomain(db))
	admin.DELETE("/allowed-domains/:id", adminDeleteAllowedDomain(db))

	// Embed config
	admin.GET("/embed-config", getEmbedConfig(db))
	admin.PUT("/embed-config", saveEmbedConfig(db))

	// Bots
	admin.GET("/bots", adminListBots(db))
	admin.POST("/bots", adminCreateBot(db))
	admin.PUT("/bots/:id", adminUpdateBot(db))
	admin.DELETE("/bots/:id", adminDeleteBot(db))
	admin.POST("/bots/:id/fire", adminFireBot(db, scheduler))
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

	// Widget assets: serve via a single wildcard handler so gin's radix tree has
	// no conflict between the dynamic chatsphere.js route and StaticFS's wildcard.
	widgetSub, _ := fs.Sub(web.WidgetFS, "widget")
	widgetFileServer := http.FileServer(http.FS(widgetSub))
	serveJS := serveWidgetJS(db)
	r.GET("/widget/*filepath", func(c *gin.Context) {
		if c.Param("filepath") == "/chatsphere.js" {
			serveJS(c)
			return
		}
		// Strip the "/widget" prefix so the file server resolves relative to widgetSub.
		c.Request.URL.Path = c.Param("filepath")
		widgetFileServer.ServeHTTP(c.Writer, c.Request)
	})
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
	AllowDM          bool   `json:"allow_dm"`
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
			AllowDM:          req.AllowDM,
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

		if err := db.Model(&ch).Updates(map[string]any{
			"name":               req.Name,
			"slug":               req.Slug,
			"description":        req.Description,
			"is_default":         req.IsDefault,
			"is_active":          req.IsActive,
			"max_msg_length":     req.MaxMsgLength,
			"rate_limit_seconds": req.RateLimitSeconds,
			"allow_anonymous":    req.AllowAnonymous,
			"allow_dm":           req.AllowDM,
			"welcome_message":    req.WelcomeMessage,
			"telegram_url":       req.TelegramURL,
			"accent_color":       req.AccentColor,
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

func adminDeleteMessage(db *gorm.DB, rdb *redis.Client, hubReg *ws.HubRegistry) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

		// Fetch the message first to know which channel it belongs to.
		var msg model.Message
		if err := db.First(&msg, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		db.Model(&model.Message{}).Where("id = ?", id).Update("is_deleted", true)

		// Purge the Redis history cache for this channel so new connections
		// don't receive the deleted message from the cache.
		cacheKey := fmt.Sprintf("chatsphere:channel:%d:messages", msg.ChannelID)
		rdb.Del(c.Request.Context(), cacheKey)

		// Broadcast a real-time deletion event to all connected clients in the channel.
		var ch model.Channel
		if err := db.First(&ch, msg.ChannelID).Error; err == nil {
			hub := hubReg.GetOrCreate(ch.Slug, ch.ID)
			hub.BroadcastJSON(ws.Event{Type: "message_deleted", MessageID: id})
		}

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

func adminCreateBlockedWord(db *gorm.DB, hubReg *ws.HubRegistry) gin.HandlerFunc {
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
		hubReg.InvalidateBlockedCacheAll()
		c.JSON(http.StatusCreated, gin.H{"blocked_word": word})
	}
}

func adminDeleteBlockedWord(db *gorm.DB, hubReg *ws.HubRegistry) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		db.Model(&model.BlockedWord{}).Where("id = ?", id).Update("is_active", false)
		hubReg.InvalidateBlockedCacheAll()
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// --- Allowed Domains ---

func adminListAllowedDomains(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var domains []model.AllowedDomain
		db.Order("created_at ASC").Find(&domains)
		c.JSON(http.StatusOK, gin.H{"domains": domains})
	}
}

func adminCreateAllowedDomain(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Domain string `json:"domain" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		domain := strings.ToLower(strings.TrimSpace(body.Domain))
		d := model.AllowedDomain{Domain: domain}
		if err := db.Create(&d).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"domain": d})
	}
}

func adminDeleteAllowedDomain(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		db.Delete(&model.AllowedDomain{}, id)
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

// --- Bots ---

func adminListBots(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var bots []model.BotConfig
		db.Order("created_at DESC").Find(&bots)
		c.JSON(http.StatusOK, gin.H{"bots": bots})
	}
}

type botRequest struct {
	Name           string  `json:"name" binding:"required"`
	AvatarColor    string  `json:"avatar_color"`
	ChannelID      *uint64 `json:"channel_id"`
	IntervalHours  int     `json:"interval_hours"`
	WebhookURL     string  `json:"webhook_url" binding:"required"`
	TelegramToken  string  `json:"telegram_token"`
	TelegramChatID string  `json:"telegram_chat_id"`
	IsActive       bool    `json:"is_active"`
}

func adminCreateBot(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req botRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if req.IntervalHours <= 0 {
			req.IntervalHours = 6
		}
		b := model.BotConfig{
			Name:           req.Name,
			AvatarColor:    req.AvatarColor,
			ChannelID:      req.ChannelID,
			IntervalHours:  req.IntervalHours,
			WebhookURL:     req.WebhookURL,
			TelegramToken:  req.TelegramToken,
			TelegramChatID: req.TelegramChatID,
			IsActive:       req.IsActive,
		}
		if err := db.Create(&b).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"bot": b})
	}
}

func adminUpdateBot(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		var b model.BotConfig
		if err := db.First(&b, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		var req botRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if req.IntervalHours <= 0 {
			req.IntervalHours = 6
		}
		db.Model(&b).Updates(map[string]interface{}{
			"name":             req.Name,
			"avatar_color":     req.AvatarColor,
			"channel_id":       req.ChannelID,
			"interval_hours":   req.IntervalHours,
			"webhook_url":      req.WebhookURL,
			"telegram_token":   req.TelegramToken,
			"telegram_chat_id": req.TelegramChatID,
			"is_active":        req.IsActive,
		})
		db.First(&b, id)
		c.JSON(http.StatusOK, gin.H{"bot": b})
	}
}

func adminDeleteBot(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		db.Delete(&model.BotConfig{}, id)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func adminFireBot(db *gorm.DB, scheduler *bot.Scheduler) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		var b model.BotConfig
		if err := db.First(&b, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if scheduler != nil {
			scheduler.FireBot(b)
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
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

		c.Header("Cache-Control", "no-cache, must-revalidate")
		c.Data(http.StatusOK, "application/javascript; charset=utf-8", buf.Bytes())
	}
}
