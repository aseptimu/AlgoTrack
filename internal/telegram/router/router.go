package router

import (
	"github.com/aseptimu/AlgoTrack/internal/telegram"
	"github.com/aseptimu/AlgoTrack/internal/telegram/commands/add"
	"github.com/aseptimu/AlgoTrack/internal/telegram/commands/goal"
	helpcmd "github.com/aseptimu/AlgoTrack/internal/telegram/commands/help"
	"github.com/aseptimu/AlgoTrack/internal/telegram/commands/link"
	"github.com/aseptimu/AlgoTrack/internal/telegram/commands/setgoal"
	"github.com/aseptimu/AlgoTrack/internal/telegram/commands/start"
	"github.com/aseptimu/AlgoTrack/internal/telegram/messages/fallback"
	tgbot "github.com/go-telegram/bot"
)

type Handlers struct {
	Add          *add.Handler
	Start        *start.Handler
	Help         *helpcmd.Handler
	Text         *fallback.Handler
	GoalCallback *goal.Handler
	SetGoal      *setgoal.Handler
	Link         *link.Handler
}

func Register(b *tgbot.Bot, h Handlers) {
	b.RegisterHandler(tgbot.HandlerTypeMessageText, telegram.Add, tgbot.MatchTypePrefix, h.Add.Handle)
	b.RegisterHandler(tgbot.HandlerTypeMessageText, telegram.Start, tgbot.MatchTypeExact, h.Start.Handle)
	b.RegisterHandler(tgbot.HandlerTypeMessageText, telegram.Help, tgbot.MatchTypeExact, h.Help.Handle)
	b.RegisterHandler(tgbot.HandlerTypeMessageText, telegram.Goal, tgbot.MatchTypePrefix, h.SetGoal.Handle)
	b.RegisterHandler(tgbot.HandlerTypeMessageText, telegram.Link, tgbot.MatchTypePrefix, h.Link.Handle)

	b.RegisterHandler(tgbot.HandlerTypeMessageText, "", tgbot.MatchTypePrefix, h.Text.Handle)

	b.RegisterHandler(tgbot.HandlerTypeCallbackQueryData, "goal_", tgbot.MatchTypePrefix, h.GoalCallback.Handle)
}
