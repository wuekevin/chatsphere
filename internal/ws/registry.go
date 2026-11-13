package ws

import (
	"sync"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// UserRegistry maps authenticated user IDs to their active Client connections
// across all channels, enabling cross-channel DM delivery.
type UserRegistry struct {
	mu      sync.RWMutex
	clients map[uint64][]*Client
}

func NewUserRegistry() *UserRegistry {
	return &UserRegistry{clients: make(map[uint64][]*Client)}
}

func (ur *UserRegistry) Register(userID uint64, c *Client) {
	ur.mu.Lock()
	ur.clients[userID] = append(ur.clients[userID], c)
	ur.mu.Unlock()
}

func (ur *UserRegistry) Unregister(userID uint64, c *Client) {
	ur.mu.Lock()
	defer ur.mu.Unlock()
	list := ur.clients[userID]
	for i, cc := range list {
		if cc == c {
			ur.clients[userID] = append(list[:i], list[i+1:]...)
			break
		}
	}
	if len(ur.clients[userID]) == 0 {
		delete(ur.clients, userID)
	}
}

// Send delivers payload to all connections of userID. Returns true if at least
// one connection was reachable.
func (ur *UserRegistry) Send(userID uint64, payload []byte) bool {
	ur.mu.RLock()
	list := ur.clients[userID]
	// copy slice to avoid holding lock during send
	targets := make([]*Client, len(list))
	copy(targets, list)
	ur.mu.RUnlock()

	for _, c := range targets {
		select {
		case c.send <- payload:
		default:
		}
	}
	return len(targets) > 0
}

// HubRegistry manages per-channel Hub goroutines with lazy creation.
type HubRegistry struct {
	hubs        sync.Map // slug → *Hub
	channelHubs sync.Map // channelID (uint64) → *Hub
	db          *gorm.DB
	rdb         *redis.Client
	userReg     *UserRegistry
	tgNotifier  TelegramNotifier
}

func NewHubRegistry(db *gorm.DB, rdb *redis.Client, userReg *UserRegistry) *HubRegistry {
	return &HubRegistry{db: db, rdb: rdb, userReg: userReg}
}

// SetTelegramNotifier wires a Telegram notifier into all hubs created hereafter.
func (r *HubRegistry) SetTelegramNotifier(tn TelegramNotifier) {
	r.tgNotifier = tn
}

// IsOnline returns true if userID has at least one active WebSocket connection.
func (ur *UserRegistry) IsOnline(userID uint64) bool {
	ur.mu.RLock()
	defer ur.mu.RUnlock()
	return len(ur.clients[userID]) > 0
}

// GetOrCreate returns the Hub for the given channel, creating and starting one if needed.
// Safe for concurrent use.
func (r *HubRegistry) GetOrCreate(slug string, channelID uint64) *Hub {
	if v, ok := r.hubs.Load(slug); ok {
		return v.(*Hub)
	}
	hub := newHub(slug, channelID, r.db, r.rdb, r.userReg, r.tgNotifier)
	if actual, loaded := r.hubs.LoadOrStore(slug, hub); loaded {
		return actual.(*Hub)
	}
	r.channelHubs.Store(channelID, hub)
	go hub.Run()
	return hub
}

// BroadcastAll broadcasts an event to every active hub.
func (r *HubRegistry) BroadcastAll(evt Event) {
	r.hubs.Range(func(_, v any) bool {
		v.(*Hub).BroadcastJSON(evt)
		return true
	})
}

// InvalidateBlockedCacheAll resets the blocked-word cache on every active hub,
// forcing a DB reload on the next message through each hub.
func (r *HubRegistry) InvalidateBlockedCacheAll() {
	r.hubs.Range(func(_, v any) bool {
		v.(*Hub).InvalidateBlockedCache()
		return true
	})
}

// BroadcastToChannel broadcasts to the hub identified by channelID (if active).
func (r *HubRegistry) BroadcastToChannel(channelID uint64, evt Event) {
	if v, ok := r.channelHubs.Load(channelID); ok {
		v.(*Hub).BroadcastJSON(evt)
	}
}

// PersistBotEventToChannel saves a bot event to the Redis history of one channel.
func (r *HubRegistry) PersistBotEventToChannel(channelID uint64, evt Event) {
	if v, ok := r.channelHubs.Load(channelID); ok {
		v.(*Hub).PersistBotEvent(evt)
	}
}

// PersistBotEventAll saves a bot event to the Redis history of every active channel.
func (r *HubRegistry) PersistBotEventAll(evt Event) {
	r.hubs.Range(func(_, v any) bool {
		v.(*Hub).PersistBotEvent(evt)
		return true
	})
}
