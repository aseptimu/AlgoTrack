package model

type ProblemInfo struct {
	Number     int
	Title      string
	TitleSlug  string
	Difficulty string
	Link       string
	Platform   string
}

// LeetCodeSubmission represents a single accepted submission from LeetCode.
type LeetCodeSubmission struct {
	ID        string
	Title     string
	TitleSlug string
	Timestamp string // Unix timestamp as string from LeetCode API.
}
