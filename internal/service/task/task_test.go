package task

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/aseptimu/AlgoTrack/internal/model"
	"github.com/aseptimu/AlgoTrack/internal/service"
)

// --- Mocks ---

type mockUserEnsurer struct {
	user *model.User
	err  error
}

func (m *mockUserEnsurer) EnsureExistsAndGet(_ context.Context, _ *model.User) (*model.User, error) {
	return m.user, m.err
}

type mockProblemProvider struct {
	problem *model.ProblemInfo
	err     error
}

func (m *mockProblemProvider) GetProblemByNumber(_ context.Context, _ int64) (*model.ProblemInfo, error) {
	return m.problem, m.err
}

type mockTaskRepo struct {
	createdTask    *model.Task
	createErr      error
	getTask        *model.Task
	getErr         error
	reviewTask     *model.Task
	reviewErr      error
	stats          *model.TaskStats
	statsErr       error
	dueReviews     []model.DueReviewBatch
	dueReviewsErr  error
}

func (m *mockTaskRepo) Create(_ context.Context, task *model.Task, _ int64) (*model.Task, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	if m.createdTask != nil {
		return m.createdTask, nil
	}
	return task, nil
}

func (m *mockTaskRepo) GetByTaskNumber(_ context.Context, _, _ int64) (*model.Task, error) {
	return m.getTask, m.getErr
}

func (m *mockTaskRepo) Review(_ context.Context, task *model.Task, _ int64) (*model.Task, error) {
	if m.reviewErr != nil {
		return nil, m.reviewErr
	}
	if m.reviewTask != nil {
		return m.reviewTask, nil
	}
	return task, nil
}

func (m *mockTaskRepo) GetStats(_ context.Context, _ int64) (*model.TaskStats, error) {
	return m.stats, m.statsErr
}

func (m *mockTaskRepo) GetDueReviews(_ context.Context, _ time.Time) ([]model.DueReviewBatch, error) {
	return m.dueReviews, m.dueReviewsErr
}

// --- Tests ---

func newTestService(users UserEnsurer, repo Repository, problems ProblemProvider) *TaskService {
	return NewTaskService(users, repo, problems, slog.Default())
}

func TestAdd_NewTask(t *testing.T) {
	userID := int64(123)
	user := &model.User{UserID: userID, ChatID: 456, Username: "testuser"}
	problem := &model.ProblemInfo{
		Number:     1,
		Title:      "Two Sum",
		TitleSlug:  "two-sum",
		Difficulty: "Easy",
		Link:       "https://leetcode.com/problems/two-sum/",
		Platform:   "leetcode",
	}
	stats := &model.TaskStats{Total: 1, Easy: 1}

	svc := newTestService(
		&mockUserEnsurer{user: user},
		&mockTaskRepo{
			getErr: service.ErrTaskNotFound,
			stats:  stats,
		},
		&mockProblemProvider{problem: problem},
	)

	result, err := svc.Add(context.Background(), 1, user)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsReview {
		t.Error("expected IsReview=false for new task")
	}
	if result.Task.TaskNumber != 1 {
		t.Errorf("task number = %d, want 1", result.Task.TaskNumber)
	}
	if result.Stats.Total != 1 {
		t.Errorf("stats total = %d, want 1", result.Stats.Total)
	}
}

func TestAdd_ReviewExistingTask(t *testing.T) {
	userID := int64(123)
	user := &model.User{UserID: userID, ChatID: 456}
	now := time.Now()
	existingTask := &model.Task{
		TaskNumber:     1,
		ReviewCount:    1,
		LastReviewedAt: &now,
	}
	problem := &model.ProblemInfo{
		Number:     1,
		Title:      "Two Sum",
		TitleSlug:  "two-sum",
		Difficulty: "Easy",
		Link:       "https://leetcode.com/problems/two-sum/",
	}
	stats := &model.TaskStats{Total: 1, Easy: 1}

	svc := newTestService(
		&mockUserEnsurer{user: user},
		&mockTaskRepo{
			getTask: existingTask,
			getErr:  nil,
			stats:   stats,
		},
		&mockProblemProvider{problem: problem},
	)

	result, err := svc.Add(context.Background(), 1, user)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsReview {
		t.Error("expected IsReview=true for existing task")
	}
	if result.Task.ReviewCount != 2 {
		t.Errorf("review count = %d, want 2", result.Task.ReviewCount)
	}
}

func TestAdd_UserEnsurerError(t *testing.T) {
	svc := newTestService(
		&mockUserEnsurer{err: errors.New("db error")},
		&mockTaskRepo{},
		&mockProblemProvider{},
	)

	_, err := svc.Add(context.Background(), 1, &model.User{UserID: 1})
	if err == nil {
		t.Error("expected error when user ensurer fails")
	}
}

func TestAdd_ProblemProviderError(t *testing.T) {
	user := &model.User{UserID: 1, ChatID: 2}

	svc := newTestService(
		&mockUserEnsurer{user: user},
		&mockTaskRepo{},
		&mockProblemProvider{err: errors.New("leetcode down")},
	)

	_, err := svc.Add(context.Background(), 1, user)
	if err == nil {
		t.Error("expected error when problem provider fails")
	}
}

func TestAdd_CreateError(t *testing.T) {
	user := &model.User{UserID: 1, ChatID: 2}
	problem := &model.ProblemInfo{Number: 1, Title: "X", TitleSlug: "x", Difficulty: "Easy", Link: "l"}

	svc := newTestService(
		&mockUserEnsurer{user: user},
		&mockTaskRepo{
			getErr:    service.ErrTaskNotFound,
			createErr: errors.New("insert failed"),
		},
		&mockProblemProvider{problem: problem},
	)

	_, err := svc.Add(context.Background(), 1, user)
	if err == nil {
		t.Error("expected error when repo create fails")
	}
}

func TestAdd_StatsError(t *testing.T) {
	user := &model.User{UserID: 1, ChatID: 2}
	problem := &model.ProblemInfo{Number: 1, Title: "X", TitleSlug: "x", Difficulty: "Easy", Link: "l"}

	svc := newTestService(
		&mockUserEnsurer{user: user},
		&mockTaskRepo{
			getErr:   service.ErrTaskNotFound,
			statsErr: errors.New("stats error"),
		},
		&mockProblemProvider{problem: problem},
	)

	_, err := svc.Add(context.Background(), 1, user)
	if err == nil {
		t.Error("expected error when stats retrieval fails")
	}
}

func TestNextReviewAt(t *testing.T) {
	base := time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		reviewCount int64
		wantDays    int
	}{
		{1, 1},
		{2, 3},
		{3, 7},
		{4, 14},
		{5, 30},
		{6, 60},
		{7, 120},
		{8, 120}, // beyond array, uses last
		{100, 120},
	}

	for _, tt := range tests {
		result := nextReviewAt(tt.reviewCount, base)
		if result == nil {
			t.Fatalf("nextReviewAt(%d, ...) returned nil", tt.reviewCount)
		}
		// Check that the result is roughly tt.wantDays days later (within 1 day tolerance due to timezone)
		diff := result.Sub(base)
		daysDiff := int(diff.Hours() / 24)
		if daysDiff < tt.wantDays-1 || daysDiff > tt.wantDays+1 {
			t.Errorf("nextReviewAt(%d) diff = %d days, want ~%d days", tt.reviewCount, daysDiff, tt.wantDays)
		}
	}
}

func TestBuildGoalProgress(t *testing.T) {
	t.Run("no goals set", func(t *testing.T) {
		user := &model.User{UserID: 1}
		stats := &model.TaskStats{Total: 5}
		result := buildGoalProgress(user, stats)
		if result != nil {
			t.Error("expected nil when no goals set")
		}
	})

	t.Run("total goal set", func(t *testing.T) {
		goal := int64(100)
		user := &model.User{UserID: 1, GoalTotal: &goal}
		stats := &model.TaskStats{Total: 30}
		result := buildGoalProgress(user, stats)
		if result == nil {
			t.Fatal("expected non-nil progress")
		}
		if len(result.Items) != 1 {
			t.Fatalf("expected 1 item, got %d", len(result.Items))
		}
		if result.Items[0].Remaining != 70 {
			t.Errorf("remaining = %d, want 70", result.Items[0].Remaining)
		}
	})

	t.Run("solved exceeds goal", func(t *testing.T) {
		goal := int64(10)
		user := &model.User{UserID: 1, GoalTotal: &goal}
		stats := &model.TaskStats{Total: 15}
		result := buildGoalProgress(user, stats)
		if result == nil {
			t.Fatal("expected non-nil progress")
		}
		if result.Items[0].Remaining != 0 {
			t.Errorf("remaining = %d, want 0 when solved exceeds goal", result.Items[0].Remaining)
		}
	})

	t.Run("multiple goals", func(t *testing.T) {
		total := int64(100)
		easy := int64(50)
		user := &model.User{UserID: 1, GoalTotal: &total, GoalEasy: &easy}
		stats := &model.TaskStats{Total: 10, Easy: 5}
		result := buildGoalProgress(user, stats)
		if result == nil {
			t.Fatal("expected non-nil progress")
		}
		if len(result.Items) != 2 {
			t.Errorf("expected 2 items, got %d", len(result.Items))
		}
	})
}

func TestNewTaskService_NilLogger(t *testing.T) {
	svc := NewTaskService(
		&mockUserEnsurer{},
		&mockTaskRepo{},
		&mockProblemProvider{},
		nil,
	)
	if svc == nil {
		t.Error("expected non-nil service")
	}
	if svc.logger == nil {
		t.Error("expected default logger when nil passed")
	}
}
