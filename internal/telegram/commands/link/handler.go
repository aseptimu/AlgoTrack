package link

import (
	"context"
	"log/slog"
	"regexp"
	"strings"

	"github.com/aseptimu/AlgoTrack/internal/service"
	"github.com/aseptimu/AlgoTrack/internal/telegram/helpers"
	"github.com/aseptimu/AlgoTrack/internal/telegram/messages"
	"github.com/aseptimu/AlgoTrack/internal/telegram/reply"
	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// validUsername matches LeetCode usernames: alphanumeric, hyphens, underscores, 1-39 chars.
var validUsername = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,39}$`)

type Handler struct {
	userManager service.UserManager
	logger      *slog.Logger
}

func New(userManager service.UserManager, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{userManager: userManager, logger: logger}
}

func (h *Handler) Handle(ctx context.Context, b *tgbot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.From == nil {
		return
	}

	chatID := update.Message.Chat.ID
	text := strings.TrimSpace(update.Message.Text)

	incomingUser := helpers.GetUser(update)
	user, err := h.userManager.EnsureExistsAndGet(ctx, incomingUser)
	if err != nil {
		h.logger.Error("failed to ensure user exists", "err", err, "userID", incomingUser.UserID)
		reply.Text(ctx, b, chatID, messages.InternalError)
		return
	}

	parts := strings.Fields(text)
	if len(parts) != 2 {
		reply.Text(ctx, b, chatID, "Используй команду так:\n/link <leetcode_username>\n\nНапример: /link johndoe")
		return
	}

	username := parts[1]
	if !validUsername.MatchString(username) {
		reply.Text(ctx, b, chatID, "Невалидный LeetCode username. Допустимы буквы, цифры, дефис и подчеркивание (до 39 символов).")
		return
	}

	if err := h.userManager.LinkLeetCode(ctx, user.UserID, username); err != nil {
		h.logger.Error("failed to link leetcode", "err", err, "userID", user.UserID, "username", username)
		reply.Text(ctx, b, chatID, messages.InternalError)
		return
	}

	reply.HTML(ctx, b, chatID, "LeetCode аккаунт <b>"+username+"</b> привязан.\n\nТеперь бот будет автоматически отслеживать твои accepted решения и добавлять их.")
}
