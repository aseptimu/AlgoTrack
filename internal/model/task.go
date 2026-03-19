package model

type Task struct {
	ID         *int64
	UserID     *int64
	TaskNumber int64
	Link       string
	Title      *string
	Difficulty *string
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
}
