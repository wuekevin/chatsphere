package model

import (
	"fmt"
	"time"
)

type Role string

const (
	RoleAdmin     Role = "admin"
	RoleModerator Role = "moderator"
	RoleMember    Role = "member"
)

type User struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Username     string    `gorm:"uniqueIndex;size:50;not null" json:"username"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	Email        string    `gorm:"size:255" json:"email,omitempty"`
	AvatarColor  string    `gorm:"size:7" json:"avatar_color"`
	Country      string    `gorm:"size:2" json:"country,omitempty"`
	Reputation   int       `gorm:"default:0" json:"reputation"`
	Role         Role      `gorm:"type:varchar(20);default:'member'" json:"role"`
	IsBanned     bool      `gorm:"default:false" json:"is_banned"`
	LastSeenAt   *time.Time `gorm:"default:null" json:"last_seen_at,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (User) TableName() string {
	return "users"
}

// levelThresholds maps experience points to level numbers.
// Experience = total messages sent (Reputation field).
var levelThresholds = []int{0, 10, 50, 150, 300, 500, 1000, 2000, 5000, 10000}

// LevelNames holds Chinese and English names for LV.1–LV.10.
var LevelNames = [][2]string{
	{"初入", "Novice"},
	{"探索", "Explorer"},
	{"进阶", "Advanced"},
	{"精英", "Elite"},
	{"卓越", "Premium"},
	{"大师", "Master"},
	{"宗师", "Grandmaster"},
	{"传奇", "Legend"},
	{"至尊", "Supreme"},
	{"永恒", "Eternal"},
}

// Level returns the numeric level (1–10) based on Reputation (experience points).
func (u *User) Level() int {
	if u.Role == RoleAdmin {
		return 0 // special — displayed as Founder
	}
	lv := 1
	for i, threshold := range levelThresholds {
		if u.Reputation >= threshold {
			lv = i + 1
		}
	}
	if lv > 10 {
		lv = 10
	}
	return lv
}

// LevelTag returns a compact tag like "LV.3" (or "LV.0" for admin placeholder, handled by callers).
func (u *User) LevelTag() string {
	if u.Role == RoleAdmin {
		return "LV.MAX"
	}
	return fmt.Sprintf("LV.%d", u.Level())
}

// PublicProfile is the subset of User safe to send to other clients.
type PublicProfile struct {
	ID          uint64 `json:"id"`
	Username    string `json:"username"`
	AvatarColor string `json:"avatar_color"`
	Country     string `json:"country,omitempty"`
	Reputation  int    `json:"reputation"`
	Badge       string `json:"badge"`
	Role        Role   `json:"role"`
}

func (u *User) PublicProfile() PublicProfile {
	return PublicProfile{
		ID:          u.ID,
		Username:    u.Username,
		AvatarColor: u.AvatarColor,
		Country:     u.Country,
		Reputation:  u.Reputation,
		Badge:       u.LevelTag(),
		Role:        u.Role,
	}
}
