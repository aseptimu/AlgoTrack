package model

import "time"

type Task struct {
	ID             *int64
	UserID         *int64
	TaskNumber     int64
	Link           string
	Title          *string
	Difficulty     *string
	ReviewCount    int64
	LastReviewedAt *time.Time
	NextReviewAt   *time.Time
}

type TaskStats struct {
	Total  int64
	Easy   int64
	Medium int64
	Hard   int64
}

type AddTaskResult struct {
	Task         Task
	Stats        TaskStats
	GoalProgress *UserProgress
	IsReview     bool
}

type UserStatsResult struct {
	Stats          TaskStats
	Streak         int64
	PendingReviews int64
	GoalProgress   *UserProgress
}

type DueReviewTask struct {
	TaskNumber     int64
	Title          string
	Link           string
	Difficulty     string
	ReviewCount    int64
	LastReviewedAt time.Time
	NextReviewAt   time.Time
}

type DueReviewBatch struct {
	UserID int64
	ChatID int64
	Tasks  []DueReviewTask
}
