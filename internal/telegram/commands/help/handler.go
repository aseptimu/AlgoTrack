package help

import (
	"context"
	"github.com/aseptimu/AlgoTrack/internal/telegram/messages"
	"github.com/aseptimu/AlgoTrack/internal/telegram/reply"
	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"log/slog"
)

type Handler struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *Handler {
	return &Handler{logger: logger}
}

func (h *Handler) Handle(ctx context.Context, b *tgbot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	h.logger.Info("Received help command")
	reply.HTML(ctx, b, update.Message.Chat.ID, messages.Help)
}
