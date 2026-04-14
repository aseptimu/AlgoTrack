package review

import (
	"context"
	"log/slog"
	"time"

	"github.com/aseptimu/AlgoTrack/internal/model"
	"github.com/aseptimu/AlgoTrack/internal/service"
	reviewsvc "github.com/aseptimu/AlgoTrack/internal/service/review"
	"github.com/aseptimu/AlgoTrack/internal/telegram/helpers"
	"github.com/aseptimu/AlgoTrack/internal/telegram/messages"
	"github.com/aseptimu/AlgoTrack/internal/telegram/reply"
	"github.com/aseptimu/AlgoTrack/internal/timezone"
	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// DueReviewSource fetches due review tasks for one user.
type DueReviewSource interface {
	GetDueReviewsForUser(ctx context.Context, userID int64, asOf time.Time) ([]model.DueReviewTask, error)
}

type Handler struct {
	users   service.UserManager
	reviews DueReviewSource
	logger  *slog.Logger
}

func New(users service.UserManager, reviews DueReviewSource, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{users: users, reviews: reviews, logger: logger}
}

func (h *Handler) Handle(ctx context.Context, b *tgbot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.From == nil {
		return
	}
	chatID := update.Message.Chat.ID

	incoming := helpers.GetUser(update)
	user, err := h.users.EnsureExistsAndGet(ctx, incoming)
	if err != nil {
		h.logger.Error("review: ensure user", "err", err)
		reply.Text(ctx, b, chatID, messages.InternalError)
		return
	}

	due, err := h.reviews.GetDueReviewsForUser(ctx, user.UserID, time.Now().UTC())
	if err != nil {
		h.logger.Error("review: load due", "err", err, "userID", user.UserID)
		reply.Text(ctx, b, chatID, messages.InternalError)
		return
	}

	capped := reviewsvc.CapByDifficulty(due)
	if len(capped) == 0 {
		reply.Text(ctx, b, chatID, "✅ Нечего повторять — иди реши новую задачу через /next.")
		return
	}

	bundle := reviewsvc.MessageBundle{Reviews: capped}
	reply.HTML(ctx, b, chatID, reviewsvc.FormatBundle(bundle, timezone.MoscowLocation))
}
