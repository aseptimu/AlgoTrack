package service

import (
	"context"
	"errors"
	"github.com/aseptimu/AlgoTrack/internal/model"
	"log/slog"
)

type TgUserCreator interface {
	Create(ctx context.Context, user *model.User) (*int64, error)
	Get(ctx context.Context, userId int64) (*model.User, error)
}

type TgUserService struct {
	repo   TgUserCreator
	logger *slog.Logger
}

func NewUserService(repo TgUserCreator, logger *slog.Logger) *TgUserService {
	return &TgUserService{
		repo,
		logger,
	}
}

func (t *TgUserService) EnsureExists(ctx context.Context, user *model.User) error {
	_, err := t.GetUser(ctx, user.UserID)
	if errors.Is(err, ErrTgUserNotFound) {
		if _, err = t.CreateUser(ctx, user); err != nil {
			t.logger.Error("failed to create tg user", "err", err, "userID", user.UserID, "chatID", user.ChatID)
			return err
		}
	} else if err != nil {
		t.logger.Error("failed to get tg user", "err", err, "userID", user.UserID, "chatID", user.ChatID)
		return err
	}
	return nil
}

func (t *TgUserService) CreateUser(ctx context.Context, user *model.User) (*int64, error) {
	_, err := t.repo.Create(ctx, user)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (t *TgUserService) GetUser(ctx context.Context, userId int64) (*model.User, error) {
	user, err := t.repo.Get(ctx, userId)
	if err != nil {
		return nil, err
	}
	return user, err
}
