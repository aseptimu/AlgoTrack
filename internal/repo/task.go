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

func (t *TaskRepo) CreateTask(ctx context.Context, task *model.Task) (*int64, error) {
	var taskId int64
	err := t.db.Pool.QueryRow(ctx,
		`INSERT INTO algo_tasks (user_id, link, description)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, link) DO NOTHING 
		 RETURNING id`,
		task.UserID, task.Link, task.Description,
	).Scan(&taskId)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, service.ErrTaskAlreadyExists
		}
		return nil, err
	}
	return &taskId, nil
}
