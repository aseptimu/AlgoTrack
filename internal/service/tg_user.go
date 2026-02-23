package service

import (
	"context"
	"github.com/aseptimu/AlgoTrack/internal/model"
)

type TgUserCreator interface {
	Create(ctx context.Context, userId int64, chatId int64, username string) (*int64, error)
	Get(ctx context.Context, userId int64) (*model.User, error)
}

type TgUserService struct {
	repo TgUserCreator
}

func NewUserService(repo TgUserCreator) *TgUserService {
	return &TgUserService{
		repo: repo,
	}
}

func (t *TgUserService) Create(ctx context.Context, user *model.User) error {
	_, err := t.repo.Create(ctx, user.UserID, user.ChatID, user.Username)
	if err != nil {
		return err
	}
	return nil
}

func (t *TgUserService) Get(ctx context.Context, userId int64) (*model.User, error) {
	user, err := t.repo.Get(ctx, userId)
	if err != nil {
		return nil, err
	}
	return user, err
}
