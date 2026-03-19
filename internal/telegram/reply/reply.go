package reply

import (
	"context"
	"log/slog"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func Text(ctx context.Context, b *tgbot.Bot, chatID int64, text string) {
	if _, err := b.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: text}); err != nil {
		slog.Error("failed to send telegram text message", "err", err, "chatID", chatID)
	}
}

func HTML(ctx context.Context, b *tgbot.Bot, chatID int64, text string) {
	if _, err := b.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:    chatID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
	}); err != nil {
		slog.Error("failed to send telegram html message", "err", err, "chatID", chatID)
	}
}
