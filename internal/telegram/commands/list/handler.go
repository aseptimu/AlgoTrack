package list

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/aseptimu/AlgoTrack/internal/model"
	"github.com/aseptimu/AlgoTrack/internal/telegram/reply"
	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const pageSize = 10

type TaskLister interface {
	List(ctx context.Context, userID int64, difficulty *string, offset, limit int64) ([]model.Task, int64, error)
}

type UserEnsurer interface {
	EnsureExistsAndGet(ctx context.Context, user *model.User) (*model.User, error)
}

type Handler struct {
	lister TaskLister
	users  UserEnsurer
	log    *slog.Logger
}

func New(lister TaskLister, users UserEnsurer, log *slog.Logger) *Handler {
	if log == nil {
		log = slog.Default()
	}
	return &Handler{lister: lister, users: users, log: log}
}

func (h *Handler) Handle(ctx context.Context, b *tgbot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.From == nil {
		return
	}

	h.log.Info("list command received")

	chatID := update.Message.Chat.ID
	userID := update.Message.From.ID
	username := update.Message.From.Username

	user := &model.User{
		UserID:   userID,
		ChatID:   chatID,
		Username: username,
	}

	ensuredUser, err := h.users.EnsureExistsAndGet(ctx, user)
	if err != nil {
		h.log.Error("failed to ensure user for list", "err", err, "userID", userID)
		reply.Text(ctx, b, chatID, "Something went wrong. Try again later.")
		return
	}

	difficulty := parseDifficultyFilter(update.Message.Text)

	h.sendPage(ctx, b, chatID, ensuredUser.UserID, difficulty, 0)
}

func (h *Handler) HandleCallback(ctx context.Context, b *tgbot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	data := update.CallbackQuery.Data
	userID := update.CallbackQuery.From.ID
	chatID := update.CallbackQuery.Message.Message.Chat.ID

	// Format: list_<offset> or list_<difficulty>_<offset>
	parts := strings.TrimPrefix(data, "list_")

	var difficulty *string
	var offset int64

	segments := strings.Split(parts, "_")
	if len(segments) == 1 {
		// list_<offset>
		offset, _ = strconv.ParseInt(segments[0], 10, 64)
	} else if len(segments) == 2 {
		// list_<difficulty>_<offset>
		d := segments[0]
		difficulty = &d
		offset, _ = strconv.ParseInt(segments[1], 10, 64)
	}

	_, _ = b.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})

	h.sendPage(ctx, b, chatID, userID, difficulty, offset)
}

func (h *Handler) sendPage(ctx context.Context, b *tgbot.Bot, chatID, userID int64, difficulty *string, offset int64) {
	tasks, total, err := h.lister.List(ctx, userID, difficulty, offset, pageSize)
	if err != nil {
		h.log.Error("failed to list tasks", "err", err, "userID", userID)
		reply.Text(ctx, b, chatID, "Something went wrong. Try again later.")
		return
	}

	if total == 0 {
		filterLabel := ""
		if difficulty != nil {
			filterLabel = fmt.Sprintf(" (%s)", *difficulty)
		}
		reply.Text(ctx, b, chatID, fmt.Sprintf("No solved problems found%s. Use /add to track your first one!", filterLabel))
		return
	}

	text := buildListMessage(tasks, difficulty, offset, total)
	keyboard := buildPaginationKeyboard(difficulty, offset, total)

	if _, err := b.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: keyboard,
	}); err != nil {
		h.log.Error("failed to send list message", "err", err, "chatID", chatID)
	}
}

func buildListMessage(tasks []model.Task, difficulty *string, offset, total int64) string {
	var sb strings.Builder

	filterLabel := ""
	if difficulty != nil && *difficulty != "" {
		filterLabel = fmt.Sprintf(" (%s)", *difficulty)
	}

	page := offset/pageSize + 1
	totalPages := (total + pageSize - 1) / pageSize
	sb.WriteString(fmt.Sprintf("<b>Solved problems%s</b> (page %d/%d)\n\n", filterLabel, page, totalPages))

	for i, task := range tasks {
		num := offset + int64(i) + 1

		title := "Untitled"
		if task.Title != nil && *task.Title != "" {
			title = html.EscapeString(*task.Title)
		}

		diffBadge := ""
		if task.Difficulty != nil {
			diffBadge = difficultyBadge(*task.Difficulty)
		}

		solvedAt := ""
		if task.LastReviewedAt != nil {
			solvedAt = formatMoscowDate(*task.LastReviewedAt)
		}

		taskLine := fmt.Sprintf("%d. %s", task.TaskNumber, title)
		if task.Link != "" {
			taskLine = fmt.Sprintf(`<a href="%s">%d. %s</a>`, html.EscapeString(task.Link), task.TaskNumber, title)
		}

		sb.WriteString(fmt.Sprintf("%d) %s %s\n   %s | Reviews: %d\n",
			num, taskLine, diffBadge, solvedAt, task.ReviewCount))
	}

	return sb.String()
}

func buildPaginationKeyboard(difficulty *string, offset, total int64) *models.InlineKeyboardMarkup {
	var buttons []models.InlineKeyboardButton

	if offset > 0 {
		prevOffset := offset - pageSize
		if prevOffset < 0 {
			prevOffset = 0
		}
		buttons = append(buttons, models.InlineKeyboardButton{
			Text:         "<< Prev",
			CallbackData: buildCallbackData(difficulty, prevOffset),
		})
	}

	if offset+pageSize < total {
		nextOffset := offset + pageSize
		buttons = append(buttons, models.InlineKeyboardButton{
			Text:         "Next >>",
			CallbackData: buildCallbackData(difficulty, nextOffset),
		})
	}

	if len(buttons) == 0 {
		return nil
	}

	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{buttons},
	}
}

func buildCallbackData(difficulty *string, offset int64) string {
	if difficulty != nil && *difficulty != "" {
		return fmt.Sprintf("list_%s_%d", *difficulty, offset)
	}
	return fmt.Sprintf("list_%d", offset)
}

func difficultyBadge(d string) string {
	switch d {
	case "Easy":
		return "[Easy]"
	case "Medium":
		return "[Medium]"
	case "Hard":
		return "[Hard]"
	default:
		return ""
	}
}

func formatMoscowDate(t time.Time) string {
	location := time.FixedZone("MSK", 3*60*60)
	return t.In(location).Format("02.01.2006")
}

func parseDifficultyFilter(text string) *string {
	parts := strings.Fields(text)
	if len(parts) < 2 {
		return nil
	}

	filter := strings.ToLower(parts[1])
	var difficulty string
	switch filter {
	case "easy":
		difficulty = "Easy"
	case "medium":
		difficulty = "Medium"
	case "hard":
		difficulty = "Hard"
	default:
		return nil
	}
	return &difficulty
}
