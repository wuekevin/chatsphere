package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/wuekevin/chatsphere/internal/bot"
	"github.com/wuekevin/chatsphere/internal/cache"
	"github.com/wuekevin/chatsphere/internal/config"
	"github.com/wuekevin/chatsphere/internal/database"
	"github.com/wuekevin/chatsphere/internal/handler"
	"github.com/wuekevin/chatsphere/internal/installer"
	"github.com/wuekevin/chatsphere/internal/repository"
	"github.com/wuekevin/chatsphere/internal/service"
	"github.com/wuekevin/chatsphere/internal/telegram"
	"github.com/wuekevin/chatsphere/internal/ws"
)

const configPath = "config.yaml"

func main() {
	printBanner()

	// First-run: no config.yaml → start the setup wizard
	if !config.IsInstalled(configPath) {
		log.Println("[chatsphere] No configuration found — starting setup wizard")
		log.Println("[chatsphere] Open http://localhost:8080 in your browser")
		fmt.Println()

		if err := installer.Run(configPath); err != nil {
			log.Fatalf("[chatsphere] Setup failed: %v", err)
		}

		log.Println("[chatsphere] Setup complete — starting server...")
		fmt.Println()
	} else {
		slog.Info("starting in normal mode")
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("[chatsphere] Cannot load config: %v", err)
	}

	if !cfg.Server.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	// Connect to dependencies
	db, err := database.Connect(cfg.Database)
	if err != nil {
		log.Fatalf("[chatsphere] Database error: %v", err)
	}
	log.Println("[chatsphere] Database connected")

	rdb, err := cache.Connect(cfg.Cache)
	if err != nil {
		log.Fatalf("[chatsphere] Redis error: %v", err)
	}
	log.Println("[chatsphere] Redis connected")

	// UserRegistry — tracks authenticated users across all hubs for DM delivery
	userReg := ws.NewUserRegistry()

	// Hub registry — creates per-channel Hub goroutines on demand
	registry := ws.NewHubRegistry(db, rdb, userReg)

	// Telegram bot service (optional — only active when bot_token is configured)
	tgSvc := telegram.NewService(cfg.Telegram.BotToken, cfg.Telegram.BotName, db, rdb, userReg)
	if tgSvc.Enabled() {
		registry.SetTelegramNotifier(tgSvc)
		log.Printf("[chatsphere] Telegram bot @%s enabled", cfg.Telegram.BotName)
	}

	// Build router
	r := gin.New()
	r.Use(gin.Recovery())
	if cfg.Server.Debug {
		r.Use(gin.Logger())
	}

	r.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.Server.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	chRepo := repository.NewChannelRepository(db)
	msgRepo := repository.NewMessageRepository(db)
	userRepo := repository.NewUserRepository(db)
	chSvc := service.NewChannelService(chRepo)
	msgSvc := service.NewMessageService(msgRepo)

	// Bot scheduler
	botScheduler := bot.NewScheduler(db, registry)
	botScheduler.Start()

	handler.RegisterAuth(r, db, cfg)
	handler.RegisterChat(r, chSvc, msgSvc, userRepo, registry, cfg)
	handler.RegisterAdmin(r, db, rdb, cfg, botScheduler)
	handler.RegisterWidget(r, db, cfg)
	handler.RegisterTelegram(r, tgSvc, cfg)

	// Redirect root to admin panel
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, cfg.Admin.Path)
	})

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{Addr: addr, Handler: r}

	go func() {
		log.Printf("[chatsphere] Listening on http://%s", addr)
		log.Printf("[chatsphere] Admin panel: http://%s%s", addr, cfg.Admin.Path)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[chatsphere] Server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[chatsphere] Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	log.Println("[chatsphere] Stopped")
}

func printBanner() {
	fmt.Println()
	fmt.Println("   ██████╗██╗  ██╗ █████╗ ████████╗███████╗██████╗ ██╗  ██╗███████╗██████╗ ███████╗")
	fmt.Println("  ██╔════╝██║  ██║██╔══██╗╚══██╔══╝██╔════╝██╔══██╗██║  ██║██╔════╝██╔══██╗██╔════╝")
	fmt.Println("  ██║     ███████║███████║   ██║   ███████╗██████╔╝███████║█████╗  ██████╔╝█████╗  ")
	fmt.Println("  ██║     ██╔══██║██╔══██║   ██║   ╚════██║██╔═══╝ ██╔══██║██╔══╝  ██╔══██╗██╔══╝  ")
	fmt.Println("  ╚██████╗██║  ██║██║  ██║   ██║   ███████║██║     ██║  ██║███████╗██║  ██║███████╗")
	fmt.Println("   ╚═════╝╚═╝  ╚═╝╚═╝  ╚═╝   ╚═╝   ╚══════╝╚═╝     ╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝╚══════╝")
	fmt.Println()
	fmt.Println("  Global Embeddable Chat Platform  —  v1.0.0")
	fmt.Println()
}
