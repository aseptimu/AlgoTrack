package task

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/aseptimu/AlgoTrack/internal/client"
	"github.com/aseptimu/AlgoTrack/internal/model"
	"github.com/aseptimu/AlgoTrack/internal/service"
)

type ProblemProvider interface {
	GetProblemByNumber(ctx context.Context, number int64) (*model.ProblemInfo, error)
}

type UserEnsurer interface {
	EnsureExistsAndGet(ctx context.Context, incomingUser *model.User) (*model.User, error)
}

type Repository interface {
	Create(ctx context.Context, task *model.Task, userID int64) (*model.Task, error)
	GetByTaskNumber(ctx context.Context, userID, taskNumber int64) (*model.Task, error)
	Review(ctx context.Context, task *model.Task, userID int64) (*model.Task, error)
	GetStats(ctx context.Context, userID int64) (*model.TaskStats, error)
	GetDueReviews(ctx context.Context, nowTime time.Time) ([]model.DueReviewBatch, error)
}

type TaskService struct {
	users    UserEnsurer
	repo     Repository
	problems ProblemProvider
	logger   *slog.Logger
}

// Expanding intervals keep early retrieval close and later reviews wider apart.
var reviewIntervals = []int{1, 3, 7, 14, 30, 60, 120}

func NewTaskService(
	users UserEnsurer,
	repo Repository,
	problems ProblemProvider,
	logger *slog.Logger,
) *TaskService {
	if logger == nil {
		logger = slog.Default()
	}

	return &TaskService{
		users:    users,
		repo:     repo,
		problems: problems,
		logger:   logger,
	}
}

func (t *TaskService) Add(ctx context.Context, taskNumber int64, incomingUser *model.User) (*model.AddTaskResult, error) {
	user, err := t.users.EnsureExistsAndGet(ctx, incomingUser)
	if err != nil {
		t.logger.Error("failed to ensure user existence while adding task", "err", err, "userID", incomingUser.UserID)
		return nil, err
	}

	problem, err := t.problems.GetProblemByNumber(ctx, taskNumber)
	if err != nil {
		// Graceful degradation: if LeetCode is unavailable, continue with minimal info.
		if errors.Is(err, client.ErrLeetCodeUnavailable) || errors.Is(err, context.DeadlineExceeded) {
			t.logger.Warn("leetcode unavailable, continuing without problem details", "err", err, "taskNumber", taskNumber)
			link := fmt.Sprintf("https://leetcode.com/problems/unknown-%d/", taskNumber)
			problem = &model.ProblemInfo{
				Number:   int(taskNumber),
				Title:    fmt.Sprintf("Problem %d", taskNumber),
				Link:     link,
				Platform: "leetcode",
			}
		} else {
			t.logger.Error("failed to get problem by number", "err", err, "taskNumber", taskNumber)
			return nil, err
		}
	}

	reviewedAt := time.Now().UTC()
	task := model.Task{
		UserID:         &user.UserID,
		TaskNumber:     taskNumber,
		Link:           problem.Link,
		Title:          &problem.Title,
		Difficulty:     &problem.Difficulty,
		ReviewCount:    1,
		LastReviewedAt: &reviewedAt,
		NextReviewAt:   nextReviewAt(1, reviewedAt),
	}

	storedTask, err := t.repo.GetByTaskNumber(ctx, user.UserID, taskNumber)
	isReview := false
	switch {
	case err == nil:
		isReview = true
		task.ReviewCount = storedTask.ReviewCount + 1
		task.NextReviewAt = nextReviewAt(task.ReviewCount, reviewedAt)
		storedTask, err = t.repo.Review(ctx, &task, user.UserID)
		if err != nil {
			t.logger.Error("failed to review task", "err", err, "userID", user.UserID, "taskNumber", taskNumber)
			return nil, err
		}
	case errors.Is(err, service.ErrTaskNotFound):
		storedTask, err = t.repo.Create(ctx, &task, user.UserID)
		if err != nil {
			t.logger.Error("failed to create task", "err", err, "userID", user.UserID, "taskNumber", taskNumber)
			return nil, err
		}
	default:
		t.logger.Error("failed to get task before add/review", "err", err, "userID", user.UserID, "taskNumber", taskNumber)
		return nil, err
	}

	stats, err := t.repo.GetStats(ctx, user.UserID)
	if err != nil {
		t.logger.Error("failed to get task stats", "err", err, "userID", user.UserID)
		return nil, err
	}

	return &model.AddTaskResult{
		Task:         *storedTask,
		Stats:        *stats,
		GoalProgress: buildGoalProgress(user, stats),
		IsReview:     isReview,
	}, nil
}

func nextReviewAt(reviewCount int64, from time.Time) *time.Time {
	location := moscowLocation()
	localTime := from.In(location)
	intervalDays := reviewIntervals[len(reviewIntervals)-1]
	if reviewCount-1 < int64(len(reviewIntervals)) {
		intervalDays = reviewIntervals[reviewCount-1]
	}

	nextDate := localTime.AddDate(0, 0, intervalDays)
	scheduled := time.Date(nextDate.Year(), nextDate.Month(), nextDate.Day(), 9, 0, 0, 0, location)
	utc := scheduled.UTC()
	return &utc
}

func moscowLocation() *time.Location {
	location, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.FixedZone("MSK", 3*60*60)
	}

	return location
}

func buildGoalProgress(user *model.User, stats *model.TaskStats) *model.UserProgress {
	items := make([]model.GoalProgress, 0, 4)
	appendGoal := func(label string, goal *int64, solved int64) {
		if goal == nil || *goal <= 0 {
			return
		}

		remaining := *goal - solved
		if remaining < 0 {
			remaining = 0
		}

		items = append(items, model.GoalProgress{
			Label:     label,
			Solved:    solved,
			Goal:      *goal,
			Remaining: remaining,
		})
	}

	appendGoal("Total", user.GoalTotal, stats.Total)
	appendGoal("Easy", user.GoalEasy, stats.Easy)
	appendGoal("Medium", user.GoalMedium, stats.Medium)
	appendGoal("Hard", user.GoalHard, stats.Hard)

	if len(items) == 0 {
		return nil
	}

	return &model.UserProgress{Items: items}
}
