package service

import (
	"context"
	"github.com/aseptimu/AlgoTrack/internal/model"
)

type UserManager interface {
	EnsureExistsAndGet(ctx context.Context, user *model.User) (*model.User, error)
	BuildWelcomeMessage(ctx context.Context, user *model.User) (string, error)
	BuildGoalMessage(ctx context.Context, user *model.User) (string, error)
	SetGoal(ctx context.Context, userId int64, goal int64, difficulty *string) error
	GetUser(ctx context.Context, userId int64) (*model.User, error)
}
