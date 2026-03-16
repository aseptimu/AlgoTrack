package helpers

import (
	"github.com/aseptimu/AlgoTrack/internal/model"
	"github.com/go-telegram/bot/models"
)

func GetUser(update *models.Update) *model.User {
	return &model.User{
		UserID:   update.Message.From.ID,
		Username: update.Message.From.Username,
		ChatID:   update.Message.Chat.ID,
	}
}
