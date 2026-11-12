package model

import "time"

type AllowedDomain struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Domain    string    `gorm:"size:255;uniqueIndex;not null" json:"domain"`
	CreatedAt time.Time `json:"created_at"`
}

func (AllowedDomain) TableName() string { return "allowed_domains" }
