package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gorilla/websocket"
	"gorm.io/gorm"

	"github.com/wuekevin/chatsphere/internal/model"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // CORS is handled at the router level
	},
}

// Client represents a single WebSocket connection.
type Client struct {
	hub         *Hub
	conn        *websocket.Conn
	send        chan []byte
	channelSlug string
	channel     *model.Channel

	// Sender identity
	userID      *uint64
	displayName string
	country     string
	avatarColor string
	reputation  int
	role        string
	isAnonymous bool
	anonID      string

	// Rate limiting
	lastSentAt time.Time
}

// InboundMessage is what a client sends to the server.
type InboundMessage struct {
	Type       string `json:"type"`
	Content    string `json:"content,omitempty"`
	MessageID  uint64 `json:"message_id,omitempty"`
	Reason     string `json:"reason,omitempty"`
	Token      string `json:"token,omitempty"`
	ToUserID   uint64 `json:"to_user_id,omitempty"`
	WithUserID uint64 `json:"with_user_id,omitempty"`
}

// ServeWS upgrades the connection and launches the client goroutines.
// anonID is a precomputed 8-hex-char identity token (SHA-256(ip+salt)[:4]).
func ServeWS(hub *Hub, channel *model.Channel, w http.ResponseWriter, r *http.Request,
	userID *uint64, displayName, country, avatarColor, role string, reputation int, isAnon bool, anonID string) {

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[ws] upgrade error: %v", err)
		return
	}

	client := &Client{
		hub:         hub,
		conn:        conn,
		send:        make(chan []byte, 256),
		channelSlug: channel.Slug,
		channel:     channel,
		userID:      userID,
		displayName: displayName,
		country:     country,
		avatarColor: avatarColor,
		reputation:  reputation,
		role:        role,
		isAnonymous: isAnon,
		anonID:      anonID,
	}

	hub.register <- client

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, rawMsg, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
		c.handleInbound(rawMsg)
	}
}

func (c *Client) handleInbound(raw []byte) {
	var msg InboundMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return
	}

	switch msg.Type {
	case "send":
		c.handleSend(msg.Content)
	case "report":
		c.handleReport(msg.MessageID, msg.Reason)
	case "dm":
		c.handleDM(msg.ToUserID, msg.Content)
	case "dm_history":
		c.handleDMHistory(msg.WithUserID)
	case "ping":
		c.sendJSON(Event{Type: "pong"})
	}
}

func (c *Client) handleSend(content string) {
	content = strings.TrimSpace(content)
	if content == "" {
		return
	}
	// Enforce message length before escaping (escape expands some chars).
	maxLen := c.channel.MaxMsgLength
	if maxLen <= 0 {
		maxLen = 200
	}
	if utf8.RuneCountInString(content) > maxLen {
		c.sendError(fmt.Sprintf("Message exceeds %d characters.", maxLen))
		return
	}
	// Sanitize blocked words before HTML escaping (escaping expands some bytes).
	content = c.hub.SanitizeBlocked(content)

	content = html.EscapeString(content)

	// Rate limiting
	rateLimit := c.channel.RateLimitSeconds
	if rateLimit <= 0 {
		rateLimit = 3
	}
	if !c.lastSentAt.IsZero() && time.Since(c.lastSentAt) < time.Duration(rateLimit)*time.Second {
		remaining := time.Duration(rateLimit)*time.Second - time.Since(c.lastSentAt)
		c.sendError(fmt.Sprintf("Please wait %.0f seconds before sending again.", remaining.Seconds()))
		return
	}
	c.lastSentAt = time.Now()

	badge := ""
	if !c.isAnonymous {
		u := &model.User{Reputation: c.reputation, Role: model.Role(c.role)}
		badge = u.LevelTag()
	}

	record := &model.Message{
		ChannelID:   c.channel.ID,
		UserID:      c.userID,
		DisplayName: c.displayName,
		Content:     content,
		IPHash:      c.anonID,
		Country:     c.country,
		IsAnonymous: c.isAnonymous,
		AvatarColor: c.avatarColor,
		Reputation:  c.reputation,
		Badge:       badge,
	}

	if err := c.hub.PersistMessage(context.Background(), record, redisHistoryCap); err != nil {
		log.Printf("[ws] persist error: %v", err)
		return
	}

	c.hub.BroadcastJSON(Event{
		Type:    "message",
		Message: record,
	})

	// Notify any @mentioned users who are online in this channel.
	go c.hub.detectAndNotifyMentions(record.Content, record, c.userID)

	// Increment XP for registered users, with anti-grind rate limiting.
	if !c.isAnonymous && c.userID != nil {
		go c.tryGainXP()
	}
}

const (
	xpCooldown = 5 * time.Minute // minimum gap between XP gains
	xpDailyMax = 20              // max XP per user per UTC day
)

// tryGainXP awards +1 XP if the user hasn't hit the cooldown or daily cap.
// Two Redis keys guard against grinding:
//   - xp_cd:{uid}   — set with 5-min TTL on each gain; skip if present
//   - xp_day:{uid}:{date} — counter capped at xpDailyMax; TTL 25h
func (c *Client) tryGainXP() {
	ctx := context.Background()
	uid := *c.userID
	rdb := c.hub.redis

	cdKey := fmt.Sprintf("chatsphere:xp_cd:%d", uid)
	dayKey := fmt.Sprintf("chatsphere:xp_day:%d:%s", uid, time.Now().UTC().Format("2006-01-02"))

	// Cooldown check: SET NX with TTL — returns true only if key didn't exist
	ok, err := rdb.SetNX(ctx, cdKey, 1, xpCooldown).Result()
	if err != nil || !ok {
		return // Redis error or cooldown active
	}

	// Daily cap check
	daily, err := rdb.Incr(ctx, dayKey).Result()
	if err != nil {
		return
	}
	if daily == 1 {
		rdb.Expire(ctx, dayKey, 25*time.Hour) // expire well after day rolls over
	}
	if daily > int64(xpDailyMax) {
		return // cap reached today
	}

	c.reputation++
	c.hub.db.Model(&model.User{}).Where("id = ?", uid).
		UpdateColumn("reputation", gorm.Expr("reputation + 1"))
}

func (c *Client) handleDM(toUserID uint64, content string) {
	if c.isAnonymous || c.userID == nil {
		c.sendError("Private messages require a registered account.")
		return
	}
	if toUserID == 0 || toUserID == *c.userID {
		c.sendError("Invalid recipient.")
		return
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return
	}
	content = html.EscapeString(content)
	if utf8.RuneCountInString(content) > 500 {
		c.sendError("Private message exceeds 500 characters.")
		return
	}

	// Look up recipient to get display info
	var toUser model.User
	if err := c.hub.db.First(&toUser, toUserID).Error; err != nil {
		c.sendError("User not found.")
		return
	}

	dm := &model.DirectMessage{
		FromUserID: *c.userID,
		ToUserID:   toUserID,
		Content:    content,
	}
	if err := c.hub.db.Create(dm).Error; err != nil {
		c.sendError("Failed to send message.")
		return
	}

	dm.FromUsername = c.displayName
	dm.FromAvatarColor = c.avatarColor
	dm.ToUsername = toUser.Username
	dm.ToAvatarColor = toUser.AvatarColor

	payload, err := json.Marshal(Event{Type: "dm", DM: dm})
	if err != nil {
		return
	}

	// Deliver to recipient (all their active connections).
	var recipientOnline bool
	if c.hub.userReg != nil {
		recipientOnline = c.hub.userReg.Send(toUserID, payload)
	}
	// Echo back to sender
	select {
	case c.send <- payload:
	default:
	}

	// Telegram push notification when recipient is offline.
	if !recipientOnline && c.hub.tgNotifier != nil && c.hub.tgNotifier.Enabled() && toUser.TelegramChatID != nil {
		fromName := c.displayName
		msgContent := dm.Content
		chatID := *toUser.TelegramChatID
		senderID := *c.userID
		go func() {
			ctx := context.Background()
			c.hub.tgNotifier.NotifyDM(ctx, chatID, fromName, msgContent)
			c.hub.tgNotifier.SetDMContext(ctx, chatID, senderID)
		}()
	}
}

func (c *Client) handleDMHistory(withUserID uint64) {
	if c.isAnonymous || c.userID == nil {
		c.sendError("Private messages require a registered account.")
		return
	}
	if withUserID == 0 {
		return
	}

	var dms []model.DirectMessage
	c.hub.db.Where(
		"(from_user_id = ? AND to_user_id = ?) OR (from_user_id = ? AND to_user_id = ?)",
		*c.userID, withUserID, withUserID, *c.userID,
	).Order("created_at ASC").Limit(50).Find(&dms)

	// Look up the other user once
	var other model.User
	c.hub.db.First(&other, withUserID)

	for i := range dms {
		if dms[i].FromUserID == *c.userID {
			dms[i].FromUsername = c.displayName
			dms[i].FromAvatarColor = c.avatarColor
			dms[i].ToUsername = other.Username
			dms[i].ToAvatarColor = other.AvatarColor
		} else {
			dms[i].FromUsername = other.Username
			dms[i].FromAvatarColor = other.AvatarColor
			dms[i].ToUsername = c.displayName
			dms[i].ToAvatarColor = c.avatarColor
		}
	}

	wid := withUserID
	c.sendJSON(Event{Type: "dm_history", DMs: dms, WithUserID: &wid})
}

func (c *Client) handleReport(messageID uint64, reason string) {
	report := &model.Report{
		MessageID: messageID,
		UserID:    c.userID,
		Reason:    reason,
		Status:    "pending",
	}
	c.hub.db.Create(report)
	c.sendJSON(Event{Type: "system", Content: "Report submitted. Thank you."})
}

func (c *Client) sendJSON(v any) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	select {
	case c.send <- data:
	default:
	}
}

func (c *Client) sendError(msg string) {
	c.sendJSON(Event{Type: "error", Error: msg})
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
