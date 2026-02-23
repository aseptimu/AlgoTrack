package repo

import (
	"context"
	"errors"
	"github.com/aseptimu/AlgoTrack/internal/db"
	"github.com/aseptimu/AlgoTrack/internal/model"
	"github.com/jackc/pgx/v5"
)

var ErrTgUserNotFound = errors.New("user not found")

type TgUserRepo struct {
	db *db.DB
}

func NewTgUserRepo(db *db.DB) *TgUserRepo {
	return &TgUserRepo{db}
}

func (t *TgUserRepo) Create(ctx context.Context, userId int64, chatId int64, username string) (*int64, error) {
	var returnedUserID int64

	err := t.db.Pool.
		QueryRow(ctx,
			"INSERT INTO tg_user (user_id, chat_id, username, created_at) VALUES ($1, $2, $3, NOW()) RETURNING user_id",
			userId,
			chatId,
			username,
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
		"SELECT user_id, chat_id, username, created_at FROM tg_user WHERE user_id=$1",
		userId,
	).Scan(&user.UserID, &user.ChatID, &user.Username, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTgUserNotFound
		}
		return nil, err
	}
	return user, nil
}
