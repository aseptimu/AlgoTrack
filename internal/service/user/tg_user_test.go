package user

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/aseptimu/AlgoTrack/internal/model"
	"github.com/aseptimu/AlgoTrack/internal/service"
)

// --- Mocks ---

type mockTgUserRepo struct {
	user      *model.User
	getErr    error
	createID  int64
	createErr error
	updateErr error
}

func (m *mockTgUserRepo) Create(_ context.Context, _ *model.User) (*int64, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	return &m.createID, nil
}

func (m *mockTgUserRepo) Get(_ context.Context, _ int64) (*model.User, error) {
	return m.user, m.getErr
}

func (m *mockTgUserRepo) UpdateGoal(_ context.Context, _ int64, _ int64, _ *string) error {
	return m.updateErr
}

type mockProgressProvider struct {
	stats *model.TaskStats
	err   error
}

func (m *mockProgressProvider) GetStats(_ context.Context, _ int64) (*model.TaskStats, error) {
	return m.stats, m.err
}

// --- Tests ---

func TestEnsureExistsAndGet_ExistingUser(t *testing.T) {
	user := &model.User{UserID: 1, ChatID: 100, Username: "test"}
	svc := NewUserService(
		&mockTgUserRepo{user: user},
		&mockProgressProvider{},
		slog.Default(),
	)

	result, err := svc.EnsureExistsAndGet(context.Background(), &model.User{UserID: 1, ChatID: 100})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.UserID != 1 {
		t.Errorf("UserID = %d, want 1", result.UserID)
	}
}

func TestEnsureExistsAndGet_CreatesNewUser(t *testing.T) {
	user := &model.User{UserID: 1, ChatID: 100, Username: "test"}
	callCount := 0
	repo := &mockTgUserRepo{
		createID: 1,
	}
	// First call returns not found, second returns user
	repo.getErr = service.ErrTgUserNotFound
	repo.user = nil

	svc := NewUserService(repo, &mockProgressProvider{}, slog.Default())

	// Override Get to return user on second call
	originalRepo := &sequentialGetRepo{
		createID: 1,
		getResults: []getResult{
			{nil, service.ErrTgUserNotFound},
			{user, nil},
		},
	}
	svc.repo = originalRepo

	result, err := svc.EnsureExistsAndGet(context.Background(), &model.User{UserID: 1, ChatID: 100})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.UserID != 1 {
		t.Errorf("UserID = %d, want 1", result.UserID)
	}
	_ = callCount
}

type getResult struct {
	user *model.User
	err  error
}

type sequentialGetRepo struct {
	createID   int64
	createErr  error
	updateErr  error
	getResults []getResult
	getCalls   int
}

func (r *sequentialGetRepo) Create(_ context.Context, _ *model.User) (*int64, error) {
	if r.createErr != nil {
		return nil, r.createErr
	}
	return &r.createID, nil
}

func (r *sequentialGetRepo) Get(_ context.Context, _ int64) (*model.User, error) {
	if r.getCalls >= len(r.getResults) {
		return nil, errors.New("no more get results")
	}
	result := r.getResults[r.getCalls]
	r.getCalls++
	return result.user, result.err
}

func (r *sequentialGetRepo) UpdateGoal(_ context.Context, _ int64, _ int64, _ *string) error {
	return r.updateErr
}

func TestSetGoal_Valid(t *testing.T) {
	svc := NewUserService(
		&mockTgUserRepo{},
		&mockProgressProvider{},
		slog.Default(),
	)

	err := svc.SetGoal(context.Background(), 1, 100, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetGoal_InvalidGoal(t *testing.T) {
	svc := NewUserService(
		&mockTgUserRepo{},
		&mockProgressProvider{},
		slog.Default(),
	)

	err := svc.SetGoal(context.Background(), 1, 0, nil)
	if !errors.Is(err, service.ErrInvalidGoal) {
		t.Errorf("expected ErrInvalidGoal, got %v", err)
	}

	err = svc.SetGoal(context.Background(), 1, -5, nil)
	if !errors.Is(err, service.ErrInvalidGoal) {
		t.Errorf("expected ErrInvalidGoal for negative, got %v", err)
	}
}

func TestSetGoal_InvalidDifficulty(t *testing.T) {
	svc := NewUserService(
		&mockTgUserRepo{},
		&mockProgressProvider{},
		slog.Default(),
	)

	diff := "impossible"
	err := svc.SetGoal(context.Background(), 1, 100, &diff)
	if !errors.Is(err, service.ErrInvalidDifficulty) {
		t.Errorf("expected ErrInvalidDifficulty, got %v", err)
	}
}

func TestSetGoal_WithDifficulty(t *testing.T) {
	svc := NewUserService(
		&mockTgUserRepo{},
		&mockProgressProvider{},
		slog.Default(),
	)

	for _, d := range []string{"easy", "Easy", "EASY", "medium", "Medium", "MEDIUM", "hard", "Hard", "HARD"} {
		diff := d
		err := svc.SetGoal(context.Background(), 1, 100, &diff)
		if err != nil {
			t.Errorf("SetGoal with difficulty %q failed: %v", d, err)
		}
	}
}

func TestNormalizeDifficulty(t *testing.T) {
	tests := []struct {
		input *string
		want  *string
		err   error
	}{
		{nil, nil, nil},
		{strPtr(""), nil, nil},
		{strPtr("easy"), strPtr("Easy"), nil},
		{strPtr("Easy"), strPtr("Easy"), nil},
		{strPtr("EASY"), strPtr("Easy"), nil},
		{strPtr("medium"), strPtr("Medium"), nil},
		{strPtr("hard"), strPtr("Hard"), nil},
		{strPtr("unknown"), nil, service.ErrInvalidDifficulty},
	}

	for _, tt := range tests {
		label := "nil"
		if tt.input != nil {
			label = *tt.input
		}
		t.Run(label, func(t *testing.T) {
			result, err := normalizeDifficulty(tt.input)
			if !errors.Is(err, tt.err) {
				t.Errorf("normalizeDifficulty(%v) err = %v, want %v", tt.input, err, tt.err)
			}
			if tt.want == nil && result != nil {
				t.Errorf("expected nil result, got %q", *result)
			}
			if tt.want != nil {
				if result == nil {
					t.Errorf("expected %q, got nil", *tt.want)
				} else if *result != *tt.want {
					t.Errorf("normalizeDifficulty(%q) = %q, want %q", *tt.input, *result, *tt.want)
				}
			}
		})
	}
}

func TestGetProgress_NoGoals(t *testing.T) {
	svc := NewUserService(
		&mockTgUserRepo{},
		&mockProgressProvider{stats: &model.TaskStats{Total: 5}},
		slog.Default(),
	)

	result, err := svc.GetProgress(context.Background(), &model.User{UserID: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Error("expected nil progress when no goals set")
	}
}

func TestGetProgress_WithGoals(t *testing.T) {
	goal := int64(100)
	svc := NewUserService(
		&mockTgUserRepo{},
		&mockProgressProvider{stats: &model.TaskStats{Total: 30, Easy: 10}},
		slog.Default(),
	)

	user := &model.User{UserID: 1, GoalTotal: &goal}
	result, err := svc.GetProgress(context.Background(), user)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil progress")
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
	if result.Items[0].Remaining != 70 {
		t.Errorf("remaining = %d, want 70", result.Items[0].Remaining)
	}
}

func TestHasAnyGoal(t *testing.T) {
	t.Run("no goals", func(t *testing.T) {
		if hasAnyGoal(&model.User{}) {
			t.Error("expected false for user with no goals")
		}
	})

	t.Run("zero goal", func(t *testing.T) {
		zero := int64(0)
		if hasAnyGoal(&model.User{GoalTotal: &zero}) {
			t.Error("expected false for zero goal")
		}
	})

	t.Run("positive goal", func(t *testing.T) {
		goal := int64(100)
		if !hasAnyGoal(&model.User{GoalTotal: &goal}) {
			t.Error("expected true for positive goal")
		}
	})
}

func TestFormatGoalProgress(t *testing.T) {
	items := []model.GoalProgress{
		{Label: "Total", Solved: 5, Goal: 100, Remaining: 95},
		{Label: "Easy", Solved: 3, Goal: 50, Remaining: 47},
	}
	result := formatGoalProgress(items)
	if result == "" {
		t.Error("expected non-empty formatted progress")
	}
}

func TestGoalBadge(t *testing.T) {
	tests := []struct {
		label    string
		contains string
	}{
		{"Total", "Total"},
		{"Easy", "Easy"},
		{"Medium", "Medium"},
		{"Hard", "Hard"},
		{"Custom", "Custom"},
	}

	for _, tt := range tests {
		result := goalBadge(tt.label)
		if result == "" {
			t.Errorf("goalBadge(%q) returned empty", tt.label)
		}
	}
}

func strPtr(s string) *string { return &s }
