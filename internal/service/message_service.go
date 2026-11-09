package service

import (
	"github.com/wuekevin/chatsphere/internal/model"
	"github.com/wuekevin/chatsphere/internal/repository"
)

type MessageService struct {
	repo repository.MessageRepository
}

func NewMessageService(repo repository.MessageRepository) *MessageService {
	return &MessageService{repo: repo}
}

func (s *MessageService) ListByChannel(channelID uint64, limit int) ([]model.Message, error) {
	return s.repo.ListByChannel(channelID, limit)
}

func (s *MessageService) Create(msg *model.Message) error {
	return s.repo.Create(msg)
}
