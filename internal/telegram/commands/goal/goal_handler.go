package goal

import (
	"context"
	"strconv"
	"strings"

	"github.com/aseptimu/AlgoTrack/internal/service"
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
	return &Handler{userManager: userManager, logger: logger}
}

func (h *Handler) Handle(ctx context.Context, b *tgbot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	data := update.CallbackQuery.Data
	if !strings.HasPrefix(data, "goal_") {
		return
	}

	userID := update.CallbackQuery.From.ID
	chatID := update.CallbackQuery.Message.Message.Chat.ID

	switch data {
	case "goal_300", "goal_500", "goal_700":
	default:
		return
	}

	goalStr := strings.TrimPrefix(data, "goal_")
	goal, err := strconv.ParseInt(goalStr, 10, 64)
	if err != nil {
		h.logger.Error("failed to parse goal from callback", "err", err, "data", data)
		reply.Text(ctx, b, chatID, "Internal error")
		return
	}

	if err := h.userManager.SetGoal(ctx, userID, goal, nil); err != nil {
		h.logger.Error("failed to set goal", "err", err, "userID", userID, "goal", goal)
		reply.Text(ctx, b, chatID, "Internal error")
		return
	}

	user, err := h.userManager.GetUser(ctx, userID)
	if err != nil {
		h.logger.Error("failed to get user after goal update", "err", err, "userID", userID)
		reply.Text(ctx, b, chatID, "Internal error")
		return
	}

	text, err := h.userManager.BuildGoalMessage(ctx, user)
	if err != nil {
		h.logger.Error("failed to build welcome after goal update", "err", err, "userID", userID)
		reply.Text(ctx, b, chatID, "Internal error")
		return
	}

	_, _ = b.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
		Text:            "Goal saved",
	})

	reply.HTML(ctx, b, chatID, text)
}
