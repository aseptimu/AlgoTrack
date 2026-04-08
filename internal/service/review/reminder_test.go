package review

import (
	"testing"
	"time"

	"github.com/aseptimu/AlgoTrack/internal/model"
)

func TestBuildReminderMessage(t *testing.T) {
	loc := time.FixedZone("MSK", 3*60*60)
	tasks := []model.DueReviewTask{
		{
			TaskNumber:     1,
			Title:          "Two Sum",
			Link:           "https://leetcode.com/problems/two-sum/",
			Difficulty:     "Easy",
			ReviewCount:    2,
			LastReviewedAt: time.Date(2026, 4, 7, 9, 0, 0, 0, time.UTC),
			NextReviewAt:   time.Date(2026, 4, 8, 6, 0, 0, 0, time.UTC),
		},
		{
			TaskNumber:     42,
			Title:          "",
			Link:           "",
			Difficulty:     "",
			ReviewCount:    1,
			LastReviewedAt: time.Date(2026, 4, 6, 9, 0, 0, 0, time.UTC),
			NextReviewAt:   time.Date(2026, 4, 8, 6, 0, 0, 0, time.UTC),
		},
	}

	msg := buildReminderMessage(tasks, loc)

	if msg == "" {
		t.Fatal("expected non-empty reminder message")
	}

	// Check that it contains task references
	checks := []string{
		"Two Sum",
		"#1",
		"#42",
		"Task 42",
		"/add number",
		"Пора повторить",
	}
	for _, check := range checks {
		found := false
		for i := range msg {
			if i+len(check) <= len(msg) && msg[i:i+len(check)] == check {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected reminder message to contain %q, got:\n%s", check, msg)
		}
	}
}

func TestNextRun(t *testing.T) {
	loc := time.FixedZone("MSK", 3*60*60)
	svc := &ReminderService{location: loc}

	t.Run("before 9am MSK", func(t *testing.T) {
		// 05:00 UTC = 08:00 MSK -> should schedule for today 09:00 MSK = 06:00 UTC
		now := time.Date(2026, 4, 8, 5, 0, 0, 0, time.UTC)
		next := svc.nextRun(now)
		expected := time.Date(2026, 4, 8, 6, 0, 0, 0, time.UTC)
		if !next.Equal(expected) {
			t.Errorf("nextRun before 9am: got %v, want %v", next, expected)
		}
	})

	t.Run("after 9am MSK", func(t *testing.T) {
		// 07:00 UTC = 10:00 MSK -> should schedule for tomorrow 09:00 MSK = 06:00 UTC
		now := time.Date(2026, 4, 8, 7, 0, 0, 0, time.UTC)
		next := svc.nextRun(now)
		expected := time.Date(2026, 4, 9, 6, 0, 0, 0, time.UTC)
		if !next.Equal(expected) {
			t.Errorf("nextRun after 9am: got %v, want %v", next, expected)
		}
	})
}
