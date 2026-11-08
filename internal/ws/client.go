package ws

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gorilla/websocket"

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
	isAnonymous bool
	ipHash      string

	// Rate limiting
	lastSentAt time.Time
}

// InboundMessage is what a client sends to the server.
type InboundMessage struct {
	Type      string `json:"type"`
	Content   string `json:"content,omitempty"`
	MessageID uint64 `json:"message_id,omitempty"`
	Reason    string `json:"reason,omitempty"`
	Token     string `json:"token,omitempty"`
}

// ServeWS upgrades the connection and launches the client goroutines.
func ServeWS(hub *Hub, channel *model.Channel, w http.ResponseWriter, r *http.Request,
	userID *uint64, displayName, country, avatarColor string, reputation int, isAnon bool) {

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[ws] upgrade error: %v", err)
		return
	}

	ip := r.Header.Get("X-Real-IP")
	if ip == "" {
		ip = r.RemoteAddr
	}
	sum := sha256.Sum256([]byte(ip))
	ipHash := fmt.Sprintf("%x", sum[:16])

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
		isAnonymous: isAnon,
		ipHash:      ipHash,
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
	case "ping":
		c.sendJSON(Event{Type: "pong"})
	}
}

func (c *Client) handleSend(content string) {
	content = strings.TrimSpace(content)
	if content == "" {
		return
	}

	// Enforce message length
	maxLen := c.channel.MaxMsgLength
	if maxLen <= 0 {
		maxLen = 200
	}
	if utf8.RuneCountInString(content) > maxLen {
		c.sendError(fmt.Sprintf("Message exceeds %d characters.", maxLen))
		return
	}

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

	record := &model.Message{
		ChannelID:   c.channel.ID,
		UserID:      c.userID,
		DisplayName: c.displayName,
		Content:     content,
		IPHash:      c.ipHash,
		Country:     c.country,
		IsAnonymous: c.isAnonymous,
		AvatarColor: c.avatarColor,
		Reputation:  c.reputation,
	}

	if err := c.hub.PersistMessage(context.Background(), record, c.channel.MaxMsgLength); err != nil {
		log.Printf("[ws] persist error: %v", err)
		return
	}

	c.hub.BroadcastJSON(c.channelSlug, Event{
		Type:    "message",
		Message: record,
	})
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
