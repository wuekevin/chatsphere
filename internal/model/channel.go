package model

import "time"

type Channel struct {
	ID               uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name             string    `gorm:"size:100;not null" json:"name"`
	Slug             string    `gorm:"uniqueIndex;size:100;not null" json:"slug"`
	Description      string    `gorm:"type:text" json:"description"`
	IsDefault        bool      `gorm:"default:false" json:"is_default"`
	IsActive         bool      `gorm:"default:true" json:"is_active"`
	MaxMsgLength     int       `gorm:"default:200" json:"max_msg_length"`
	RateLimitSeconds int       `gorm:"default:3" json:"rate_limit_seconds"`
	AllowAnonymous   bool      `gorm:"default:true" json:"allow_anonymous"`
	AllowDM          bool      `gorm:"default:true" json:"allow_dm"`
	WelcomeMessage   string    `gorm:"type:text" json:"welcome_message,omitempty"`
	TelegramURL      string    `gorm:"size:255" json:"telegram_url,omitempty"`
	AccentColor      string    `gorm:"size:7" json:"accent_color,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (Channel) TableName() string {
	return "channels"
}
