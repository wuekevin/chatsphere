package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/wuekevin/chatsphere/internal/config"
	"github.com/wuekevin/chatsphere/internal/middleware"
	"github.com/wuekevin/chatsphere/internal/model"
)

// RegisterAuth wires authentication routes.
func RegisterAuth(r *gin.Engine, db *gorm.DB, cfg *config.Config) {
	g := r.Group("/api/auth")
	g.POST("/register", registerHandler(db, cfg))
	g.POST("/login", loginHandler(db, cfg))
	g.GET("/me", middleware.Auth(cfg.Server.SecretKey), meHandler(db))
	g.PUT("/me", middleware.Auth(cfg.Server.SecretKey), updateMeHandler(db))

	r.GET("/api/users/:id", userProfileHandler(db))
}

type registerRequest struct {
	Username string `json:"username" binding:"required,min=3,max=30"`
	Password string `json:"password" binding:"required,min=6"`
	Email    string `json:"email"`
}

func registerHandler(db *gorm.DB, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req registerRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}

		// Best-effort country detection from CDN headers, then Accept-Language.
		country := detectCountryHeader(c)

		user := &model.User{
			Username:     req.Username,
			PasswordHash: string(hash),
			Email:        req.Email,
			AvatarColor:  randomColor(req.Username),
			Country:      country,
			Role:         model.RoleMember,
		}

		if err := db.Create(user).Error; err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": "username already taken"})
			return
		}

		token, err := issueToken(user, cfg.Server.SecretKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "token error"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"token": token, "user": user.PublicProfile()})
	}
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func loginHandler(db *gorm.DB, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req loginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var user model.User
		if err := db.Where("username = ?", req.Username).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}

		if user.IsBanned {
			c.JSON(http.StatusForbidden, gin.H{"error": "account banned"})
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}

		db.Model(&user).Update("last_seen_at", time.Now())

		token, err := issueToken(&user, cfg.Server.SecretKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "token error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"token": token, "user": user.PublicProfile()})
	}
}

func meHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := middleware.GetClaims(c)
		if claims == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		var user model.User
		if err := db.First(&user, claims.UserID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"user": user.PublicProfile()})
	}
}

func updateMeHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := middleware.GetClaims(c)
		if claims == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		var body struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		updates := map[string]any{}
		if body.Email != "" {
			updates["email"] = body.Email
		}
		if body.Password != "" {
			if len(body.Password) < 6 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "password must be at least 6 characters"})
				return
			}
			hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
				return
			}
			updates["password_hash"] = string(hash)
		}

		if len(updates) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "nothing to update"})
			return
		}

		db.Model(&model.User{}).Where("id = ?", claims.UserID).Updates(updates)

		var user model.User
		db.First(&user, claims.UserID)
		c.JSON(http.StatusOK, gin.H{"user": user.PublicProfile()})
	}
}

func userProfileHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var user model.User
		if err := db.First(&user, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"user": user.PublicProfile()})
	}
}

func issueToken(user *model.User, secret string) (string, error) {
	claims := &middleware.Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     string(user.Role),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// detectCountryHeader is a header-only country detection (no config) used at registration.
// It checks the same default headers as detectCountry in chat.go.
func detectCountryHeader(c *gin.Context) string {
	for _, h := range []string{"CF-IPCountry", "Smart-Country-Region", "EO-Client-Country-Region", "X-Country"} {
		if cc := extractCountryCode(c.GetHeader(h)); cc != "" {
			return cc
		}
	}
	al := c.GetHeader("Accept-Language")
	for _, seg := range strings.Split(al, ",") {
		seg = strings.TrimSpace(strings.SplitN(seg, ";", 2)[0])
		parts := strings.Split(seg, "-")
		if len(parts) >= 2 {
			cc := strings.ToUpper(parts[len(parts)-1])
			if len(cc) == 2 {
				return cc
			}
		}
	}
	return ""
}

// randomColor generates a deterministic avatar color from a username.
func randomColor(username string) string {
	colors := []string{
		"#4CAF50", "#2196F3", "#9C27B0", "#FF5722",
		"#009688", "#F44336", "#3F51B5", "#FF9800",
		"#00BCD4", "#795548", "#607D8B", "#E91E63",
	}
	var sum int
	for _, r := range username {
		sum += int(r)
	}
	return colors[sum%len(colors)]
}
