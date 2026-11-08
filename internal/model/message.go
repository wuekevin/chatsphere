package model

import "time"

type Message struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ChannelID   uint64    `gorm:"not null;index" json:"channel_id"`
	UserID      *uint64   `gorm:"index" json:"user_id,omitempty"`
	DisplayName string    `gorm:"size:50;not null" json:"display_name"`
	Content     string    `gorm:"type:text;not null" json:"content"`
	IPHash      string    `gorm:"size:64" json:"-"`
	Country     string    `gorm:"size:2" json:"country,omitempty"`
	IsAnonymous bool      `gorm:"default:false" json:"is_anonymous"`
	IsDeleted   bool      `gorm:"default:false" json:"-"`
	AvatarColor string    `gorm:"size:7" json:"avatar_color"`
	Reputation  int       `gorm:"default:0" json:"reputation"`
	CreatedAt   time.Time `json:"created_at"`

	Channel Channel `gorm:"foreignKey:ChannelID" json:"-"`
	User    *User   `gorm:"foreignKey:UserID" json:"-"`
}

func (Message) TableName() string {
	return "messages"
}

type Report struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	MessageID uint64    `gorm:"not null;index" json:"message_id"`
	UserID    *uint64   `json:"user_id,omitempty"`
	Reason    string    `gorm:"size:255" json:"reason"`
	Status    string    `gorm:"type:varchar(20);default:'pending'" json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func (Report) TableName() string {
	return "reports"
}
