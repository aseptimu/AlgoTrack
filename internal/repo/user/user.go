package user

import (
	"context"
	"errors"
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
			"INSERT INTO tg_user (user_id, chat_id, username, created_at) VALUES ($1, $2, $3, NOW()) RETURNING user_id",
			user.UserID,
			user.ChatID,
			user.Username,
		).
		Scan(&returnedUserID)

	if err != nil {
		return nil, err
	}

	return &returnedUserID, nil
}

func (t *TgUserRepo) Get(ctx context.Context, userId int64) (*model.User, error) {
	user := &model.User{}

	err := t.db.Pool.QueryRow(ctx,
		"SELECT user_id, chat_id, username, created_at, goal_total FROM tg_user WHERE user_id=$1",
		userId,
	).Scan(&user.UserID, &user.ChatID, &user.Username, &user.CreatedAt, &user.GoalTotal)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, service.ErrTgUserNotFound
		}
		return nil, err
	}
	return user, nil
}

func (t *TgUserRepo) UpdateGoal(ctx context.Context, userId int64, goal int64) error {
	_, err := t.db.Pool.Exec(ctx,
		`UPDATE tg_user
		 SET goal_total = $2
		 WHERE user_id = $1`,
		userId,
		goal,
	)
	return err
}

func (t *TgUserRepo) CountSolvedTasks(ctx context.Context, userId int64) (int64, error) {
	var count int64

	err := t.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*)
		 FROM algo_tasks
		 WHERE user_id = $1`,
		userId,
	).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}
