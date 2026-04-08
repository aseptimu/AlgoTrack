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

// BuildGoalProgress constructs goal progress items from a user's goals and task stats.
func BuildGoalProgress(user *User, stats *TaskStats) *UserProgress {
	items := make([]GoalProgress, 0, 4)
	appendGoal := func(label string, goal *int64, solved int64) {
		if goal == nil || *goal <= 0 {
			return
		}

		remaining := *goal - solved
		if remaining < 0 {
			remaining = 0
		}

		items = append(items, GoalProgress{
			Label:     label,
			Solved:    solved,
			Goal:      *goal,
			Remaining: remaining,
		})
	}

	appendGoal("Total", user.GoalTotal, stats.Total)
	appendGoal("Easy", user.GoalEasy, stats.Easy)
	appendGoal("Medium", user.GoalMedium, stats.Medium)
	appendGoal("Hard", user.GoalHard, stats.Hard)

	if len(items) == 0 {
		return nil
	}

	return &UserProgress{Items: items}
}
