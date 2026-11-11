package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/wuekevin/chatsphere/internal/config"
	"github.com/wuekevin/chatsphere/internal/middleware"
	"github.com/wuekevin/chatsphere/internal/telegram"
)

// RegisterTelegram wires Telegram bot webhook and account-link routes.
func RegisterTelegram(r *gin.Engine, svc *telegram.Service, cfg *config.Config) {
	if !svc.Enabled() {
		return
	}

	// Webhook: Telegram calls this when it receives a message.
	r.POST("/api/telegram/webhook", telegramWebhookHandler(svc, cfg))

	// Link: authenticated user requests a one-time link code.
	r.GET("/api/me/telegram-link", middleware.Auth(cfg.Server.SecretKey), telegramLinkHandler(svc))

	// Unlink: authenticated user removes their Telegram link.
	r.DELETE("/api/me/telegram-link", middleware.Auth(cfg.Server.SecretKey), telegramUnlinkHandler(svc))
}

func telegramWebhookHandler(svc *telegram.Service, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Optionally verify the webhook secret token Telegram sends in X-Telegram-Bot-Api-Secret-Token.
		// For now we accept all POSTs to this endpoint (keep it URL-secret).
		var update telegram.Update
		if err := c.ShouldBindJSON(&update); err != nil {
			c.Status(http.StatusBadRequest)
			return
		}
		go svc.HandleUpdate(c.Request.Context(), update)
		c.Status(http.StatusOK)
	}
}

func telegramLinkHandler(svc *telegram.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := middleware.GetClaims(c)
		if claims == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		code, err := svc.GenerateLinkCode(c.Request.Context(), claims.UserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate code"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"code":     code,
			"bot_name": svc.BotName(),
			"command":  "/link " + code,
		})
	}
}

func telegramUnlinkHandler(svc *telegram.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := middleware.GetClaims(c)
		if claims == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		_ = svc.UnlinkUser(c.Request.Context(), claims.UserID)
		c.Status(http.StatusNoContent)
	}
}
