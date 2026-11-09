package repository

import (
	"gorm.io/gorm"

	"github.com/wuekevin/chatsphere/internal/model"
)

type MessageRepository interface {
	// ListByChannel returns messages for a channel ordered by created_at DESC (newest first).
	// Callers that need chronological display (oldest first) must reverse the slice.
	ListByChannel(channelID uint64, limit int) ([]model.Message, error)
	Create(msg *model.Message) error
}

type gormMessageRepository struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) MessageRepository {
	return &gormMessageRepository{db: db}
}

func (r *gormMessageRepository) ListByChannel(channelID uint64, limit int) ([]model.Message, error) {
	var messages []model.Message
	err := r.db.Where("channel_id = ? AND is_deleted = false", channelID).
		Order("created_at DESC").
		Limit(limit).
		Find(&messages).Error
	return messages, err
}

func (r *gormMessageRepository) Create(msg *model.Message) error {
	return r.db.Create(msg).Error
}
