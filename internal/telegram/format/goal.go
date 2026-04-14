package format

import (
	"fmt"
	"strings"

	"github.com/aseptimu/AlgoTrack/internal/model"
)

// GoalBadge returns an HTML-formatted badge for a goal label.
func GoalBadge(label string) string {
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

// GoalLines formats goal progress items into HTML lines (Russian locale).
func GoalLines(items []model.GoalProgress) string {
	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, fmt.Sprintf(
			"%s <b>%d / %d</b> <i>(осталось %d)</i>",
			GoalBadge(item.Label), item.Solved, item.Goal, item.Remaining,
		))
	}
	return strings.Join(lines, "\n")
}

