package start

import (
	"context"
	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type Handler struct {
}

func New() *Handler {
	return &Handler{}
}

func (h *Handler) Handle(ctx context.Context, b *tgbot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	chatID := update.Message.Chat.ID

	_, _ = b.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: chatID,
		Text: "👋 Привет!\n\n" +
			"Я бот для трекинга и повторения алгоритмических задач " +
			"(LeetCode, Яндекс, Codeforces и др.).\n\n" +
			"📌 Я помогу тебе:\n" +
			"• сохранять решённые задачи\n" +
			"• планировать повторения (пока недоступно)\n" +
			"• не забывать сложные темы (пока недоступно)\n\n" +
			"➕ Чтобы добавить задачу, введи команду:\n" +
			"/add <ссылка>\n\n" +
			"Например:\n" +
			"/add https://leetcode.com/problems/two-sum/",
	})
}
