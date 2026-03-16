package start

import (
	"context"
	"github.com/aseptimu/AlgoTrack/internal/service"
	"github.com/aseptimu/AlgoTrack/internal/telegram/helpers"
	"github.com/aseptimu/AlgoTrack/internal/telegram/messages"
	"github.com/aseptimu/AlgoTrack/internal/telegram/reply"
	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"log/slog"
)

type Handler struct {
	userManager service.UserManager
	logger      *slog.Logger
}

func New(userManger service.UserManager, logger *slog.Logger) *Handler {
	return &Handler{userManager: userManger, logger: logger}
}

func (h *Handler) Handle(ctx context.Context, b *tgbot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	h.logger.Info("Received start command")

	chatID := update.Message.Chat.ID
	incomingUser := helpers.GetUser(update)

	user, err := h.userManager.EnsureExistsAndGet(ctx, incomingUser)
	if err != nil {
		reply.Text(ctx, b, chatID, messages.InternalError)
		return
	}

	text, err := h.userManager.BuildWelcomeMessage(ctx, user)
	if err != nil {
		reply.Text(ctx, b, chatID, messages.InternalError)
		return
	}

	message := &tgbot.SendMessageParams{
		ChatID: chatID,
		Text:   text,
	}

	if user.GoalTotal == nil || *user.GoalTotal <= 0 {
		message.ReplyMarkup = &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{Text: "300", CallbackData: "goal_300"},
					{Text: "500", CallbackData: "goal_500"},
					{Text: "700", CallbackData: "goal_700"},
				},
			},
		}
	}

	msg, err := b.SendMessage(ctx, message)
	if err != nil {
		h.logger.Error("failed to send message", "err", err)
		return
	}

	h.logger.Info("message sent", "message_id", msg.ID)
}
