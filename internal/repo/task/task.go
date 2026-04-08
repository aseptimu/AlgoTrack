package task

import (
	"context"
	"errors"
	"fmt"
	"time"

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

func (t *TaskRepo) Create(ctx context.Context, task *model.Task, userID int64) (*model.Task, error) {
	storedTask := *task

	err := t.db.Pool.QueryRow(ctx, `
		INSERT INTO algo_tasks (
			user_id,
			link,
			description,
			task_number,
			difficulty,
			review_count,
			last_reviewed_at,
			next_review_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT ON CONSTRAINT algo_tasks_user_task_number_uq DO NOTHING
		RETURNING id, review_count, last_reviewed_at, next_review_at
	`,
		userID,
		task.Link,
		task.Title,
		task.TaskNumber,
		task.Difficulty,
		task.ReviewCount,
		task.LastReviewedAt,
		task.NextReviewAt,
	).Scan(
		&storedTask.ID,
		&storedTask.ReviewCount,
		&storedTask.LastReviewedAt,
		&storedTask.NextReviewAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, service.ErrTaskAlreadyExists
		}
		return nil, fmt.Errorf("task.Create: %w", err)
	}

	return &storedTask, nil
}

func (t *TaskRepo) GetByTaskNumber(ctx context.Context, userID, taskNumber int64) (*model.Task, error) {
	task := &model.Task{}

	err := t.db.Pool.QueryRow(ctx, `
		SELECT id, user_id, task_number, link, description, difficulty, review_count, last_reviewed_at, next_review_at
		FROM algo_tasks
		WHERE user_id = $1 AND task_number = $2
	`, userID, taskNumber).Scan(
		&task.ID,
		&task.UserID,
		&task.TaskNumber,
		&task.Link,
		&task.Title,
		&task.Difficulty,
		&task.ReviewCount,
		&task.LastReviewedAt,
		&task.NextReviewAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, service.ErrTaskNotFound
		}
		return nil, fmt.Errorf("task.GetByTaskNumber: %w", err)
	}

	return task, nil
}

func (t *TaskRepo) Review(ctx context.Context, task *model.Task, userID int64) (*model.Task, error) {
	storedTask := *task

	err := t.db.Pool.QueryRow(ctx, `
		UPDATE algo_tasks
		SET
			link = $3,
			description = $4,
			difficulty = $5,
			review_count = $6,
			last_reviewed_at = $7,
			next_review_at = $8
		WHERE user_id = $1 AND task_number = $2
		RETURNING id, review_count, last_reviewed_at, next_review_at
	`,
		userID,
		task.TaskNumber,
		task.Link,
		task.Title,
		task.Difficulty,
		task.ReviewCount,
		task.LastReviewedAt,
		task.NextReviewAt,
	).Scan(
		&storedTask.ID,
		&storedTask.ReviewCount,
		&storedTask.LastReviewedAt,
		&storedTask.NextReviewAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, service.ErrTaskNotFound
		}
		return nil, fmt.Errorf("task.Review: %w", err)
	}

	return &storedTask, nil
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
		return nil, fmt.Errorf("task.GetStats: %w", err)
	}

	return stats, nil
}

func (t *TaskRepo) GetDueReviews(ctx context.Context, nowTime time.Time) ([]model.DueReviewBatch, error) {
	rows, err := t.db.Pool.Query(ctx, `
		SELECT
			u.user_id,
			u.chat_id,
			t.task_number,
			COALESCE(t.description, ''),
			t.link,
			COALESCE(t.difficulty, ''),
			t.review_count,
			t.last_reviewed_at,
			t.next_review_at
		FROM algo_tasks t
		JOIN tg_user u ON u.user_id = t.user_id
		WHERE t.next_review_at <= $1
		ORDER BY u.user_id, t.next_review_at, t.task_number
	`, nowTime)
	if err != nil {
		return nil, fmt.Errorf("task.GetDueReviews: %w", err)
	}
	defer rows.Close()

	batches := make([]model.DueReviewBatch, 0)
	indexByUser := make(map[int64]int)

	for rows.Next() {
		var (
			userID int64
			chatID int64
			task   model.DueReviewTask
		)

		if err := rows.Scan(
			&userID,
			&chatID,
			&task.TaskNumber,
			&task.Title,
			&task.Link,
			&task.Difficulty,
			&task.ReviewCount,
			&task.LastReviewedAt,
			&task.NextReviewAt,
		); err != nil {
			return nil, err
		}

		batchIndex, ok := indexByUser[userID]
		if !ok {
			batches = append(batches, model.DueReviewBatch{
				UserID: userID,
				ChatID: chatID,
				Tasks:  []model.DueReviewTask{task},
			})
			indexByUser[userID] = len(batches) - 1
			continue
		}

		batches[batchIndex].Tasks = append(batches[batchIndex].Tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("task.GetDueReviews: %w", err)
	}

	return batches, nil
}
