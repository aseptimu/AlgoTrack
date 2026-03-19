package task

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
	return &TaskRepo{db: db}
}

func (t *TaskRepo) Create(ctx context.Context, task *model.Task, userID int64) (*int64, error) {
	var taskID int64

	err := t.db.Pool.QueryRow(ctx, `
		INSERT INTO algo_tasks (user_id, link, description, task_number, difficulty)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT ON CONSTRAINT algo_tasks_user_task_number_uq DO NOTHING
		RETURNING id
	`,
		userID,
		task.Link,
		task.Title,
		task.TaskNumber,
		task.Difficulty,
	).Scan(&taskID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, service.ErrTaskAlreadyExists
		}
		return nil, err
	}

	return &taskID, nil
}

func (t *TaskRepo) GetStats(ctx context.Context, userID int64) (*model.TaskStats, error) {
	stats := &model.TaskStats{}

	err := t.db.Pool.QueryRow(ctx, `
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE difficulty = 'Easy') AS easy,
			COUNT(*) FILTER (WHERE difficulty = 'Medium') AS medium,
			COUNT(*) FILTER (WHERE difficulty = 'Hard') AS hard
		FROM algo_tasks
		WHERE user_id = $1
	`, userID).Scan(
		&stats.Total,
		&stats.Easy,
		&stats.Medium,
		&stats.Hard,
	)
	if err != nil {
		return nil, err
	}

	return stats, nil
}
