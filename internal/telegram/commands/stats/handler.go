package stats

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/aseptimu/AlgoTrack/internal/model"
	"github.com/aseptimu/AlgoTrack/internal/telegram/reply"
	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type StatsProvider interface {
	GetUserStats(ctx context.Context, userID int64, user *model.User) (*model.UserStatsResult, error)
}

type UserEnsurer interface {
	EnsureExistsAndGet(ctx context.Context, user *model.User) (*model.User, error)
}

type Handler struct {
	stats StatsProvider
	users UserEnsurer
	log   *slog.Logger
}

func New(stats StatsProvider, users UserEnsurer, log *slog.Logger) *Handler {
	if log == nil {
		log = slog.Default()
	}
	return &Handler{stats: stats, users: users, log: log}
}

func (h *Handler) Handle(ctx context.Context, b *tgbot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.From == nil {
		return
	}

	h.log.Info("stats command received")

	chatID := update.Message.Chat.ID
	userID := update.Message.From.ID
	username := update.Message.From.Username

	user := &model.User{
		UserID:   userID,
		ChatID:   chatID,
		Username: username,
	}

	ensuredUser, err := h.users.EnsureExistsAndGet(ctx, user)
	if err != nil {
		h.log.Error("failed to ensure user for stats", "err", err, "userID", userID)
		reply.Text(ctx, b, chatID, "Something went wrong. Try again later.")
		return
	}

	result, err := h.stats.GetUserStats(ctx, ensuredUser.UserID, ensuredUser)
	if err != nil {
		h.log.Error("failed to get user stats", "err", err, "userID", userID)
		reply.Text(ctx, b, chatID, "Something went wrong. Try again later.")
		return
	}

	reply.HTML(ctx, b, chatID, buildStatsMessage(result))
}

func buildStatsMessage(result *model.UserStatsResult) string {
	var sb strings.Builder

	sb.WriteString("<b>Your Progress Dashboard</b>\n\n")

	// Total solved.
	fmt.Fprintf(&sb, "Solved: <b>%d</b>\n", result.Stats.Total)
	fmt.Fprintf(&sb, "  Easy: <b>%d</b>\n", result.Stats.Easy)
	fmt.Fprintf(&sb, "  Medium: <b>%d</b>\n", result.Stats.Medium)
	fmt.Fprintf(&sb, "  Hard: <b>%d</b>\n", result.Stats.Hard)

	// Streak.
	fmt.Fprintf(&sb, "\nStreak: <b>%d</b> days\n", result.Streak)

	// Pending reviews.
	fmt.Fprintf(&sb, "Pending reviews: <b>%d</b>\n", result.PendingReviews)

	// Goal progress.
	if result.GoalProgress != nil && len(result.GoalProgress.Items) > 0 {
		sb.WriteString("\n<b>Goals</b>\n")
		for _, item := range result.GoalProgress.Items {
			badge := goalBadge(item.Label)
			pct := int64(0)
			if item.Goal > 0 {
				pct = item.Solved * 100 / item.Goal
			}
			bar := progressBar(pct)
			fmt.Fprintf(&sb, "%s %s <b>%d / %d</b> (%d%%)\n", badge, bar, item.Solved, item.Goal, pct)
		}
	}

	return sb.String()
}

func goalBadge(label string) string {
	switch label {
	case "Total":
		return "🎯"
	case "Easy":
		return "🟢"
	case "Medium":
		return "🟠"
	case "Hard":
		return "🔴"
	default:
		return "•"
	}
}

func progressBar(pct int64) string {
	const barLen = 10
	filled := pct * barLen / 100
	if filled > barLen {
		filled = barLen
	}

	return strings.Repeat("▓", int(filled)) + strings.Repeat("░", int(barLen-filled))
}
