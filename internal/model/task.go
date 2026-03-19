package model

type Task struct {
	ID          *int64
	TaskNumber  int64
	Link        string
	Description *string
	Difficulty  *string
}
