package handler

import (
	"net/http"
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

		user := &model.User{
			Username:     req.Username,
			PasswordHash: string(hash),
			Email:        req.Email,
			AvatarColor:  randomColor(req.Username),
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
