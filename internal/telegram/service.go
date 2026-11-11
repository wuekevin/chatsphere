package telegram

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/wuekevin/chatsphere/internal/model"
	"github.com/wuekevin/chatsphere/internal/ws"
)

const (
	linkTTL  = 10 * time.Minute
	dmCtxTTL = 2 * time.Hour
)

// Update is a Telegram Bot API update object.
type Update struct {
	UpdateID int64    `json:"update_id"`
	Message  *TgMsg  `json:"message"`
}

type TgMsg struct {
	MessageID int64   `json:"message_id"`
	Chat      TgChat  `json:"chat"`
	From      *TgUser `json:"from"`
	Text      string  `json:"text"`
}

type TgChat struct {
	ID int64 `json:"id"`
}

type TgUser struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

// Service handles Telegram bot interactions.
type Service struct {
	botToken string
	botName  string
	db       *gorm.DB
	rdb      *redis.Client
	userReg  *ws.UserRegistry
}

func NewService(botToken, botName string, db *gorm.DB, rdb *redis.Client, userReg *ws.UserRegistry) *Service {
	return &Service{botToken: botToken, botName: botName, db: db, rdb: rdb, userReg: userReg}
}

func (s *Service) Enabled() bool {
	return s.botToken != ""
}

func (s *Service) BotName() string {
	return s.botName
}

// send calls the Telegram Bot API sendMessage endpoint.
func (s *Service) send(chatID int64, text string) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", s.botToken)
	body, _ := json.Marshal(map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "HTML",
	})
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return
	}
	resp.Body.Close()
}

// NotifyDM sends a DM push notification to a user via Telegram.
// Implements ws.TelegramNotifier.
func (s *Service) NotifyDM(ctx context.Context, toChatID int64, fromName, content string) {
	safeFrom := html.EscapeString(fromName)
	safeContent := html.EscapeString(content)
	text := fmt.Sprintf("💬 <b>%s</b>\n%s\n\n<i>Reply here to send a message back.</i>", safeFrom, safeContent)
	s.send(toChatID, text)
}

// SetDMContext records who last sent a DM to this Telegram chat so replies can be routed back.
// Implements ws.TelegramNotifier.
func (s *Service) SetDMContext(ctx context.Context, toChatID int64, fromUserID uint64) {
	key := fmt.Sprintf("chatsphere:tg:dm:ctx:%d", toChatID)
	s.rdb.Set(ctx, key, fromUserID, dmCtxTTL)
}

// GenerateLinkCode creates a short-lived one-time code the user can send to the bot.
func (s *Service) GenerateLinkCode(ctx context.Context, userID uint64) (string, error) {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	code := strings.ToUpper(hex.EncodeToString(b)) // 6 uppercase hex chars
	key := fmt.Sprintf("chatsphere:tg:link:%s", code)
	return code, s.rdb.SetEx(ctx, key, userID, linkTTL).Err()
}

// HandleUpdate processes an incoming Telegram update.
func (s *Service) HandleUpdate(ctx context.Context, update Update) {
	if update.Message == nil || update.Message.Text == "" {
		return
	}
	msg := update.Message
	chatID := msg.Chat.ID
	text := strings.TrimSpace(msg.Text)

	if strings.HasPrefix(text, "/") {
		s.handleCommand(ctx, chatID, text)
	} else {
		s.handleReply(ctx, chatID, text)
	}
}

func (s *Service) handleCommand(ctx context.Context, chatID int64, text string) {
	parts := strings.SplitN(text, " ", 2)
	cmd := strings.ToLower(parts[0])
	arg := ""
	if len(parts) == 2 {
		arg = strings.TrimSpace(parts[1])
	}

	switch cmd {
	case "/start":
		if arg != "" {
			s.linkAccount(ctx, chatID, arg)
		} else {
			s.send(chatID, "👋 Welcome to ChatSphere Bot!\n\nTo link your account, go to the chat widget → Account → Link Telegram and follow the instructions.")
		}
	case "/link":
		if arg != "" {
			s.linkAccount(ctx, chatID, arg)
		} else {
			s.send(chatID, "Usage: /link CODE")
		}
	case "/unlink":
		s.db.Model(&model.User{}).Where("telegram_chat_id = ?", chatID).UpdateColumn("telegram_chat_id", nil)
		s.send(chatID, "✅ Telegram unlinked from your ChatSphere account.")
	default:
		s.send(chatID, "Unknown command. Send /start for help.")
	}
}

func (s *Service) linkAccount(ctx context.Context, chatID int64, code string) {
	key := fmt.Sprintf("chatsphere:tg:link:%s", strings.ToUpper(code))
	val, err := s.rdb.Get(ctx, key).Result()
	if err != nil {
		s.send(chatID, "❌ Invalid or expired code. Please request a new one from the chat widget.")
		return
	}

	var userID uint64
	fmt.Sscanf(val, "%d", &userID)
	s.rdb.Del(ctx, key)

	s.db.Model(&model.User{}).Where("id = ?", userID).UpdateColumn("telegram_chat_id", chatID)
	s.send(chatID, "✅ Telegram linked! You'll now receive DM notifications here.\n\nReply to any notification to send a message back.")
}

// UnlinkUser removes the Telegram link from the user identified by userID.
func (s *Service) UnlinkUser(ctx context.Context, userID uint64) error {
	return s.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", userID).UpdateColumn("telegram_chat_id", nil).Error
}

func (s *Service) handleReply(ctx context.Context, chatID int64, text string) {
	// Find user by telegram_chat_id
	var fromUser model.User
	if err := s.db.Where("telegram_chat_id = ?", chatID).First(&fromUser).Error; err != nil {
		s.send(chatID, "⚠️ Your Telegram account isn't linked. Use /start to get started.")
		return
	}

	// Find DM context (last person who sent them a DM)
	ctxKey := fmt.Sprintf("chatsphere:tg:dm:ctx:%d", chatID)
	val, err := s.rdb.Get(ctx, ctxKey).Result()
	if err != nil {
		s.send(chatID, "⚠️ No active conversation. DM context expires 2 hours after the last message.")
		return
	}
	var toUserID uint64
	fmt.Sscanf(val, "%d", &toUserID)

	var toUser model.User
	if err := s.db.First(&toUser, toUserID).Error; err != nil {
		return
	}

	dm := &model.DirectMessage{
		FromUserID: fromUser.ID,
		ToUserID:   toUserID,
		Content:    text,
	}
	if err := s.db.Create(dm).Error; err != nil {
		return
	}
	dm.FromUsername = fromUser.Username
	dm.FromAvatarColor = fromUser.AvatarColor
	dm.ToUsername = toUser.Username
	dm.ToAvatarColor = toUser.AvatarColor

	// Deliver via WebSocket if recipient is online.
	if s.userReg != nil {
		payload, _ := json.Marshal(ws.Event{Type: "dm", DM: dm})
		s.userReg.Send(toUserID, payload)
	}

	// Extend DM context so they can keep replying.
	s.rdb.Expire(ctx, ctxKey, dmCtxTTL)
	s.send(chatID, fmt.Sprintf("✅ Sent to %s", toUser.Username))
}
