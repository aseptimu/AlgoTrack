package add

import (
	"context"
	"errors"
	"github.com/aseptimu/AlgoTrack/internal/service"
	"log/slog"
	"net/url"
	"strings"

	"github.com/aseptimu/AlgoTrack/internal/model"
	"github.com/aseptimu/AlgoTrack/internal/repo"
	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type TgUserService interface {
	Get(ctx context.Context, userID int64) (*model.User, error)
	Create(ctx context.Context, u *model.User) error
}

type TaskCreator interface {
	Create(ctx context.Context, task *model.Task) error
}

type Handler struct {
	users TgUserService
	tasks TaskCreator
	log   *slog.Logger
}

func New(users TgUserService, tasks TaskCreator, log *slog.Logger) *Handler {
	if log == nil {
		log = slog.Default()
	}
	return &Handler{users: users, tasks: tasks, log: log}
}

func (h *Handler) Handle(ctx context.Context, b *tgbot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.From == nil {
		return
	}

	h.log.Info("Add call")

	chatID := update.Message.Chat.ID
	userID := update.Message.From.ID
	username := update.Message.From.Username

	_, err := h.users.Get(ctx, userID)
	if errors.Is(err, repo.ErrTgUserNotFound) {
		if err := h.users.Create(ctx, &model.User{UserID: userID, ChatID: chatID, Username: username}); err != nil {
			h.log.Error("failed to create tg user", "err", err, "userID", userID, "chatID", chatID)
			_, _ = b.SendMessage(ctx, &tgbot.SendMessageParams{
				ChatID: chatID,
				Text:   "Не смог сохранить пользователя 😔 Попробуй позже.",
			})
			return
		}
	} else if err != nil {
		h.log.Error("failed to get tg user", "err", err, "userID", userID, "chatID", chatID)
		_, _ = b.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: chatID,
			Text:   "Ошибка 😔 Попробуй позже.",
		})
		return
	}

	link, ok := parseAddLink(update.Message.Text)
	if !ok {
		_, _ = b.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: chatID,
			Text:   "Используй так: /add <ссылка>\nНапример: /add https://leetcode.com/problems/two-sum/",
		})
		return
	}

	if !looksLikeURL(link) {
		_, _ = b.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: chatID,
			Text:   "Похоже, это не ссылка. Пример: /add https://leetcode.com/problems/two-sum/",
		})
		return
	}

	task := model.Task{
		UserID:      userID,
		Link:        link,
		Description: nil,
	}

	if err := h.tasks.Create(ctx, &task); err != nil {
		h.log.Error("failed to create task", "err", err, "userID", userID, "chatID", chatID)
		text := "Не смог сохранить задачу 😔 Попробуй позже."
		if errors.Is(err, service.ErrTaskAlreadyExists) {
			text = "👌 Эта задача уже добавлена. Используй /update для обновления прогресса"
		}
		_, _ = b.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: chatID,
			Text:   text,
		})
		return
	}

	_, _ = b.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: chatID,
		Text:   "✅ Задача сохранена",
	})
}

func parseAddLink(text string) (string, bool) {
	parts := strings.Fields(strings.TrimSpace(text))
	if len(parts) < 2 {
		return "", false
	}
	if !strings.HasPrefix(parts[0], "/add") {
		return "", false
	}
	return parts[1], true
}

func looksLikeURL(s string) bool {
	u, err := url.ParseRequestURI(s)
	return err == nil && u.Scheme != "" && u.Host != ""
}
