package repository

import (
	"errors"

	"gorm.io/gorm"

	"github.com/wuekevin/chatsphere/internal/model"
)

// ErrNotFound is returned by repository methods when a record does not exist.
var ErrNotFound = errors.New("record not found")

type ChannelRepository interface {
	ListActive() ([]model.Channel, error)
	// GetBySlug returns the channel with the given slug when it is active.
	// Applies is_active = true filter; returns ErrNotFound when missing or inactive.
	// Note: wsHandler (Story 2.3) also applies this filter directly — keep in sync.
	GetBySlug(slug string) (*model.Channel, error)
}

type gormChannelRepository struct {
	db *gorm.DB
}

func NewChannelRepository(db *gorm.DB) ChannelRepository {
	return &gormChannelRepository{db: db}
}

func (r *gormChannelRepository) ListActive() ([]model.Channel, error) {
	var channels []model.Channel
	err := r.db.Where("is_active = true").
		Order("is_default DESC, name ASC").
		Find(&channels).Error
	return channels, err
}

func (r *gormChannelRepository) GetBySlug(slug string) (*model.Channel, error) {
	var ch model.Channel
	err := r.db.Where("slug = ? AND is_active = true", slug).First(&ch).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &ch, nil
}
