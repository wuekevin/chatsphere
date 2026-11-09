package model

import (
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
	LastSeenAt   time.Time `json:"last_seen_at"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (User) TableName() string {
	return "users"
}

// Badge returns the display badge for the reputation tier.
// ANON is not a user-level concept here; callers that know IsAnonymous should skip the badge.
func (u *User) Badge() string {
	switch {
	case u.Reputation >= 500:
		return "GOLD"
	case u.Reputation >= 100:
		return "SILVER"
	default:
		return "BRONZE"
	}
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
		Badge:       u.Badge(),
		Role:        u.Role,
	}
}
