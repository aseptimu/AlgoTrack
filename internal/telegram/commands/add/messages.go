package add

import (
	"errors"
	"fmt"
	"github.com/aseptimu/AlgoTrack/internal/model"
	"github.com/aseptimu/AlgoTrack/internal/service"
	"html"
	"strings"
)

func taskErrorText(err error) string {
	if errors.Is(err, service.ErrTaskAlreadyExists) {
		return "Данная задача уже добавлена. Используй /update"
	} else if errors.Is(err, service.ErrFailedUserCreate) {
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
		goalBlock = "\n\n<b>Цели</b>\n" + formatGoalLines(result.GoalProgress.Items)
	}

	return fmt.Sprintf(
		"✅ <b>Задача сохранена</b>\n\n%s\n\n➖➖➖➖➖➖\n<b>Статистика</b>\nРешено всего: %d\nEasy: %d\nMedium: %d\nHard: %d%s",
		taskLine,
		stats.Total,
		stats.Easy,
		stats.Medium,
		stats.Hard,
		goalBlock,
	)
}

func formatGoalLines(items []model.GoalProgress) string {
	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, fmt.Sprintf("%s <b>%d / %d</b> <i>(осталось %d)</i>", goalBadge(item.Label), item.Solved, item.Goal, item.Remaining))
	}

	return strings.Join(lines, "\n")
}

func goalBadge(label string) string {
	switch label {
	case "Total":
		return "🎯 <b>Total</b>"
	case "Easy":
		return "🟢 <b>Easy</b>"
	case "Medium":
		return "🟠 <b>Medium</b>"
	case "Hard":
		return "🔴 <b>Hard</b>"
	default:
		return "<b>" + label + "</b>"
	}
}
