package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"gorm.io/gorm"

	"github.com/wuekevin/chatsphere/internal/model"
	"github.com/wuekevin/chatsphere/internal/ws"
)

// Scheduler polls the database and fires bots on their configured intervals.
type Scheduler struct {
	db      *gorm.DB
	hubReg  *ws.HubRegistry
	stop    chan struct{}
}

func NewScheduler(db *gorm.DB, hubReg *ws.HubRegistry) *Scheduler {
	return &Scheduler{db: db, hubReg: hubReg, stop: make(chan struct{})}
}

func (s *Scheduler) Start() {
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.tick()
			case <-s.stop:
				return
			}
		}
	}()
}

func (s *Scheduler) Stop() { close(s.stop) }

func (s *Scheduler) tick() {
	var bots []model.BotConfig
	if err := s.db.Where("is_active = true").Find(&bots).Error; err != nil {
		return
	}
	now := time.Now()
	for _, b := range bots {
		due := b.LastFiredAt == nil || now.Sub(*b.LastFiredAt) >= time.Duration(b.IntervalHours)*time.Hour
		if due {
			go s.fire(b)
		}
	}
}

// FireBot fires a bot immediately (used by admin "fire now" endpoint).
func (s *Scheduler) FireBot(b model.BotConfig) { go s.fire(b) }

func (s *Scheduler) fire(b model.BotConfig) {
	payload, err := s.fetchCards(b.WebhookURL)
	if err != nil || len(payload.Cards) == 0 {
		log.Printf("[bot] webhook fetch error for bot %d: %v", b.ID, err)
		return
	}

	now := time.Now()
	s.db.Model(&model.BotConfig{}).Where("id = ?", b.ID).Update("last_fired_at", now)

	evt := ws.Event{
		Type:        "bot_card",
		Content:     b.Name,
		AvatarColor: b.AvatarColor,
		BotCards:    payload.Cards,
	}

	if b.ChannelID != nil {
		s.hubReg.PersistBotEventToChannel(*b.ChannelID, evt)
		s.hubReg.BroadcastToChannel(*b.ChannelID, evt)
	} else {
		s.hubReg.PersistBotEventAll(evt)
		s.hubReg.BroadcastAll(evt)
	}

	if b.TelegramToken != "" && b.TelegramChatID != "" {
		go s.postToTelegram(b.TelegramToken, b.TelegramChatID, b.Name, payload.Cards)
	}
}

func (s *Scheduler) fetchCards(url string) (*model.BotCardPayload, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var p model.BotCardPayload
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Scheduler) postToTelegram(token, chatID, botName string, cards []model.BotCard) {
	apiBase := fmt.Sprintf("https://api.telegram.org/bot%s", token)
	for _, card := range cards {
		s.sendTelegramCard(apiBase, chatID, botName, card)
	}
}

func (s *Scheduler) sendTelegramCard(apiBase, chatID, botName string, card model.BotCard) {
	caption := fmt.Sprintf("🎬 <b>%s</b>\n\n%s", htmlEscape(card.Title), htmlEscape(card.Desc))
	if card.Link != "" {
		caption += fmt.Sprintf("\n\n<a href=\"%s\">详情</a>", card.Link)
	}

	if card.Poster != "" {
		body, _ := json.Marshal(map[string]interface{}{
			"chat_id":    chatID,
			"photo":      card.Poster,
			"caption":    caption,
			"parse_mode": "HTML",
		})
		resp, err := http.Post(apiBase+"/sendPhoto", "application/json", bytes.NewReader(body))
		if err == nil {
			resp.Body.Close()
			return
		}
	}

	body, _ := json.Marshal(map[string]interface{}{
		"chat_id":    chatID,
		"text":       caption,
		"parse_mode": "HTML",
	})
	resp, err := http.Post(apiBase+"/sendMessage", "application/json", bytes.NewReader(body))
	if err == nil {
		resp.Body.Close()
	}
}

func htmlEscape(s string) string {
	s = replaceAll(s, "&", "&amp;")
	s = replaceAll(s, "<", "&lt;")
	s = replaceAll(s, ">", "&gt;")
	return s
}

func replaceAll(s, old, new string) string {
	result := []byte{}
	for i := 0; i < len(s); {
		if i+len(old) <= len(s) && s[i:i+len(old)] == old {
			result = append(result, new...)
			i += len(old)
		} else {
			result = append(result, s[i])
			i++
		}
	}
	return string(result)
}
