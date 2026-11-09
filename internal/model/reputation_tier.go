package model

type ReputationTier struct {
	ID         uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	Name       string `gorm:"size:50;not null" json:"name"`
	MinPoints  int    `gorm:"not null;default:0" json:"min_points"`
	BadgeEmoji string `gorm:"size:10" json:"badge_emoji,omitempty"`
	Color      string `gorm:"size:7" json:"color,omitempty"`
}

func (ReputationTier) TableName() string {
	return "reputation_tiers"
}
