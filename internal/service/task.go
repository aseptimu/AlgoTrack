package service

import (
	"context"
	"errors"
	"github.com/aseptimu/AlgoTrack/internal/model"
)

var ErrTaskAlreadyExists = errors.New("task already exists for user")

type TaskManager interface {
	CreateTask(ctx context.Context, task *model.Task) (*int64, error)
}

type TaskService struct {
	repo TaskManager
}

func NewTaskService(repo TaskManager) *TaskService {
	return &TaskService{repo}
}

func (t *TaskService) Create(ctx context.Context, task *model.Task) error {
	_, err := t.repo.CreateTask(ctx, task)

	if err != nil {
		return err
	}
	return nil
}
