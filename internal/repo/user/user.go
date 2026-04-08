package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/aseptimu/AlgoTrack/internal/db"
	"github.com/aseptimu/AlgoTrack/internal/model"
	"github.com/aseptimu/AlgoTrack/internal/service"
	"github.com/jackc/pgx/v5"
)

type TgUserRepo struct {
	db *db.DB
}

func NewTgUserRepo(db *db.DB) *TgUserRepo {
	return &TgUserRepo{db}
}

func (t *TgUserRepo) Create(ctx context.Context, user *model.User) (*int64, error) {
	var returnedUserID int64

	err := t.db.Pool.
		QueryRow(ctx,
			`INSERT INTO tg_user (user_id, chat_id, username, created_at)
			 VALUES ($1, $2, $3, NOW())
			 ON CONFLICT (user_id) DO UPDATE SET chat_id = EXCLUDED.chat_id
			 RETURNING user_id`,
			user.UserID,
			user.ChatID,
			user.Username,
		).
		Scan(&returnedUserID)

	if err != nil {
		return nil, fmt.Errorf("user.Create: %w", err)
	}

	return &returnedUserID, nil
}

func (t *TgUserRepo) Get(ctx context.Context, userId int64) (*model.User, error) {
	user := &model.User{}

	err := t.db.Pool.QueryRow(ctx,
		"SELECT user_id, chat_id, username, created_at, goal_total, goal_easy, goal_medium, goal_hard FROM tg_user WHERE user_id=$1",
		userId,
	).Scan(
		&user.UserID,
		&user.ChatID,
		&user.Username,
		&user.CreatedAt,
		&user.GoalTotal,
		&user.GoalEasy,
		&user.GoalMedium,
		&user.GoalHard,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, service.ErrTgUserNotFound
		}
		return nil, fmt.Errorf("user.Get: %w", err)
	}
	return user, nil
}

func (t *TgUserRepo) UpdateGoal(ctx context.Context, userId int64, goal int64, difficulty *string) error {
	query := `UPDATE tg_user SET goal_total = $2 WHERE user_id = $1`
	args := []any{userId, goal}

	if difficulty != nil {
		switch *difficulty {
		case "Easy":
			query = `UPDATE tg_user SET goal_easy = $2 WHERE user_id = $1`
		case "Medium":
			query = `UPDATE tg_user SET goal_medium = $2 WHERE user_id = $1`
		case "Hard":
			query = `UPDATE tg_user SET goal_hard = $2 WHERE user_id = $1`
		}
	}

	_, err := t.db.Pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("user.UpdateGoal: %w", err)
	}
	return nil
}
