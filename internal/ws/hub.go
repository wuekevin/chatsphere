package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/wuekevin/chatsphere/internal/model"
)

const (
	redisChannelKey = "chatsphere:channel:%s:messages"
	redisOnlineKey  = "chatsphere:channel:%s:online"
)

// Hub maintains all active WebSocket connections, organized by channel slug.
type Hub struct {
	mu       sync.RWMutex
	channels map[string]map[*Client]bool

	broadcast  chan *Envelope
	register   chan *Client
	unregister chan *Client

	db    *gorm.DB
	redis *redis.Client
}

// Envelope wraps an outbound message with its target channel.
type Envelope struct {
	ChannelSlug string
	Payload     []byte
}

func NewHub(db *gorm.DB, rdb *redis.Client) *Hub {
	return &Hub{
		channels:   make(map[string]map[*Client]bool),
		broadcast:  make(chan *Envelope, 512),
		register:   make(chan *Client, 64),
		unregister: make(chan *Client, 64),
		db:         db,
		redis:      rdb,
	}
}

// Run is the hub's main event loop — call it in a goroutine.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.addClient(client)

		case client := <-h.unregister:
			h.removeClient(client)

		case env := <-h.broadcast:
			h.broadcastToChannel(env)
		}
	}
}

func (h *Hub) addClient(c *Client) {
	h.mu.Lock()
	if h.channels[c.channelSlug] == nil {
		h.channels[c.channelSlug] = make(map[*Client]bool)
	}
	h.channels[c.channelSlug][c] = true
	h.mu.Unlock()

	count := h.onlineCount(c.channelSlug)
	h.BroadcastJSON(c.channelSlug, Event{
		Type:  "online",
		Count: &count,
	})

	// Send chat history from Redis
	go h.sendHistory(c)
}

func (h *Hub) removeClient(c *Client) {
	h.mu.Lock()
	if _, ok := h.channels[c.channelSlug][c]; ok {
		delete(h.channels[c.channelSlug], c)
		close(c.send)
	}
	h.mu.Unlock()

	count := h.onlineCount(c.channelSlug)
	h.BroadcastJSON(c.channelSlug, Event{
		Type:  "online",
		Count: &count,
	})
}

func (h *Hub) broadcastToChannel(env *Envelope) {
	h.mu.RLock()
	clients := h.channels[env.ChannelSlug]
	h.mu.RUnlock()

	for c := range clients {
		select {
		case c.send <- env.Payload:
		default:
			h.unregister <- c
		}
	}
}

// BroadcastJSON serialises v and broadcasts it to all clients in the channel.
func (h *Hub) BroadcastJSON(channelSlug string, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		log.Printf("[hub] marshal error: %v", err)
		return
	}
	h.broadcast <- &Envelope{ChannelSlug: channelSlug, Payload: data}
}

// OnlineCount returns the number of connected clients in a channel.
func (h *Hub) onlineCount(slug string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.channels[slug])
}

// PersistMessage stores msg in Redis (capped list) and MySQL.
func (h *Hub) PersistMessage(ctx context.Context, msg *model.Message, maxHistory int) error {
	// Save to MySQL
	if err := h.db.WithContext(ctx).Create(msg).Error; err != nil {
		return fmt.Errorf("db create: %w", err)
	}

	// Cache in Redis as a capped list
	key := fmt.Sprintf(redisChannelKey, msg.ChannelID)
	data, _ := json.Marshal(msg)
	pipe := h.redis.Pipeline()
	pipe.RPush(ctx, key, data)
	pipe.LTrim(ctx, key, int64(-maxHistory), -1)
	pipe.Expire(ctx, key, 24*time.Hour)
	_, err := pipe.Exec(ctx)
	return err
}

// sendHistory pushes recent messages to a newly connected client.
func (h *Hub) sendHistory(c *Client) {
	ctx := context.Background()
	key := fmt.Sprintf(redisChannelKey, c.channelSlug)

	raw, err := h.redis.LRange(ctx, key, 0, -1).Result()
	if err != nil || len(raw) == 0 {
		return
	}

	var messages []json.RawMessage
	for _, r := range raw {
		messages = append(messages, json.RawMessage(r))
	}

	payload, _ := json.Marshal(Event{
		Type:     "history",
		Messages: messages,
	})
	select {
	case c.send <- payload:
	default:
	}
}

// Event is the JSON envelope sent over WebSocket.
type Event struct {
	Type     string            `json:"type"`
	Count    *int              `json:"count,omitempty"`
	Message  *model.Message    `json:"message,omitempty"`
	Messages []json.RawMessage `json:"messages,omitempty"`
	Error    string            `json:"error,omitempty"`
	ID       *uint64           `json:"id,omitempty"`
	Content  string            `json:"content,omitempty"`
}
