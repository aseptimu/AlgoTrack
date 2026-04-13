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
		"SELECT user_id, chat_id, username, leetcode_username, created_at, goal_total, goal_easy, goal_medium, goal_hard FROM tg_user WHERE user_id=$1",
		userId,
	).Scan(
		&user.UserID,
		&user.ChatID,
		&user.Username,
		&user.LeetCodeUsername,
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

func (t *TgUserRepo) UpdateLeetCodeUsername(ctx context.Context, userID int64, leetcodeUsername string) error {
	_, err := t.db.Pool.Exec(ctx,
		`UPDATE tg_user SET leetcode_username = $2 WHERE user_id = $1`,
		userID, leetcodeUsername,
	)
	return err
}

func (t *TgUserRepo) GetUsersWithLeetCode(ctx context.Context) ([]model.User, error) {
	rows, err := t.db.Pool.Query(ctx,
		`SELECT user_id, chat_id, username, leetcode_username FROM tg_user WHERE leetcode_username IS NOT NULL AND leetcode_username != ''`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.UserID, &u.ChatID, &u.Username, &u.LeetCodeUsername); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// GetLastPolledSubmissionID returns the last accepted submission id we've
// inspected for this user, or "" with found=false if none was recorded yet.
// Used by the poller to dedup across restarts.
func (t *TgUserRepo) GetLastPolledSubmissionID(ctx context.Context, userID int64) (string, bool, error) {
	var id *string
	err := t.db.Pool.QueryRow(ctx,
		`SELECT last_polled_submission_id FROM tg_user WHERE user_id = $1`,
		userID,
	).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", false, nil
		}
		return "", false, err
	}
	if id == nil {
		return "", false, nil
	}
	return *id, true, nil
}

func (t *TgUserRepo) SetLastPolledSubmissionID(ctx context.Context, userID int64, submissionID string) error {
	_, err := t.db.Pool.Exec(ctx,
		`UPDATE tg_user SET last_polled_submission_id = $2 WHERE user_id = $1`,
		userID, submissionID,
	)
	return err
}

// WasNotifiedToday reports whether the poller has already processed this
// (user, problem, day) tuple. The day is passed as a string in YYYY-MM-DD
// format so the caller controls the timezone (Europe/Moscow in our case).
func (t *TgUserRepo) WasNotifiedToday(ctx context.Context, userID int64, titleSlug, day string) (bool, error) {
	var exists bool
	err := t.db.Pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM notified_problem WHERE user_id = $1 AND title_slug = $2 AND notified_day = $3::date)`,
		userID, titleSlug, day,
	).Scan(&exists)
	return exists, err
}

func (t *TgUserRepo) MarkNotified(ctx context.Context, userID int64, titleSlug, day string) error {
	_, err := t.db.Pool.Exec(ctx,
		`INSERT INTO notified_problem (user_id, title_slug, notified_day)
		 VALUES ($1, $2, $3::date)
		 ON CONFLICT DO NOTHING`,
		userID, titleSlug, day,
	)
	return err
}
