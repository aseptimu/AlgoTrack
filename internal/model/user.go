package model

import "time"

type User struct {
	UserID           int64
	ChatID           int64
	Username         string
	LeetCodeUsername *string
	GoalTotal        *int64
	GoalEasy         *int64
	GoalMedium       *int64
	GoalHard         *int64
	CreatedAt        *time.Time
}

type GoalProgress struct {
	Label     string
	Solved    int64
	Goal      int64
	Remaining int64
}

type UserProgress struct {
	Items []GoalProgress
}
