package router

import (
	"github.com/aseptimu/AlgoTrack/internal/telegram"
	"github.com/aseptimu/AlgoTrack/internal/telegram/commands/add"
	"github.com/aseptimu/AlgoTrack/internal/telegram/commands/start"
	"github.com/aseptimu/AlgoTrack/internal/telegram/messages/fallback"
	tgbot "github.com/go-telegram/bot"
)

type Handlers struct {
	Add   *add.Handler
	Start *start.Handler
	Text  *fallback.Handler
}

func Register(b *tgbot.Bot, h Handlers) {
	b.RegisterHandler(tgbot.HandlerTypeMessageText, telegram.Add, tgbot.MatchTypePrefix, h.Add.Handle)
	b.RegisterHandler(tgbot.HandlerTypeMessageText, telegram.Start, tgbot.MatchTypeExact, h.Start.Handle)

	b.RegisterHandler(tgbot.HandlerTypeMessageText, "", tgbot.MatchTypePrefix, h.Text.Handle)
}
