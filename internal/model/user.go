package model

import "time"

type User struct {
	UserID    int64
	ChatID    int64
	Username  string
	CreatedAt *time.Time
}
