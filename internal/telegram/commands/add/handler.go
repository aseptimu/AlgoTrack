package add

import (
	"context"
	"log/slog"

	"github.com/aseptimu/AlgoTrack/internal/model"
	"github.com/aseptimu/AlgoTrack/internal/telegram/reply"
	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type TaskAdder interface {
	Add(ctx context.Context, taskNumber int64, user *model.User) (*model.AddTaskResult, error)
}

type Handler struct {
	adder TaskAdder
	log   *slog.Logger
}

func New(adder TaskAdder, log *slog.Logger) *Handler {
	if log == nil {
		log = slog.Default()
	}
	return &Handler{adder: adder, log: log}
}

func (h *Handler) Handle(ctx context.Context, b *tgbot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.From == nil {
		return
	}

	h.log.Info("add command received")

	chatID := update.Message.Chat.ID
	userID := update.Message.From.ID
	username := update.Message.From.Username
	message := update.Message.Text

	user := &model.User{
		UserID:   userID,
		ChatID:   chatID,
		Username: username,
	}

	taskNumber, ok := parseAddTaskNumber(message)
	if !ok {
		reply.Text(ctx, b, chatID, "Введите команду в формате: /add 42")
		return
	}

	result, err := h.adder.Add(ctx, taskNumber, user)
	if err != nil {
		h.log.Error("failed to add task", "err", err, "userID", userID, "taskNumber", taskNumber)
		reply.Text(ctx, b, chatID, taskErrorText(err))
		return
	}

	reply.HTML(ctx, b, chatID, buildAddSuccessMessage(result))
}
