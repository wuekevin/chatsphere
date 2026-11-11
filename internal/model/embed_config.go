package model

import "time"

// EmbedConfig stores the widget embed settings that get baked into the served chatsphere.js.
// Only one row is used (ID=1); upsert on save.
type EmbedConfig struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	ServerURL   string    `gorm:"size:255;default:''" json:"server_url"`
	Channel     string    `gorm:"size:100;default:'general'" json:"channel"`
	Position    string    `gorm:"size:20;default:'bottom-right'" json:"position"`
	Offset      int       `gorm:"default:20" json:"offset"`
	Title       string    `gorm:"size:200;default:''" json:"title"`
	Notice      string    `gorm:"type:text" json:"notice"`
	AccentColor string    `gorm:"size:7;default:''" json:"accent_color"`
	Lang        string    `gorm:"size:5;default:'zh'" json:"lang"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (EmbedConfig) TableName() string { return "embed_configs" }
