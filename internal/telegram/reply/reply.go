package reply

import (
	"context"
	"log/slog"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// noPreview is a singleton link preview options pointer used by every reply
// helper in this package. We always disable web previews because every link
// the bot ever sends is a leetcode.com problem URL and the preview cards
// fill the screen with redundant noise.
var noPreview = func() *models.LinkPreviewOptions {
	disabled := true
	return &models.LinkPreviewOptions{IsDisabled: &disabled}
}()

// NoPreview exposes the same singleton for callers that build their own
// SendMessageParams (poller, daily reminder) so the whole bot is consistent.
func NoPreview() *models.LinkPreviewOptions { return noPreview }

func Text(ctx context.Context, b *tgbot.Bot, chatID int64, text string) {
	if _, err := b.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:             chatID,
		Text:               text,
		LinkPreviewOptions: noPreview,
	}); err != nil {
		slog.Error("failed to send telegram text message", "err", err, "chatID", chatID)
	}
}

func HTML(ctx context.Context, b *tgbot.Bot, chatID int64, text string) {
	if _, err := b.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:             chatID,
		Text:               text,
		ParseMode:          models.ParseModeHTML,
		LinkPreviewOptions: noPreview,
	}); err != nil {
		slog.Error("failed to send telegram html message", "err", err, "chatID", chatID)
	}
}
