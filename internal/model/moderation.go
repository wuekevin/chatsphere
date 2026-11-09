package model

import "time"

type BlockedWord struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ChannelID *uint64   `gorm:"index" json:"channel_id,omitempty"` // nil = global
	Pattern   string    `gorm:"size:255;not null" json:"pattern"`
	IsRegex   bool      `gorm:"default:false" json:"is_regex"`
	IsActive  bool      `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

func (BlockedWord) TableName() string {
	return "blocked_words"
}
