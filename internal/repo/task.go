package repo

import (
	"context"
	"errors"
	"github.com/aseptimu/AlgoTrack/internal/db"
	"github.com/aseptimu/AlgoTrack/internal/model"
	"github.com/aseptimu/AlgoTrack/internal/service"
	"github.com/jackc/pgx/v5"
)

type TaskRepo struct {
	db *db.DB
}

func NewTaskRepo(db *db.DB) *TaskRepo {
	return &TaskRepo{db}
}

func (t *TaskRepo) CreateTask(ctx context.Context, task *model.Task, userID int64) (*int64, error) {
	var taskId int64
	err := t.db.Pool.QueryRow(ctx,
		`INSERT INTO algo_tasks (user_id, link, description, task_number)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (task_number) DO NOTHING 
		 RETURNING id`,
		userID, task.Link, task.Description, task.TaskNumber,
	).Scan(&taskId)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, service.ErrTaskAlreadyExists
		}
		return nil, err
	}
	return &taskId, nil
}
