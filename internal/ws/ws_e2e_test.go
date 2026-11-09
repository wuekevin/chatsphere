package ws_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/wuekevin/chatsphere/internal/config"
	"github.com/wuekevin/chatsphere/internal/handler"
	"github.com/wuekevin/chatsphere/internal/model"
	"github.com/wuekevin/chatsphere/internal/repository"
	"github.com/wuekevin/chatsphere/internal/service"
	"github.com/wuekevin/chatsphere/internal/testutil"
	"github.com/wuekevin/chatsphere/internal/ws"
)

func init() { gin.SetMode(gin.TestMode) }

func buildServer(t *testing.T) *httptest.Server {
	t.Helper()
	db := testutil.MustDB(t)
	rdb := testutil.MustRedis(t)

	cfg := &config.Config{
		Server: config.ServerConfig{SecretKey: "testsecret"},
		Chat:   config.ChatConfig{MaxHistory: 50},
	}

	userReg := ws.NewUserRegistry()
	registry := ws.NewHubRegistry(db, rdb, userReg)

	chRepo := repository.NewChannelRepository(db)
	msgRepo := repository.NewMessageRepository(db)
	userRepo := repository.NewUserRepository(db)
	chSvc := service.NewChannelService(chRepo)
	msgSvc := service.NewMessageService(msgRepo)

	r := gin.New()
	r.Use(gin.Recovery())
	handler.RegisterAuth(r, db, cfg)
	handler.RegisterChat(r, chSvc, msgSvc, userRepo, registry, cfg)

	return httptest.NewServer(r)
}

func seedChannel(t *testing.T, db interface{ Create(interface{}) interface{ Error() error } }) *model.Channel {
	t.Helper()
	return nil // replaced below with real db
}

// dial opens a WebSocket to the test server's /ws/:channel endpoint.
func dial(t *testing.T, srv *httptest.Server, slug string, header http.Header) *websocket.Conn {
	t.Helper()
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws/" + slug
	conn, resp, err := websocket.DefaultDialer.Dial(url, header)
	if err != nil {
		t.Fatalf("dial %s: %v (status %v)", url, err, resp)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func readEvent(t *testing.T, conn *websocket.Conn) map[string]interface{} {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var ev map[string]interface{}
	if err := json.Unmarshal(msg, &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return ev
}

func readUntil(t *testing.T, conn *websocket.Conn, typ string) map[string]interface{} {
	t.Helper()
	for i := 0; i < 20; i++ {
		ev := readEvent(t, conn)
		if ev["type"] == typ {
			return ev
		}
	}
	t.Fatalf("never received event type %q", typ)
	return nil
}

// --- Tests ---

// TestAnonConnect verifies an anonymous user can join a channel and receives a joined event.
func TestAnonConnect(t *testing.T) {
	db := testutil.MustDB(t)
	ch := &model.Channel{
		Name: "general", Slug: "general",
		IsActive: true, AllowAnonymous: true,
		MaxMsgLength: 200, RateLimitSeconds: 1,
	}
	db.Create(ch)

	srv := buildServer(t)
	defer srv.Close()

	conn := dial(t, srv, "general", nil)
	ev := readUntil(t, conn, "joined")

	if ev["display_name"] == nil || ev["display_name"] == "" {
		t.Fatalf("joined event missing display_name: %v", ev)
	}
}

// TestSendAndBroadcast verifies a sent message is broadcast to all clients in the channel.
func TestSendAndBroadcast(t *testing.T) {
	db := testutil.MustDB(t)
	ch := &model.Channel{
		Name: "broadcast", Slug: "broadcast",
		IsActive: true, AllowAnonymous: true,
		MaxMsgLength: 200, RateLimitSeconds: 1,
	}
	db.Create(ch)

	srv := buildServer(t)
	defer srv.Close()

	sender := dial(t, srv, "broadcast", nil)
	readUntil(t, sender, "joined")

	receiver := dial(t, srv, "broadcast", nil)
	readUntil(t, receiver, "joined")

	// drain online_count events
	time.Sleep(100 * time.Millisecond)

	msg, _ := json.Marshal(map[string]string{"type": "send", "content": "hello world"})
	if err := sender.WriteMessage(websocket.TextMessage, msg); err != nil {
		t.Fatalf("write: %v", err)
	}

	ev := readUntil(t, receiver, "message")
	if ev["message"] == nil {
		t.Fatalf("expected message payload, got: %v", ev)
	}
	msgData := ev["message"].(map[string]interface{})
	if msgData["content"] != "hello world" {
		t.Errorf("content = %q, want %q", msgData["content"], "hello world")
	}
}

// TestBlockedWordFilter verifies messages containing blocked words are rejected.
func TestBlockedWordFilter(t *testing.T) {
	db := testutil.MustDB(t)
	ch := &model.Channel{
		Name: "filtered", Slug: "filtered",
		IsActive: true, AllowAnonymous: true,
		MaxMsgLength: 200, RateLimitSeconds: 1,
	}
	db.Create(ch)
	db.Create(&model.BlockedWord{Pattern: "badword", IsActive: true, IsRegex: false})

	srv := buildServer(t)
	defer srv.Close()

	conn := dial(t, srv, "filtered", nil)
	readUntil(t, conn, "joined")

	msg, _ := json.Marshal(map[string]string{"type": "send", "content": "this is badword content"})
	conn.WriteMessage(websocket.TextMessage, msg)

	ev := readUntil(t, conn, "error")
	if !strings.Contains(ev["error"].(string), "prohibited") {
		t.Errorf("expected prohibited error, got: %v", ev["error"])
	}
}

// TestBlockedWordBypassPrevented verifies HTML-entity bypass is blocked (< > & in word).
func TestBlockedWordBypassPrevented(t *testing.T) {
	db := testutil.MustDB(t)
	ch := &model.Channel{
		Name: "bypass", Slug: "bypass",
		IsActive: true, AllowAnonymous: true,
		MaxMsgLength: 200, RateLimitSeconds: 1,
	}
	db.Create(ch)
	db.Create(&model.BlockedWord{Pattern: "shit", IsActive: true, IsRegex: false})

	srv := buildServer(t)
	defer srv.Close()

	conn := dial(t, srv, "bypass", nil)
	readUntil(t, conn, "joined")

	// Attempt bypass: s<hit — after EscapeString would become s&lt;hit, missing the match
	msg, _ := json.Marshal(map[string]string{"type": "send", "content": "s<hit"})
	conn.WriteMessage(websocket.TextMessage, msg)

	ev := readUntil(t, conn, "error")
	if !strings.Contains(ev["error"].(string), "prohibited") {
		t.Errorf("bypass succeeded — expected prohibited error, got: %v", ev["error"])
	}
}

// TestRateLimit verifies the per-channel rate limiter rejects rapid successive sends.
func TestRateLimit(t *testing.T) {
	db := testutil.MustDB(t)
	ch := &model.Channel{
		Name: "ratelimit", Slug: "ratelimit",
		IsActive: true, AllowAnonymous: true,
		MaxMsgLength: 200, RateLimitSeconds: 10,
	}
	db.Create(ch)

	srv := buildServer(t)
	defer srv.Close()

	conn := dial(t, srv, "ratelimit", nil)
	readUntil(t, conn, "joined")

	send := func(c string) {
		msg, _ := json.Marshal(map[string]string{"type": "send", "content": c})
		conn.WriteMessage(websocket.TextMessage, msg)
	}

	send("first message")
	readUntil(t, conn, "message")

	send("second message — too fast")
	ev := readUntil(t, conn, "error")
	if !strings.Contains(ev["error"].(string), "wait") {
		t.Errorf("expected rate-limit error, got: %v", ev["error"])
	}
}

// TestAnonBlockedOnPrivateChannel verifies that a channel with AllowAnonymous=false rejects anonymous WS.
func TestAnonBlockedOnPrivateChannel(t *testing.T) {
	db := testutil.MustDB(t)
	db.Create(&model.Channel{
		Name: "private", Slug: "private",
		IsActive: true, AllowAnonymous: false,
		MaxMsgLength: 200, RateLimitSeconds: 1,
	})

	srv := buildServer(t)
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws/private"
	_, resp, err := websocket.DefaultDialer.Dial(url, nil)
	if err == nil {
		t.Fatal("expected connection to be refused, got nil error")
	}
	if resp != nil && resp.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want 403", resp.StatusCode)
	}
}

// TestMessageHistory verifies that a new client receives recent message history on connect.
func TestMessageHistory(t *testing.T) {
	db := testutil.MustDB(t)
	ch := &model.Channel{
		Name: "history", Slug: "history",
		IsActive: true, AllowAnonymous: true,
		MaxMsgLength: 200, RateLimitSeconds: 1,
	}
	db.Create(ch)

	srv := buildServer(t)
	defer srv.Close()

	// First client sends a message.
	c1 := dial(t, srv, "history", nil)
	readUntil(t, c1, "joined")
	msg, _ := json.Marshal(map[string]string{"type": "send", "content": "persisted message"})
	c1.WriteMessage(websocket.TextMessage, msg)
	readUntil(t, c1, "message")
	time.Sleep(200 * time.Millisecond)
	c1.Close()

	// Second client joins later and should receive history.
	c2 := dial(t, srv, "history", nil)
	readUntil(t, c2, "joined")
	ev := readUntil(t, c2, "history")
	msgs, ok := ev["messages"].([]interface{})
	if !ok || len(msgs) == 0 {
		t.Fatalf("expected history messages, got: %v", ev)
	}
}

// TestOnlineCount verifies the online_count event is broadcast on connect/disconnect.
func TestOnlineCount(t *testing.T) {
	db := testutil.MustDB(t)
	db.Create(&model.Channel{
		Name: "online", Slug: "online",
		IsActive: true, AllowAnonymous: true,
		MaxMsgLength: 200, RateLimitSeconds: 1,
	})

	srv := buildServer(t)
	defer srv.Close()

	c1 := dial(t, srv, "online", nil)
	readUntil(t, c1, "joined")

	c2 := dial(t, srv, "online", nil)
	readUntil(t, c2, "joined")

	// c1 should receive an online_count after c2 joins
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = ctx

	ev := readUntil(t, c1, "online_count")
	count, ok := ev["count"].(float64)
	if !ok || count < 1 {
		t.Errorf("online_count = %v, want >= 1", ev["count"])
	}
}
