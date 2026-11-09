package handler

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/wuekevin/chatsphere/internal/config"
	"github.com/wuekevin/chatsphere/internal/middleware"
	"github.com/wuekevin/chatsphere/internal/repository"
	"github.com/wuekevin/chatsphere/internal/service"
	"github.com/wuekevin/chatsphere/internal/ws"
)

// RegisterChat wires chat and WebSocket routes.
func RegisterChat(
	r *gin.Engine,
	chSvc *service.ChannelService,
	msgSvc *service.MessageService,
	userRepo repository.UserRepository,
	registry *ws.HubRegistry,
	cfg *config.Config,
) {
	r.GET("/ws/:channel", middleware.OptionalAuth(cfg.Server.SecretKey), wsHandler(chSvc, userRepo, registry, cfg))

	api := r.Group("/api")
	api.GET("/channels", listChannelsHandler(chSvc))
	api.GET("/channels/:slug/messages", getMessagesHandler(chSvc, msgSvc))
}

func wsHandler(chSvc *service.ChannelService, userRepo repository.UserRepository, registry *ws.HubRegistry, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := c.Param("channel")

		channel, err := chSvc.GetBySlug(slug)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			}
			return
		}
		hub := registry.GetOrCreate(channel.Slug, channel.ID)

		// anonID: stable 8-char hex derived from client IP + server salt.
		ip := c.GetHeader("X-Real-IP")
		if ip == "" {
			ip = c.Request.RemoteAddr
		}
		sum := sha256.Sum256([]byte(ip + cfg.Server.SecretKey))
		anonID := fmt.Sprintf("%x", sum[:4])

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
			user, err := userRepo.GetByID(claims.UserID)
			if err == nil && !user.IsBanned {
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
			displayName = anonName(anonID)
			avatarColor = randomColor(displayName)
		}

		ws.ServeWS(hub, channel, c.Writer, c.Request,
			userID, displayName, country, avatarColor, reputation, isAnon, anonID)
	}
}

func listChannelsHandler(chSvc *service.ChannelService) gin.HandlerFunc {
	return func(c *gin.Context) {
		channels, err := chSvc.ListActive()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list channels"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"channels": channels})
	}
}

func getMessagesHandler(chSvc *service.ChannelService, msgSvc *service.MessageService) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := c.Param("slug")

		channel, err := chSvc.GetBySlug(slug)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch channel"})
			}
			return
		}

		messages, err := msgSvc.ListByChannel(channel.ID, 100)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch messages"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"messages": messages})
	}
}

// anonName returns a stable display name from an 8-hex-char anonID.
// 20 adjectives × 20 nouns = 400 possible names.
func anonName(anonID string) string {
	adjectives := []string{
		"Quick", "Calm", "Bold", "Wise", "Bright", "Swift", "Cool", "Kind",
		"Brave", "Sharp", "Wild", "Keen", "Fair", "Free", "Pure", "Warm",
		"Soft", "Vast", "Dark", "Stark",
	}
	nouns := []string{
		"Fox", "Bear", "Wolf", "Hawk", "Lynx", "Deer", "Crow", "Hare",
		"Owl", "Pike", "Kite", "Mink", "Puma", "Ibis", "Wren", "Vole",
		"Newt", "Moth", "Frog", "Swan",
	}
	n, _ := strconv.ParseUint(anonID, 16, 64)
	return adjectives[n%uint64(len(adjectives))] + nouns[(n/uint64(len(adjectives)))%uint64(len(nouns))]
}
