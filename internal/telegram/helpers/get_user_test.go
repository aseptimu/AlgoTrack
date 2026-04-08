package helpers

import (
	"testing"

	"github.com/go-telegram/bot/models"
)

func TestGetUser(t *testing.T) {
	update := &models.Update{
		Message: &models.Message{
			From: &models.User{
				ID:       123,
				Username: "testuser",
			},
			Chat: models.Chat{
				ID: 456,
			},
		},
	}

	user := GetUser(update)

	if user.UserID != 123 {
		t.Errorf("UserID = %d, want 123", user.UserID)
	}
	if user.Username != "testuser" {
		t.Errorf("Username = %q, want 'testuser'", user.Username)
	}
	if user.ChatID != 456 {
		t.Errorf("ChatID = %d, want 456", user.ChatID)
	}
}
