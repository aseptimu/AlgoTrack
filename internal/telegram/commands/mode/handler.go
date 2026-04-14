package mode

import (
	"context"
	"log/slog"
	"strings"

	"github.com/aseptimu/AlgoTrack/internal/service"
	"github.com/aseptimu/AlgoTrack/internal/telegram/helpers"
	"github.com/aseptimu/AlgoTrack/internal/telegram/messages"
	"github.com/aseptimu/AlgoTrack/internal/telegram/reply"
	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// ModeSetter persists a user's recommendation mode.
type ModeSetter interface {
	SetRecommendMode(ctx context.Context, userID int64, mode string) error
}

type Handler struct {
	users  service.UserManager
	repo   ModeSetter
	logger *slog.Logger
}

func New(users service.UserManager, repo ModeSetter, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{users: users, repo: repo, logger: logger}
}

func (h *Handler) Handle(ctx context.Context, b *tgbot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.From == nil {
		return
	}
	chatID := update.Message.Chat.ID

	incoming := helpers.GetUser(update)
	user, err := h.users.EnsureExistsAndGet(ctx, incoming)
	if err != nil {
		h.logger.Error("mode: ensure user", "err", err)
		reply.Text(ctx, b, chatID, messages.InternalError)
		return
	}

	parts := strings.Fields(update.Message.Text)
	if len(parts) != 2 {
		reply.Text(ctx, b, chatID, "Используй: /mode default | /mode js")
		return
	}

	mode := strings.ToLower(parts[1])
	if mode != "default" && mode != "js" {
		reply.Text(ctx, b, chatID, "Допустимые режимы: default, js")
		return
	}

	if err := h.repo.SetRecommendMode(ctx, user.UserID, mode); err != nil {
		h.logger.Error("mode: persist", "err", err, "userID", user.UserID)
		reply.Text(ctx, b, chatID, messages.InternalError)
		return
	}
	reply.Text(ctx, b, chatID, "Режим рекомендаций обновлён: "+mode)
}
