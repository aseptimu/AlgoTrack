package review

import "github.com/aseptimu/AlgoTrack/internal/model"

// Difficulty buckets recognized by the cap rule.
const (
	difficultyEasy   = "Easy"
	difficultyMedium = "Medium"
	difficultyHard   = "Hard"
)

// CapByDifficulty trims a sorted-oldest-first list of due reviews down to a
// small daily reminder bundle:
//
//   - if any Hard is due, return only that single oldest Hard;
//   - otherwise if any Medium is due, return the oldest Medium and (if any)
//     the oldest Easy too — at most 2 items;
//   - otherwise (only Easy is due), return up to 3 oldest Easy items.
//
// Input is expected to be ordered by next_review_at ASC. Items with an
// unknown / empty difficulty are bucketed as Easy so they still surface.
func CapByDifficulty(tasks []model.DueReviewTask) []model.DueReviewTask {
	var hard, medium, easy []model.DueReviewTask
	for _, t := range tasks {
		switch t.Difficulty {
		case difficultyHard:
			hard = append(hard, t)
		case difficultyMedium:
			medium = append(medium, t)
		default:
			easy = append(easy, t)
		}
	}

	if len(hard) > 0 {
		return []model.DueReviewTask{hard[0]}
	}
	if len(medium) > 0 {
		out := []model.DueReviewTask{medium[0]}
		if len(easy) > 0 {
			out = append(out, easy[0])
		}
		return out
	}
	if len(easy) > 0 {
		n := 3
		if n > len(easy) {
			n = len(easy)
		}
		return easy[:n]
	}
	return nil
}
