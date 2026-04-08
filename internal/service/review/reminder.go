package review

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"strings"
	"time"

	"github.com/aseptimu/AlgoTrack/internal/model"
	"github.com/aseptimu/AlgoTrack/internal/timezone"
	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type DueReviewProvider interface {
	GetDueReviews(ctx context.Context, nowTime time.Time) ([]model.DueReviewBatch, error)
}

type ReminderService struct {
	repo     DueReviewProvider
	bot      *tgbot.Bot
	logger   *slog.Logger
	location *time.Location
}

func NewReminderService(repo DueReviewProvider, bot *tgbot.Bot, logger *slog.Logger) *ReminderService {
	location := timezone.MoscowLocation

	if logger == nil {
		logger = slog.Default()
	}

	return &ReminderService{
		repo:     repo,
		bot:      bot,
		logger:   logger,
		location: location,
	}
}

func (r *ReminderService) Start(ctx context.Context) {
	for {
		runAt := r.nextRun(time.Now())
		timer := time.NewTimer(time.Until(runAt))

		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			r.sendDueReviews(ctx, runAt)
		}
	}
}

func (r *ReminderService) nextRun(now time.Time) time.Time {
	localNow := now.In(r.location)
	runAt := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 9, 0, 0, 0, r.location)
	if !localNow.Before(runAt) {
		runAt = runAt.AddDate(0, 0, 1)
	}

	return runAt
}

func (r *ReminderService) sendDueReviews(ctx context.Context, runAt time.Time) {
	batches, err := r.repo.GetDueReviews(ctx, runAt.UTC())
	if err != nil {
		r.logger.Error("failed to load due reviews", "err", err)
		return
	}

	for _, batch := range batches {
		message := buildReminderMessage(batch.Tasks, r.location)
		if _, err := r.bot.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID:    batch.ChatID,
			Text:      message,
			ParseMode: models.ParseModeHTML,
		}); err != nil {
			r.logger.Error("failed to send due review reminder", "err", err, "userID", batch.UserID, "chatID", batch.ChatID)
		}
	}
}

func buildReminderMessage(tasks []model.DueReviewTask, location *time.Location) string {
	lines := make([]string, 0, len(tasks))
	for _, task := range tasks {
		title := html.EscapeString(task.Title)
		if title == "" {
			title = fmt.Sprintf("Task %d", task.TaskNumber)
		}

		taskLabel := fmt.Sprintf("<b>#%d</b> %s", task.TaskNumber, title)
		if task.Link != "" {
			taskLabel = fmt.Sprintf(`<b>#%d</b> <a href="%s">%s</a>`, task.TaskNumber, html.EscapeString(task.Link), title)
		}

		difficulty := task.Difficulty
		if difficulty == "" {
			difficulty = "Unknown"
		}

		lines = append(lines, fmt.Sprintf(
			"%s\n%s • repeats: <b>%d</b> • last: <i>%s</i>",
			taskLabel,
			html.EscapeString(difficulty),
			task.ReviewCount,
			task.LastReviewedAt.In(location).Format("02.01 15:04"),
		))
	}

	return "<b>Пора повторить задачи</b>\n\n" + strings.Join(lines, "\n\n") + "\n\nОтметь повторение через <code>/add number</code>."
}
