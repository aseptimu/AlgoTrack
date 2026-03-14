package reply

import (
	"context"
	tgbot "github.com/go-telegram/bot"
)

func Text(ctx context.Context, b *tgbot.Bot, chatID int64, text string) {
	_, _ = b.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: text})
}
