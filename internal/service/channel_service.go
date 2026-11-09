package service

import (
	"github.com/wuekevin/chatsphere/internal/model"
	"github.com/wuekevin/chatsphere/internal/repository"
)

type ChannelService struct {
	repo repository.ChannelRepository
}

func NewChannelService(repo repository.ChannelRepository) *ChannelService {
	return &ChannelService{repo: repo}
}

func (s *ChannelService) ListActive() ([]model.Channel, error) {
	return s.repo.ListActive()
}

func (s *ChannelService) GetBySlug(slug string) (*model.Channel, error) {
	return s.repo.GetBySlug(slug)
}
