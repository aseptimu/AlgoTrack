package model

import "time"

type User struct {
	UserID    int64
	ChatID    int64
	Username  string
	GoalTotal *int64
	CreatedAt *time.Time
}

type UserProgress struct {
	GoalTotal int64
	Solved    int64
	Remaining int64
}
