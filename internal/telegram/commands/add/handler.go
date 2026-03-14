package add

import (
	"context"
	"github.com/aseptimu/AlgoTrack/internal/model"
	"github.com/aseptimu/AlgoTrack/internal/telegram/reply"
	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"log/slog"
)

type TaskCreator interface {
	Add(ctx context.Context, task *model.Task, user *model.User) error
}

type Handler struct {
	creator TaskCreator
	log     *slog.Logger
}

func New(creator TaskCreator, log *slog.Logger) *Handler {
	if log == nil {
		log = slog.Default()
	}
	return &Handler{creator: creator, log: log}
}

func (h *Handler) Handle(ctx context.Context, b *tgbot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.From == nil {
		return
	}

	h.log.Info("Add call")

	chatID := update.Message.Chat.ID
	userID := update.Message.From.ID
	username := update.Message.From.Username
	message := update.Message.Text

	user := model.User{
		UserID:   userID,
		ChatID:   chatID,
		Username: username,
	}

	taskNumber, ok := parseAddTaskNumber(message)
	if !ok {
		reply.Text(ctx, b, chatID, "Введите /add число")
		return
	}

	task := model.Task{
		TaskNumber: taskNumber,
	}

	err := h.creator.Add(ctx, &task, &user)
	if err != nil {
		reply.Text(ctx, b, chatID, taskErrorText(err))
		return
	}

	reply.Text(ctx, b, chatID, "✅ Задача сохранена")
}
