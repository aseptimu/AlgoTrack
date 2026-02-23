package telegram

import (
	"context"
	tgbot "github.com/go-telegram/bot"
	"log/slog"
)

type Bot struct {
	raw    *tgbot.Bot
	logger *slog.Logger
}

func New(token string, logger *slog.Logger, opts ...tgbot.Option) (*Bot, error) {
	bot := &Bot{logger: logger}

	if logger == nil {
		bot.logger = slog.Default()
	}

	baseOpts := []tgbot.Option{
		tgbot.WithErrorsHandler(func(err error) {
			bot.logger.Error("Telegram error", "err", err)
		}),
	}

	baseOpts = append(baseOpts, opts...)

	b, err := tgbot.New(token, baseOpts...)
	if err != nil {
		return nil, err
	}

	bot.raw = b

	return bot, nil
}

func (b *Bot) Run(ctx context.Context) {
	b.logger.Info("telegram bot started")
	b.raw.Start(ctx)
}

func (b *Bot) Raw() *tgbot.Bot { return b.raw }
