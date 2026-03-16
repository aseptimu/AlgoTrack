package setgoal

import (
	"context"
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
	if len(parts) != 2 {
		reply.Text(ctx, b, chatID, messages.GoalUsage)
		return
	}

	goal, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || goal <= 0 {
		reply.Text(ctx, b, chatID, messages.InvalidGoal)
		return
	}

	if err := h.userManager.SetGoal(ctx, user.UserID, goal); err != nil {
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

	welcomeText, err := h.userManager.BuildWelcomeMessage(ctx, updatedUser)
	if err != nil {
		h.logger.Error("failed to build progress message after goal command", "err", err, "userID", user.UserID)
		reply.Text(ctx, b, chatID, messages.InternalError)
		return
	}

	reply.Text(ctx, b, chatID, welcomeText)
}
