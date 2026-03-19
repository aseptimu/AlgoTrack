package task

import (
	"context"
	"log/slog"

	"github.com/aseptimu/AlgoTrack/internal/model"
)

type ProblemProvider interface {
	GetProblemByNumber(ctx context.Context, number int64) (*model.ProblemInfo, error)
}

type UserEnsurer interface {
	EnsureExistsAndGet(ctx context.Context, incomingUser *model.User) (*model.User, error)
}

type Repository interface {
	Create(ctx context.Context, task *model.Task, userID int64) (*int64, error)
	GetStats(ctx context.Context, userID int64) (*model.TaskStats, error)
}

type TaskService struct {
	users    UserEnsurer
	repo     Repository
	problems ProblemProvider
	logger   *slog.Logger
}

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
		t.logger.Error("failed to get problem by number", "err", err, "taskNumber", taskNumber)
		return nil, err
	}

	task := model.Task{
		UserID:     &user.UserID,
		TaskNumber: taskNumber,
		Link:       problem.Link,
		Title:      &problem.Title,
		Difficulty: &problem.Difficulty,
	}

	taskID, err := t.repo.Create(ctx, &task, user.UserID)
	if err != nil {
		t.logger.Error("failed to create task", "err", err, "userID", user.UserID, "taskNumber", taskNumber)
		return nil, err
	}
	task.ID = taskID

	stats, err := t.repo.GetStats(ctx, user.UserID)
	if err != nil {
		t.logger.Error("failed to get task stats", "err", err, "userID", user.UserID)
		return nil, err
	}

	return &model.AddTaskResult{
		Task:         task,
		Stats:        *stats,
		GoalProgress: buildGoalProgress(user, stats),
	}, nil
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
