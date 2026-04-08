package add

import (
	"errors"
	"fmt"
	"html"
	"time"

	"github.com/aseptimu/AlgoTrack/internal/model"
	"github.com/aseptimu/AlgoTrack/internal/service"
	"github.com/aseptimu/AlgoTrack/internal/telegram/format"
	"github.com/aseptimu/AlgoTrack/internal/timezone"
)

func taskErrorText(err error) string {
	if errors.Is(err, service.ErrFailedUserCreate) {
		return "Не смог сохранить пользователя 😔 Попробуй позже."
	} else if errors.Is(err, service.ErrTgUserNotFound) {
		return "Пользователь не найден 😔"
	} else {
		return "Ошибка 😔 Попробуй позже."
	}
}

func buildAddSuccessMessage(result *model.AddTaskResult) string {
	task := result.Task
	stats := result.Stats

	title := "Untitled problem"
	if task.Title != nil && *task.Title != "" {
		title = html.EscapeString(*task.Title)
	}

	taskLine := fmt.Sprintf("<b>%d. %s</b>", task.TaskNumber, title)
	if task.Link != "" {
		taskLine = fmt.Sprintf(`<b>%d. <a href="%s">%s</a></b>`, task.TaskNumber, html.EscapeString(task.Link), title)
	}

	goalBlock := ""
	if result.GoalProgress != nil && len(result.GoalProgress.Items) > 0 {
		goalBlock = "\n\n<b>Цели</b>\n" + format.GoalLines(result.GoalProgress.Items)
	}

	statusTitle := "✅ <b>Задача сохранена</b>"
	if result.IsReview {
		statusTitle = "🔁 <b>Повторение засчитано</b>"
	}

	reviewMeta := ""
	if task.LastReviewedAt != nil && task.NextReviewAt != nil {
		reviewMeta = fmt.Sprintf(
			"\n\n<b>Повторения</b>\nРешена раз: <b>%d</b>\nПоследний раз: <b>%s</b>\nСледующее повторение: <b>%s</b>",
			task.ReviewCount,
			formatMoscowTime(*task.LastReviewedAt),
			formatMoscowTime(*task.NextReviewAt),
		)
	}

	return fmt.Sprintf(
		"%s\n\n%s%s\n\n➖➖➖➖➖➖\n<b>Статистика</b>\nРешено всего: %d\nEasy: %d\nMedium: %d\nHard: %d%s",
		statusTitle,
		taskLine,
		reviewMeta,
		stats.Total,
		stats.Easy,
		stats.Medium,
		stats.Hard,
		goalBlock,
	)
}

func formatMoscowTime(value time.Time) string {
	return value.In(timezone.MoscowLocation).Format("02.01.2006 15:04 MSK")
}
