package user

import (
	"context"
	"errors"
	"fmt"
	"github.com/aseptimu/AlgoTrack/internal/model"
	"github.com/aseptimu/AlgoTrack/internal/service"
	"github.com/aseptimu/AlgoTrack/internal/telegram/messages"
	"log/slog"
)

type TgUserRepository interface {
	Create(ctx context.Context, user *model.User) (*int64, error)
	Get(ctx context.Context, userId int64) (*model.User, error)
	UpdateGoal(ctx context.Context, userId int64, goal int64) error
	CountSolvedTasks(ctx context.Context, userId int64) (int64, error)
}

type TgUserService struct {
	repo   TgUserRepository
	logger *slog.Logger
}

func NewUserService(repo TgUserRepository, logger *slog.Logger) *TgUserService {
	return &TgUserService{
		repo,
		logger,
	}
}

func (t *TgUserService) SetGoal(ctx context.Context, userID, goal int64) error {
	if goal <= 0 {
		return service.ErrInvalidGoal
	}

	return t.repo.UpdateGoal(ctx, userID, goal)
}

func (t *TgUserService) BuildWelcomeMessage(ctx context.Context, user *model.User) (string, error) {
	if user.GoalTotal == nil || *user.GoalTotal <= 0 {
		return fmt.Sprintf(messages.WelcomeNoGoal, messages.Commands), nil
	}

	progress, err := t.GetProgress(ctx, user)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(
		messages.WelcomeWithProgress,
		progress.Solved,
		progress.GoalTotal,
		progress.Remaining,
		messages.Commands,
	), nil
}

func (t *TgUserService) EnsureExistsAndGet(ctx context.Context, incomingUser *model.User) (*model.User, error) {
	user, err := t.repo.Get(ctx, incomingUser.UserID)
	if errors.Is(err, service.ErrTgUserNotFound) {
		t.logger.Info("tg user not found, creating", "userID", incomingUser.UserID, "chatID", incomingUser.ChatID)

		if _, createErr := t.repo.Create(ctx, incomingUser); createErr != nil {
			t.logger.Error("failed to create tg user", "err", createErr, "userID", incomingUser.UserID, "chatID", incomingUser.ChatID)
			return nil, createErr
		}

		user, err = t.repo.Get(ctx, incomingUser.UserID)
		if err != nil {
			t.logger.Error("failed to get newly created tg user", "err", err, "userID", incomingUser.UserID, "chatID", incomingUser.ChatID)
			return nil, err
		}

		return user, nil
	}

	if err != nil {
		t.logger.Error("failed to get tg user", "err", err, "userID", incomingUser.UserID, "chatID", incomingUser.ChatID)
		return nil, err
	}

	return user, nil
}

func (t *TgUserService) GetProgress(ctx context.Context, user *model.User) (*model.UserProgress, error) {
	if user.GoalTotal == nil || *user.GoalTotal <= 0 {
		return nil, nil
	}

	solved, err := t.repo.CountSolvedTasks(ctx, user.UserID)
	if err != nil {
		t.logger.Error("failed to count solved tasks", "err", err, "userID", user.UserID)
		return nil, err
	}

	goal := *user.GoalTotal
	remaining := goal - solved
	if remaining < 0 {
		remaining = 0
	}

	return &model.UserProgress{
		GoalTotal: goal,
		Solved:    solved,
		Remaining: remaining,
	}, nil
}

func (t *TgUserService) CreateUser(ctx context.Context, user *model.User) (*int64, error) {
	return t.repo.Create(ctx, user)
}

func (t *TgUserService) GetUser(ctx context.Context, userId int64) (*model.User, error) {
	return t.repo.Get(ctx, userId)
}
