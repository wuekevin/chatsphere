package model

import "time"

type DirectMessage struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	FromUserID uint64    `gorm:"not null;index" json:"from_user_id"`
	ToUserID   uint64    `gorm:"not null;index" json:"to_user_id"`
	Content    string    `gorm:"type:text;not null" json:"content"`
	IsRead     bool      `gorm:"default:false" json:"is_read"`
	CreatedAt  time.Time `json:"created_at"`

	// Transient fields — not persisted, populated on delivery.
	FromUsername    string `gorm:"-" json:"from_username,omitempty"`
	FromAvatarColor string `gorm:"-" json:"from_avatar_color,omitempty"`
	ToUsername      string `gorm:"-" json:"to_username,omitempty"`
	ToAvatarColor   string `gorm:"-" json:"to_avatar_color,omitempty"`
}

func (DirectMessage) TableName() string { return "direct_messages" }
