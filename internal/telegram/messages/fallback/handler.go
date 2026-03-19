package fallback

import (
	"context"
	"github.com/aseptimu/AlgoTrack/internal/telegram/reply"
	"log/slog"
	"strings"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type Handler struct {
	log *slog.Logger
}

func New(log *slog.Logger) *Handler {
	if log == nil {
		log = slog.Default()
	}
	return &Handler{log: log}
}

func (h *Handler) Handle(ctx context.Context, b *tgbot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	chatID := update.Message.Chat.ID
	text := strings.TrimSpace(update.Message.Text)
	if text == "" {
		return
	}

	reply.Text(ctx, b, chatID, "Я пока понимаю только команды 🙂 Нажми /start")
}
