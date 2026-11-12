package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/wuekevin/chatsphere/internal/model"
)

// TelegramNotifier is an optional integration for DM push notifications via Telegram.
type TelegramNotifier interface {
	Enabled() bool
	NotifyDM(ctx context.Context, toChatID int64, fromName, content string)
	SetDMContext(ctx context.Context, toChatID int64, fromUserID uint64)
}

type blockedPattern struct {
	raw string
	re  *regexp.Regexp // non-nil when IsRegex
}

const (
	redisChannelKey = "chatsphere:channel:%d:messages"
	redisOnlineKey  = "chatsphere:channel:%d:online"
	historyLimit    = 20  // messages sent to a client on join
	redisHistoryCap = 500 // max messages retained in Redis per channel
)

// Hub manages WebSocket connections for a single channel.
type Hub struct {
	mu      sync.RWMutex
	clients map[*Client]bool

	channelSlug string
	channelID   uint64

	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client

	db      *gorm.DB
	redis   *redis.Client
	userReg *UserRegistry

	// Blocked-word cache (TTL-refreshed, avoids per-message DB queries).
	blockedMu       sync.RWMutex
	blockedPatterns []blockedPattern
	blockedLoadedAt time.Time

	tgNotifier TelegramNotifier
}

func newHub(slug string, channelID uint64, db *gorm.DB, rdb *redis.Client, userReg *UserRegistry, tgn TelegramNotifier) *Hub {
	return &Hub{
		clients:     make(map[*Client]bool),
		channelSlug: slug,
		channelID:   channelID,
		broadcast:   make(chan []byte, 512),
		register:    make(chan *Client, 64),
		unregister:  make(chan *Client, 64),
		db:          db,
		redis:       rdb,
		userReg:     userReg,
		tgNotifier:  tgn,
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
		case payload := <-h.broadcast:
			h.broadcastToAll(payload)
		}
	}
}

func (h *Hub) addClient(c *Client) {
	h.mu.Lock()
	h.clients[c] = true
	h.mu.Unlock()

	if c.userID != nil && h.userReg != nil {
		h.userReg.Register(*c.userID, c)
	}

	go h.updateOnlineCount(c, true)
	// Send identity (+ channel accent color) before history so the widget labels
	// own messages correctly and applies the right theme color immediately.
	go func() {
		defer func() { recover() }() // client may disconnect while this goroutine runs
		var accent string
		if c.channel != nil {
			accent = c.channel.AccentColor
		}
		evt := Event{
			Type:        "joined",
			Content:     c.displayName,
			AccentColor: accent,
			AvatarColor: c.avatarColor,
			UserID:      c.userID,
		}
		if payload, err := json.Marshal(evt); err == nil {
			select {
			case c.send <- payload:
			default:
			}
		}
		h.sendHistory(c)
	}()
}

func (h *Hub) removeClient(c *Client) {
	h.mu.Lock()
	if _, ok := h.clients[c]; ok {
		delete(h.clients, c)
		close(c.send)
	}
	h.mu.Unlock()

	if c.userID != nil && h.userReg != nil {
		h.userReg.Unregister(*c.userID, c)
	}

	go h.updateOnlineCount(c, false)
}

func (h *Hub) broadcastToAll(payload []byte) {
	h.mu.RLock()
	clients := make([]*Client, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	h.mu.RUnlock()

	for _, c := range clients {
		select {
		case c.send <- payload:
		default:
			// Client too slow: remove synchronously to avoid blocking Run() on the unregister channel.
			h.mu.Lock()
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
			}
			h.mu.Unlock()
			go h.updateOnlineCount(c, false)
		}
	}
}

// BroadcastJSON serialises v and broadcasts it to all clients in this channel.
func (h *Hub) BroadcastJSON(v any) {
	data, err := json.Marshal(v)
	if err != nil {
		log.Printf("[hub] marshal error: %v", err)
		return
	}
	h.broadcast <- data
}

// updateOnlineCount updates the Redis online-member set and broadcasts the new count.
// joining=true: SADD; joining=false: SREM (only when no duplicate client identity remains).
func (h *Hub) updateOnlineCount(c *Client, joining bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	key := fmt.Sprintf(redisOnlineKey, h.channelID)
	member := clientMember(c)

	if joining {
		h.redis.SAdd(ctx, key, member)
		h.redis.Expire(ctx, key, 24*time.Hour)
	} else {
		// Only remove the set member when no other connected client shares the same identity.
		h.mu.RLock()
		hasDuplicate := false
		for other := range h.clients {
			if clientMember(other) == member {
				hasDuplicate = true
				break
			}
		}
		h.mu.RUnlock()
		if !hasDuplicate {
			h.redis.SRem(ctx, key, member)
		}
	}

	count, err := h.redis.SCard(ctx, key).Result()
	if err != nil {
		// Redis unavailable: fall back to in-memory connection count.
		h.mu.RLock()
		count = int64(len(h.clients))
		h.mu.RUnlock()
	}
	n := int(count)
	select {
	case h.broadcast <- func() []byte {
		data, _ := json.Marshal(Event{Type: "online_count", Count: &n})
		return data
	}():
	default:
	}
}

// clientMember returns the Redis set member string for a client.
// Anonymous clients from the same IP share an anonID and count as one visitor.
// Authenticated users across multiple tabs count as one user.
func clientMember(c *Client) string {
	if c.userID != nil {
		return fmt.Sprintf("u%d", *c.userID)
	}
	return "a" + c.anonID
}

// PersistMessage stores msg in MySQL, then caches it in Redis.
// MySQL failure returns an error (caller aborts). Redis failure is logged but non-fatal.
func (h *Hub) PersistMessage(ctx context.Context, msg *model.Message, maxHistory int) error {
	if err := h.db.WithContext(ctx).Create(msg).Error; err != nil {
		return fmt.Errorf("db create: %w", err)
	}

	key := fmt.Sprintf(redisChannelKey, h.channelID)
	data, _ := json.Marshal(msg)
	pipe := h.redis.Pipeline()
	pipe.RPush(ctx, key, data)
	pipe.LTrim(ctx, key, int64(-maxHistory), -1)
	pipe.Expire(ctx, key, 24*time.Hour)
	if _, err := pipe.Exec(ctx); err != nil {
		log.Printf("[hub] redis cache error channel=%d: %v", h.channelID, err)
	}
	return nil
}

func (h *Hub) sendHistory(c *Client) {
	ctx := context.Background()
	key := fmt.Sprintf(redisChannelKey, h.channelID)

	raw, err := h.redis.LRange(ctx, key, -historyLimit, -1).Result()
	if err != nil || len(raw) == 0 {
		// Redis empty (e.g. after a message deletion cleared the cache) —
		// fall back to MySQL, skip deleted messages, then repopulate Redis.
		var dbMsgs []model.Message
		h.db.WithContext(ctx).
			Where("channel_id = ? AND is_deleted = false", h.channelID).
			Order("created_at DESC").
			Limit(historyLimit).
			Find(&dbMsgs)
		if len(dbMsgs) == 0 {
			return
		}
		// Reverse to chronological order.
		for i, j := 0, len(dbMsgs)-1; i < j; i, j = i+1, j-1 {
			dbMsgs[i], dbMsgs[j] = dbMsgs[j], dbMsgs[i]
		}
		pipe := h.redis.Pipeline()
		var messages []json.RawMessage
		for _, m := range dbMsgs {
			data, _ := json.Marshal(m)
			messages = append(messages, json.RawMessage(data))
			pipe.RPush(ctx, key, data)
		}
		pipe.LTrim(ctx, key, int64(-redisHistoryCap), -1)
		pipe.Expire(ctx, key, 24*time.Hour)
		pipe.Exec(ctx)

		payload, _ := json.Marshal(Event{Type: "history", Messages: messages})
		select {
		case c.send <- payload:
		default:
		}
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
	Type        string                 `json:"type"`
	Count       *int                   `json:"count,omitempty"`
	Message     *model.Message         `json:"message,omitempty"`
	Messages    []json.RawMessage      `json:"messages,omitempty"`
	Error       string                 `json:"error,omitempty"`
	ID          *uint64                `json:"id,omitempty"`
	Content     string                 `json:"content,omitempty"`
	AccentColor string                 `json:"accent_color,omitempty"`
	AvatarColor string                 `json:"avatar_color,omitempty"`
	UserID      *uint64                `json:"user_id,omitempty"`
	DM          *model.DirectMessage   `json:"dm,omitempty"`
	DMs         []model.DirectMessage  `json:"dms,omitempty"`
	WithUserID  *uint64                `json:"with_user_id,omitempty"`
	BotCards    []model.BotCard        `json:"bot_cards,omitempty"`
	MessageID   uint64                 `json:"message_id,omitempty"`
}

var mentionRe = regexp.MustCompile(`@(\w+)`)

// detectAndNotifyMentions looks for @username in a message and delivers a
// "mention" event to any matching users who are currently connected to this hub.
func (h *Hub) detectAndNotifyMentions(content string, msg *model.Message, senderID *uint64) {
	matches := mentionRe.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return
	}

	const maxMentions = 10
	seen := map[string]bool{}
	var usernames []string
	for _, m := range matches {
		if len(usernames) >= maxMentions {
			break
		}
		lower := strings.ToLower(m[1])
		if !seen[lower] {
			seen[lower] = true
			usernames = append(usernames, m[1])
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var users []model.User
	if err := h.db.WithContext(ctx).Where("username IN ?", usernames).Find(&users).Error; err != nil || len(users) == 0 {
		return
	}

	payload, err := json.Marshal(Event{Type: "mention", Message: msg})
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, u := range users {
		if senderID != nil && *senderID == u.ID {
			continue // don't notify the sender of their own mention
		}
		uid := u.ID
		for c := range h.clients {
			if c.userID != nil && *c.userID == uid {
				select {
				case c.send <- payload:
				default:
				}
			}
		}
	}
}

// InvalidateBlockedCache resets the blocked-word cache TTL so the next
// SanitizeBlocked call reloads from the database immediately.
func (h *Hub) InvalidateBlockedCache() {
	h.blockedMu.Lock()
	h.blockedLoadedAt = time.Time{}
	h.blockedMu.Unlock()
}

// loadBlockedPatterns queries the DB and rebuilds the in-memory cache.
// Every pattern (regex or plain) is compiled to a *regexp.Regexp for uniform replace.
func (h *Hub) loadBlockedPatterns() {
	var words []model.BlockedWord
	h.db.Where("is_active = true AND (channel_id IS NULL OR channel_id = ?)", h.channelID).Find(&words)

	patterns := make([]blockedPattern, 0, len(words))
	for _, w := range words {
		p := blockedPattern{raw: w.Pattern}
		if w.IsRegex {
			re, err := regexp.Compile(w.Pattern)
			if err != nil {
				continue
			}
			p.re = re
		} else {
			re, err := regexp.Compile(`(?i)` + regexp.QuoteMeta(w.Pattern))
			if err != nil {
				continue
			}
			p.re = re
		}
		patterns = append(patterns, p)
	}

	h.blockedMu.Lock()
	h.blockedPatterns = patterns
	h.blockedLoadedAt = time.Now()
	h.blockedMu.Unlock()
}

// CheckURLAllowed returns true if content contains no URLs, or all URLs are from
// admin-allowed domains. If no allowed domains are configured, all URLs are permitted.
func (h *Hub) CheckURLAllowed(content string) bool {
	urlRe := regexp.MustCompile(`https?://([^/\s"'<>]+)`)
	matches := urlRe.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return true
	}

	var domains []string
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := h.db.WithContext(ctx).Model(&struct {
		Domain string
	}{}).Table("allowed_domains").Pluck("domain", &domains).Error; err != nil || len(domains) == 0 {
		return true // no rules configured → allow everything
	}

	domainSet := make(map[string]bool, len(domains))
	for _, d := range domains {
		domainSet[strings.ToLower(d)] = true
	}

	for _, m := range matches {
		host := strings.ToLower(m[1])
		// strip port
		if idx := strings.Index(host, ":"); idx != -1 {
			host = host[:idx]
		}
		allowed := false
		for d := range domainSet {
			if host == d || strings.HasSuffix(host, "."+d) {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}
	return true
}

// SanitizeBlocked replaces matched blocked words/patterns with *** in content.
// The cache refreshes automatically after 5 minutes.
func (h *Hub) SanitizeBlocked(content string) string {
	h.blockedMu.RLock()
	stale := time.Since(h.blockedLoadedAt) > 5*time.Minute
	h.blockedMu.RUnlock()

	if stale {
		h.loadBlockedPatterns()
	}

	h.blockedMu.RLock()
	defer h.blockedMu.RUnlock()
	for _, p := range h.blockedPatterns {
		if p.re == nil {
			continue
		}
		content = p.re.ReplaceAllStringFunc(content, func(m string) string {
			return strings.Repeat("*", utf8.RuneCountInString(m))
		})
	}
	return content
}
