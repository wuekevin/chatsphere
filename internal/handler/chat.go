package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/wuekevin/chatsphere/internal/config"
	"github.com/wuekevin/chatsphere/internal/middleware"
	"github.com/wuekevin/chatsphere/internal/model"
	"github.com/wuekevin/chatsphere/internal/ws"
)

// RegisterChat wires chat and WebSocket routes.
func RegisterChat(r *gin.Engine, db *gorm.DB, rdb *redis.Client, hub *ws.Hub, cfg *config.Config) {
	r.GET("/ws/:channel", middleware.OptionalAuth(cfg.Server.SecretKey), wsHandler(db, hub, cfg))

	api := r.Group("/api")
	api.GET("/channels", listChannelsHandler(db))
	api.GET("/channels/:slug/messages", getMessagesHandler(db))
}

func wsHandler(db *gorm.DB, hub *ws.Hub, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := c.Param("channel")

		var channel model.Channel
		if err := db.Where("slug = ? AND is_active = true", slug).First(&channel).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
			return
		}

		claims := middleware.GetClaims(c)

		var (
			userID      *uint64
			displayName string
			country     string
			avatarColor string
			reputation  int
			isAnon      = true
		)

		if claims != nil {
			var user model.User
			if err := db.First(&user, claims.UserID).Error; err == nil && !user.IsBanned {
				uid := user.ID
				userID = &uid
				displayName = user.Username
				country = user.Country
				avatarColor = user.AvatarColor
				reputation = user.Reputation
				isAnon = false
			}
		}

		if isAnon {
			if !channel.AllowAnonymous {
				c.JSON(http.StatusForbidden, gin.H{"error": "this channel requires a registered account"})
				return
			}
			// Generate a temporary display name for anonymous users
			displayName = anonName(c.ClientIP())
			avatarColor = randomColor(displayName)
		}

		ws.ServeWS(hub, &channel, c.Writer, c.Request,
			userID, displayName, country, avatarColor, reputation, isAnon)
	}
}

func listChannelsHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var channels []model.Channel
		db.Where("is_active = true").Order("is_default DESC, name ASC").Find(&channels)
		c.JSON(http.StatusOK, gin.H{"channels": channels})
	}
}

func getMessagesHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := c.Param("slug")

		var channel model.Channel
		if err := db.Where("slug = ?", slug).First(&channel).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
			return
		}

		var messages []model.Message
		db.Where("channel_id = ? AND is_deleted = false", channel.ID).
			Order("created_at DESC").
			Limit(100).
			Find(&messages)

		c.JSON(http.StatusOK, gin.H{"messages": messages})
	}
}

// anonName creates a short, stable display name for anonymous users from their IP.
func anonName(ip string) string {
	adjectives := []string{"Quick", "Calm", "Bold", "Wise", "Bright", "Swift", "Cool", "Kind"}
	nouns := []string{"Fox", "Bear", "Wolf", "Hawk", "Lynx", "Deer", "Crow", "Hare"}
	var sum int
	for _, r := range ip {
		sum += int(r)
	}
	return adjectives[sum%len(adjectives)] + nouns[(sum/8)%len(nouns)]
}
