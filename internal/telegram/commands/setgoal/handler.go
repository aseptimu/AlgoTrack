package setgoal

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/aseptimu/AlgoTrack/internal/service"
	"github.com/aseptimu/AlgoTrack/internal/telegram/helpers"
	"github.com/aseptimu/AlgoTrack/internal/telegram/messages"
	"github.com/aseptimu/AlgoTrack/internal/telegram/reply"
	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"log/slog"
)

type Handler struct {
	userManager service.UserManager
	logger      *slog.Logger
}

func New(userManager service.UserManager, logger *slog.Logger) *Handler {
	return &Handler{
		userManager: userManager,
		logger:      logger,
	}
}

func (h *Handler) Handle(ctx context.Context, b *tgbot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	chatID := update.Message.Chat.ID
	text := strings.TrimSpace(update.Message.Text)

	incomingUser := helpers.GetUser(update)
	user, err := h.userManager.EnsureExistsAndGet(ctx, incomingUser)
	if err != nil {
		h.logger.Error("failed to ensure user exists", "err", err, "userID", incomingUser.UserID)
		reply.Text(ctx, b, chatID, messages.InternalError)
		return
	}

	parts := strings.Fields(text)
	if len(parts) < 2 || len(parts) > 3 {
		reply.Text(ctx, b, chatID, messages.GoalUsage)
		return
	}

	goal, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || goal <= 0 {
		reply.Text(ctx, b, chatID, messages.InvalidGoal)
		return
	}

	var difficulty *string
	if len(parts) == 3 {
		value := strings.TrimSpace(parts[2])
		difficulty = &value
	}

	if err := h.userManager.SetGoal(ctx, user.UserID, goal, difficulty); err != nil {
		if errors.Is(err, service.ErrInvalidDifficulty) {
			reply.Text(ctx, b, chatID, messages.InvalidGoalDifficulty)
			return
		}
		h.logger.Error("failed to set goal from command", "err", err, "userID", user.UserID, "goal", goal)
		reply.Text(ctx, b, chatID, messages.InternalError)
		return
	}

	updatedUser, err := h.userManager.GetUser(ctx, user.UserID)
	if err != nil {
		h.logger.Error("failed to get user after goal command", "err", err, "userID", user.UserID)
		reply.Text(ctx, b, chatID, messages.InternalError)
		return
	}

	goalText, err := h.userManager.BuildGoalMessage(ctx, updatedUser)
	if err != nil {
		h.logger.Error("failed to build progress message after goal command", "err", err, "userID", user.UserID)
		reply.Text(ctx, b, chatID, messages.InternalError)
		return
	}

	reply.HTML(ctx, b, chatID, goalText)
}
