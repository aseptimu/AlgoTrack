package next

import (
	"context"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"strings"

	"github.com/aseptimu/AlgoTrack/internal/catalog"
	"github.com/aseptimu/AlgoTrack/internal/service"
	"github.com/aseptimu/AlgoTrack/internal/service/recommend"
	"github.com/aseptimu/AlgoTrack/internal/telegram/helpers"
	"github.com/aseptimu/AlgoTrack/internal/telegram/messages"
	"github.com/aseptimu/AlgoTrack/internal/telegram/reply"
	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Recommender is the surface the /next handler talks to. *recommend.Service
// implements this.
type Recommender interface {
	Next(ctx context.Context, userID int64, mode string) (*catalog.Problem, error)
}

type Handler struct {
	users       service.UserManager
	recommender Recommender
	logger      *slog.Logger
}

func New(users service.UserManager, r Recommender, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{users: users, recommender: r, logger: logger}
}

func (h *Handler) Handle(ctx context.Context, b *tgbot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.From == nil {
		return
	}
	chatID := update.Message.Chat.ID

	incoming := helpers.GetUser(update)
	user, err := h.users.EnsureExistsAndGet(ctx, incoming)
	if err != nil {
		h.logger.Error("next: ensure user", "err", err)
		reply.Text(ctx, b, chatID, messages.InternalError)
		return
	}

	// "/next" or "/next js" — the trailing arg is an ad-hoc mode override
	// that does NOT change the user's persistent recommend_mode.
	parts := strings.Fields(update.Message.Text)
	mode := user.RecommendMode
	if mode == "" {
		mode = "default"
	}
	if len(parts) >= 2 {
		switch strings.ToLower(parts[1]) {
		case "js", "default":
			mode = strings.ToLower(parts[1])
		}
	}

	problem, err := h.recommender.Next(ctx, user.UserID, mode)
	if err != nil {
		if errors.Is(err, recommend.ErrCatalogExhausted) {
			reply.Text(ctx, b, chatID, "🎉 Вы прошли все задачи из доступных списков. Поздравляю!")
			return
		}
		h.logger.Error("next: recommend", "err", err, "userID", user.UserID)
		reply.Text(ctx, b, chatID, messages.InternalError)
		return
	}

	msg := fmt.Sprintf(
		"📌 <b>Новая задача</b>\n\n<a href=\"%s\">#%d %s</a> [%s]\n<i>%s</i>\n\nКогда решишь — отметь через <code>/add %d</code>.",
		html.EscapeString(problem.Link()),
		problem.Number,
		html.EscapeString(problem.Title),
		html.EscapeString(problem.Difficulty),
		html.EscapeString(problem.Topic),
		problem.Number,
	)
	reply.HTML(ctx, b, chatID, msg)
}
