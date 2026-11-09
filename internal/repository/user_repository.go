package repository

import (
	"errors"

	"gorm.io/gorm"

	"github.com/wuekevin/chatsphere/internal/model"
)

type UserRepository interface {
	// GetByID returns ErrNotFound when no user matches or the record is deleted.
	GetByID(id uint64) (*model.User, error)
}

type gormUserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &gormUserRepository{db: db}
}

func (r *gormUserRepository) GetByID(id uint64) (*model.User, error) {
	var user model.User
	if err := r.db.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}
