package add

import (
	"strings"
	"testing"
	"time"

	"github.com/aseptimu/AlgoTrack/internal/model"
	"github.com/aseptimu/AlgoTrack/internal/service"
)

func TestTaskErrorText(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		contains string
	}{
		{
			name:     "failed user create",
			err:      service.ErrFailedUserCreate,
			contains: "сохранить пользователя",
		},
		{
			name:     "user not found",
			err:      service.ErrTgUserNotFound,
			contains: "не найден",
		},
		{
			name:     "generic error",
			err:      service.ErrTaskNotFound,
			contains: "Ошибка",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := taskErrorText(tt.err)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("taskErrorText(%v) = %q, want to contain %q", tt.err, result, tt.contains)
			}
		})
	}
}

func TestBuildAddSuccessMessage(t *testing.T) {
	title := "Two Sum"
	difficulty := "Easy"
	now := time.Now()
	later := now.Add(24 * time.Hour)

	t.Run("new task", func(t *testing.T) {
		result := &model.AddTaskResult{
			Task: model.Task{
				TaskNumber:     1,
				Title:          &title,
				Difficulty:     &difficulty,
				Link:           "https://leetcode.com/problems/two-sum/",
				ReviewCount:    1,
				LastReviewedAt: &now,
				NextReviewAt:   &later,
			},
			Stats: model.TaskStats{
				Total:  10,
				Easy:   5,
				Medium: 3,
				Hard:   2,
			},
			IsReview: false,
		}

		msg := buildAddSuccessMessage(result)

		if !strings.Contains(msg, "Задача сохранена") {
			t.Error("expected new task message to contain 'Задача сохранена'")
		}
		if !strings.Contains(msg, "Two Sum") {
			t.Error("expected message to contain task title")
		}
		if !strings.Contains(msg, "10") {
			t.Error("expected message to contain total stats")
		}
	})

	t.Run("review task", func(t *testing.T) {
		result := &model.AddTaskResult{
			Task: model.Task{
				TaskNumber:     1,
				Title:          &title,
				Difficulty:     &difficulty,
				Link:           "https://leetcode.com/problems/two-sum/",
				ReviewCount:    2,
				LastReviewedAt: &now,
				NextReviewAt:   &later,
			},
			Stats:    model.TaskStats{Total: 10},
			IsReview: true,
		}

		msg := buildAddSuccessMessage(result)
		if !strings.Contains(msg, "Повторение засчитано") {
			t.Error("expected review message to contain 'Повторение засчитано'")
		}
	})

	t.Run("with goal progress", func(t *testing.T) {
		result := &model.AddTaskResult{
			Task: model.Task{
				TaskNumber:     1,
				Title:          &title,
				ReviewCount:    1,
				LastReviewedAt: &now,
				NextReviewAt:   &later,
			},
			Stats: model.TaskStats{Total: 5},
			GoalProgress: &model.UserProgress{
				Items: []model.GoalProgress{
					{Label: "Total", Solved: 5, Goal: 100, Remaining: 95},
				},
			},
		}

		msg := buildAddSuccessMessage(result)
		if !strings.Contains(msg, "Цели") {
			t.Error("expected message to contain goal section")
		}
		if !strings.Contains(msg, "5 / 100") {
			t.Error("expected message to contain goal progress")
		}
	})

	t.Run("nil title uses Untitled", func(t *testing.T) {
		result := &model.AddTaskResult{
			Task: model.Task{
				TaskNumber:     99,
				ReviewCount:    1,
				LastReviewedAt: &now,
				NextReviewAt:   &later,
			},
			Stats: model.TaskStats{Total: 1},
		}

		msg := buildAddSuccessMessage(result)
		if !strings.Contains(msg, "Untitled problem") {
			t.Error("expected message to contain 'Untitled problem' for nil title")
		}
	})
}

func TestFormatMoscowTime(t *testing.T) {
	loc := time.FixedZone("MSK", 3*60*60)
	input := time.Date(2026, 4, 8, 6, 0, 0, 0, time.UTC) // 09:00 MSK
	result := formatMoscowTime(input)

	if !strings.Contains(result, "08.04.2026 09:00 MSK") {
		t.Errorf("formatMoscowTime returned %q, want to contain '08.04.2026 09:00 MSK'", result)
	}

	_ = loc
}
