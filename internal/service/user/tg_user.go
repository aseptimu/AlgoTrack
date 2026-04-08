package user

import (
	"context"
	"errors"
	"fmt"
	"github.com/aseptimu/AlgoTrack/internal/model"
	"github.com/aseptimu/AlgoTrack/internal/service"
	"github.com/aseptimu/AlgoTrack/internal/telegram/messages"
	"log/slog"
	"strings"
)

type TgUserRepository interface {
	Create(ctx context.Context, user *model.User) (*int64, error)
	Get(ctx context.Context, userId int64) (*model.User, error)
	UpdateGoal(ctx context.Context, userId int64, goal int64, difficulty *string) error
	UpdateLeetCodeUsername(ctx context.Context, userID int64, leetcodeUsername string) error
	GetUsersWithLeetCode(ctx context.Context) ([]model.User, error)
}

type ProgressProvider interface {
	GetStats(ctx context.Context, userID int64) (*model.TaskStats, error)
}

type TgUserService struct {
	repo     TgUserRepository
	progress ProgressProvider
	logger   *slog.Logger
}

func NewUserService(repo TgUserRepository, progress ProgressProvider, logger *slog.Logger) *TgUserService {
	return &TgUserService{
		repo,
		progress,
		logger,
	}
}

func normalizeDifficulty(difficulty *string) (*string, error) {
	if difficulty == nil {
		return nil, nil
	}

	value := strings.TrimSpace(*difficulty)

	switch value {
	case "easy", "Easy", "EASY":
		value := "Easy"
		return &value, nil
	case "medium", "Medium", "MEDIUM":
		value := "Medium"
		return &value, nil
	case "hard", "Hard", "HARD":
		value := "Hard"
		return &value, nil
	case "":
		return nil, nil
	default:
		return nil, service.ErrInvalidDifficulty
	}
}

func (t *TgUserService) SetGoal(ctx context.Context, userID, goal int64, difficulty *string) error {
	if goal <= 0 {
		return service.ErrInvalidGoal
	}

	normalizedDifficulty, err := normalizeDifficulty(difficulty)
	if err != nil {
		return err
	}

	return t.repo.UpdateGoal(ctx, userID, goal, normalizedDifficulty)
}

func (t *TgUserService) BuildWelcomeMessage(ctx context.Context, user *model.User) (string, error) {
	if !hasAnyGoal(user) {
		return fmt.Sprintf(messages.WelcomeNoGoal, messages.Commands), nil
	}

	progress, err := t.GetProgress(ctx, user)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(
		messages.WelcomeWithProgress(formatGoalProgress(progress.Items)),
		messages.Commands,
	), nil
}

func (t *TgUserService) BuildGoalMessage(ctx context.Context, user *model.User) (string, error) {
	progress, err := t.GetProgress(ctx, user)
	if err != nil {
		return "", err
	}

	if progress == nil || len(progress.Items) == 0 {
		return messages.GoalSavedNoProgress, nil
	}

	return fmt.Sprintf(messages.GoalSavedWithProgress, formatGoalProgress(progress.Items)), nil
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
	if !hasAnyGoal(user) {
		return nil, nil
	}

	stats, err := t.progress.GetStats(ctx, user.UserID)
	if err != nil {
		t.logger.Error("failed to get task stats", "err", err, "userID", user.UserID)
		return nil, err
	}

	items := make([]model.GoalProgress, 0, 4)
	appendGoal := func(label string, goal *int64, solved int64) {
		if goal == nil || *goal <= 0 {
			return
		}

		remaining := *goal - solved
		if remaining < 0 {
			remaining = 0
		}

		items = append(items, model.GoalProgress{
			Label:     label,
			Solved:    solved,
			Goal:      *goal,
			Remaining: remaining,
		})
	}

	appendGoal("Total", user.GoalTotal, stats.Total)
	appendGoal("Easy", user.GoalEasy, stats.Easy)
	appendGoal("Medium", user.GoalMedium, stats.Medium)
	appendGoal("Hard", user.GoalHard, stats.Hard)

	return &model.UserProgress{
		Items: items,
	}, nil
}

func hasAnyGoal(user *model.User) bool {
	goals := []*int64{user.GoalTotal, user.GoalEasy, user.GoalMedium, user.GoalHard}
	for _, goal := range goals {
		if goal != nil && *goal > 0 {
			return true
		}
	}

	return false
}

func formatGoalProgress(items []model.GoalProgress) string {
	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, fmt.Sprintf("%s <b>%d / %d</b> <i>(remaining %d)</i>", goalBadge(item.Label), item.Solved, item.Goal, item.Remaining))
	}

	return strings.Join(lines, "\n")
}

func goalBadge(label string) string {
	switch label {
	case "Total":
		return "🎯 <b>Total</b>"
	case "Easy":
		return "🟢 <b>Easy</b>"
	case "Medium":
		return "🟠 <b>Medium</b>"
	case "Hard":
		return "🔴 <b>Hard</b>"
	default:
		return "<b>" + label + "</b>"
	}
}

func (t *TgUserService) CreateUser(ctx context.Context, user *model.User) (*int64, error) {
	return t.repo.Create(ctx, user)
}

func (t *TgUserService) GetUser(ctx context.Context, userId int64) (*model.User, error) {
	return t.repo.Get(ctx, userId)
}

func (t *TgUserService) LinkLeetCode(ctx context.Context, userID int64, leetcodeUsername string) error {
	if leetcodeUsername == "" {
		return service.ErrInvalidLeetCodeUsername
	}
	return t.repo.UpdateLeetCodeUsername(ctx, userID, leetcodeUsername)
}

func (t *TgUserService) GetUsersWithLeetCode(ctx context.Context) ([]model.User, error) {
	return t.repo.GetUsersWithLeetCode(ctx)
}
