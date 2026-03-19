package task

import (
	"context"
	"github.com/aseptimu/AlgoTrack/internal/model"
	"github.com/aseptimu/AlgoTrack/internal/service"
	"log/slog"
)

type ProblemProvider interface {
	GetProblemByNumber(ctx context.Context, number int64) (*model.ProblemInfo, error)
}

type TaskManager interface {
	CreateTask(ctx context.Context, task *model.Task, userID int64) (*int64, error)
}

type TaskService struct {
	userManager service.UserManager
	repo        TaskManager
	problems    ProblemProvider
	logger      *slog.Logger
}

func NewTaskService(userManager service.UserManager, repo TaskManager, problem ProblemProvider, logger *slog.Logger) *TaskService {
	return &TaskService{userManager, repo, problem, logger}
}

func (t *TaskService) Add(ctx context.Context, task *model.Task, user *model.User) error {
	_, err := t.userManager.EnsureExistsAndGet(ctx, user)
	if err != nil {
		t.logger.Error("failed to ensure user existence while creating task", "err", err)
		return err
	}

	problem, err := t.problems.GetProblemByNumber(ctx, task.TaskNumber)
	if err != nil {
		t.logger.Error("failed to get leetcode problem", "err", err, "number", task.TaskNumber)
		return err
	}

	task.Link = problem.Link
	task.Description = &problem.Title
	task.Difficulty = &problem.Difficulty
	_, err = t.repo.CreateTask(ctx, task, user.UserID)
	if err != nil {
		t.logger.Error("failed to create task", "err", err)
		return err
	}
	return nil
}
