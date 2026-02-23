package model

type Task struct {
	ID          *int64
	UserID      int64
	Link        string
	Description *string
}
