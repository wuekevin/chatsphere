package model

import "time"

// BotConfig defines a scheduled recommendation bot.
type BotConfig struct {
	ID             uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name           string     `gorm:"size:100;not null" json:"name"`
	AvatarColor    string     `gorm:"size:7;default:'#6366f1'" json:"avatar_color"`
	ChannelID      *uint64    `gorm:"index" json:"channel_id,omitempty"`
	IntervalHours  int        `gorm:"not null;default:6" json:"interval_hours"`
	WebhookURL     string     `gorm:"size:500;not null" json:"webhook_url"`
	TelegramToken  string     `gorm:"size:200" json:"telegram_token,omitempty"`
	TelegramChatID string     `gorm:"size:100" json:"telegram_chat_id,omitempty"`
	IsActive       bool       `gorm:"default:true" json:"is_active"`
	LastFiredAt    *time.Time `json:"last_fired_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (BotConfig) TableName() string { return "bot_configs" }

// BotCard is a single recommendation card returned by a bot webhook.
type BotCard struct {
	Title  string `json:"title"`
	Poster string `json:"poster"`
	Desc   string `json:"desc"`
	Link   string `json:"link,omitempty"`
}

// BotCardPayload is the JSON structure a webhook must return.
type BotCardPayload struct {
	Cards []BotCard `json:"cards"`
}
