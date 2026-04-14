package review

import (
	"strings"
	"testing"
	"time"

	"github.com/aseptimu/AlgoTrack/internal/catalog"
	"github.com/aseptimu/AlgoTrack/internal/model"
)

func TestFormatBundle_RecommendationOnly(t *testing.T) {
	loc := time.FixedZone("MSK", 3*60*60)
	b := MessageBundle{
		Recommendation: &catalog.Problem{
			Number: 1, TitleSlug: "two-sum", Title: "Two Sum",
			Difficulty: "Easy", Topic: "Arrays & Hashing",
		},
	}
	msg := FormatBundle(b, loc)
	for _, want := range []string{"Доброе утро", "#1 Two Sum", "Easy", "Arrays &amp; Hashing"} {
		if !strings.Contains(msg, want) {
			t.Errorf("missing %q in:\n%s", want, msg)
		}
	}
	if strings.Contains(msg, "повторение") {
		t.Errorf("review block should be hidden when no due tasks; got:\n%s", msg)
	}
}

func TestFormatBundle_ReviewsOnly(t *testing.T) {
	loc := time.FixedZone("MSK", 3*60*60)
	b := MessageBundle{
		Reviews: []model.DueReviewTask{
			{TaskNumber: 1, Title: "Two Sum", Link: "https://leetcode.com/problems/two-sum/", Difficulty: "Easy",
				ReviewCount: 2, LastReviewedAt: time.Date(2026, 4, 7, 9, 0, 0, 0, time.UTC)},
		},
	}
	msg := FormatBundle(b, loc)
	for _, want := range []string{"На повторение", "#1 Two Sum", "/add номер"} {
		if !strings.Contains(msg, want) {
			t.Errorf("missing %q in:\n%s", want, msg)
		}
	}
	if strings.Contains(msg, "Новая задача") {
		t.Errorf("recommendation block should be hidden; got:\n%s", msg)
	}
}

func TestFormatBundle_BothBlocks(t *testing.T) {
	loc := time.FixedZone("MSK", 3*60*60)
	b := MessageBundle{
		Recommendation: &catalog.Problem{Number: 20, TitleSlug: "valid-parentheses", Title: "Valid Parentheses", Difficulty: "Easy", Topic: "Stack"},
		Reviews: []model.DueReviewTask{
			{TaskNumber: 1, Title: "Two Sum", Link: "x", Difficulty: "Easy", LastReviewedAt: time.Now()},
		},
	}
	msg := FormatBundle(b, loc)
	if !strings.Contains(msg, "Новая задача") || !strings.Contains(msg, "На повторение") {
		t.Errorf("expected both blocks; got:\n%s", msg)
	}
}

func TestFormatBundle_EmptyIsTriviallyShort(t *testing.T) {
	b := MessageBundle{}
	if !b.Empty() {
		t.Error("expected empty bundle to report Empty()=true")
	}
}

func TestNextRun(t *testing.T) {
	loc := time.FixedZone("MSK", 3*60*60)
	svc := &ReminderService{location: loc}

	t.Run("before 9am MSK", func(t *testing.T) {
		now := time.Date(2026, 4, 8, 5, 0, 0, 0, time.UTC) // 08:00 MSK
		next := svc.nextRun(now)
		expected := time.Date(2026, 4, 8, 6, 0, 0, 0, time.UTC) // 09:00 MSK same day
		if !next.Equal(expected) {
			t.Errorf("nextRun before 9am: got %v, want %v", next, expected)
		}
	})

	t.Run("after 9am MSK", func(t *testing.T) {
		now := time.Date(2026, 4, 8, 7, 0, 0, 0, time.UTC) // 10:00 MSK
		next := svc.nextRun(now)
		expected := time.Date(2026, 4, 9, 6, 0, 0, 0, time.UTC) // tomorrow 09:00 MSK
		if !next.Equal(expected) {
			t.Errorf("nextRun after 9am: got %v, want %v", next, expected)
		}
	})
}

// --- Cap tests ---

func mkTask(num int64, diff string, last time.Time) model.DueReviewTask {
	return model.DueReviewTask{TaskNumber: num, Difficulty: diff, LastReviewedAt: last}
}

func TestCapByDifficulty(t *testing.T) {
	t1 := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)

	t.Run("hard wins everything", func(t *testing.T) {
		got := CapByDifficulty([]model.DueReviewTask{
			mkTask(1, "Easy", t1),
			mkTask(2, "Medium", t2),
			mkTask(3, "Hard", t3),
		})
		if len(got) != 1 || got[0].Difficulty != "Hard" {
			t.Errorf("hard wins: %+v", got)
		}
	})

	t.Run("medium + easy = 2", func(t *testing.T) {
		got := CapByDifficulty([]model.DueReviewTask{
			mkTask(1, "Easy", t1),
			mkTask(2, "Easy", t2),
			mkTask(3, "Medium", t3),
		})
		if len(got) != 2 {
			t.Fatalf("medium + easy: want 2 got %d", len(got))
		}
		if got[0].Difficulty != "Medium" || got[1].Difficulty != "Easy" {
			t.Errorf("order/diff: %+v", got)
		}
		// Easy slot should be the OLDEST easy, not the newest.
		if got[1].TaskNumber != 1 {
			t.Errorf("expected oldest easy (#1), got #%d", got[1].TaskNumber)
		}
	})

	t.Run("medium only -> 1 medium (no easy filler)", func(t *testing.T) {
		got := CapByDifficulty([]model.DueReviewTask{
			mkTask(1, "Medium", t1),
			mkTask(2, "Medium", t2),
		})
		if len(got) != 1 || got[0].Difficulty != "Medium" || got[0].TaskNumber != 1 {
			t.Errorf("medium only: %+v", got)
		}
	})

	t.Run("easy only -> up to 3", func(t *testing.T) {
		got := CapByDifficulty([]model.DueReviewTask{
			mkTask(1, "Easy", t1),
			mkTask(2, "Easy", t2),
			mkTask(3, "Easy", t3),
			mkTask(4, "Easy", t3),
		})
		if len(got) != 3 {
			t.Fatalf("easy: want 3 got %d", len(got))
		}
		// Should be the three OLDEST in order.
		want := []int64{1, 2, 3}
		for i, w := range want {
			if got[i].TaskNumber != w {
				t.Errorf("easy[%d]: want #%d got #%d", i, w, got[i].TaskNumber)
			}
		}
	})

	t.Run("two easy -> two easy", func(t *testing.T) {
		got := CapByDifficulty([]model.DueReviewTask{
			mkTask(1, "Easy", t1),
			mkTask(2, "Easy", t2),
		})
		if len(got) != 2 {
			t.Errorf("expected 2, got %d", len(got))
		}
	})

	t.Run("empty input -> nil", func(t *testing.T) {
		if got := CapByDifficulty(nil); len(got) != 0 {
			t.Errorf("expected empty, got %+v", got)
		}
	})

	t.Run("unknown difficulty bucketed as easy", func(t *testing.T) {
		got := CapByDifficulty([]model.DueReviewTask{
			mkTask(1, "", t1),
		})
		if len(got) != 1 {
			t.Errorf("unknown difficulty should still surface; got %+v", got)
		}
	})
}
