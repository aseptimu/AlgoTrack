// Package recommend picks the next problem to suggest to a user.
//
// Rules:
//   - Walk the catalog chain for the user's mode ("default" or "js").
//   - Skip problems already in the user's algo_tasks (solved/tracked).
//   - Skip problems already in recommended_problem (already pitched).
//   - Hard problems are throttled to at most 1 every 14 days, measured by
//     the most recent algo_tasks row with difficulty='Hard' for the user.
//   - The first eligible problem in catalog order wins.
//
// On a successful pick the service writes a row into recommended_problem
// so subsequent calls don't re-pitch the same problem.
package recommend

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/aseptimu/AlgoTrack/internal/catalog"
)

// HardCooldown is the minimum gap between two Hard-difficulty recommendations.
const HardCooldown = 14 * 24 * time.Hour

// Repo is the persistence surface the recommender relies on.
type Repo interface {
	GetSolvedNumbers(ctx context.Context, userID int64) (map[int]bool, error)
	RecommendedNumbers(ctx context.Context, userID int64) (map[int]bool, error)
	MarkRecommended(ctx context.Context, userID int64, taskNumber int) error
	LastHardCreatedAt(ctx context.Context, userID int64) (time.Time, bool, error)
}

type Service struct {
	repo Repo
	log  *slog.Logger
	now  func() time.Time
}

func New(repo Repo, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{repo: repo, log: log, now: time.Now}
}

// ErrCatalogExhausted is returned when no catalog in the chain has any
// remaining eligible problem for the user.
var ErrCatalogExhausted = errors.New("recommend: catalog exhausted")

// Next picks the next problem to recommend, marks it as recommended, and
// returns it. mode ∈ {"default", "js"}; unknown values fall back to default.
func (s *Service) Next(ctx context.Context, userID int64, mode string) (*catalog.Problem, error) {
	solved, err := s.repo.GetSolvedNumbers(ctx, userID)
	if err != nil {
		return nil, err
	}
	already, err := s.repo.RecommendedNumbers(ctx, userID)
	if err != nil {
		return nil, err
	}

	skipHard := false
	if last, ok, err := s.repo.LastHardCreatedAt(ctx, userID); err != nil {
		return nil, err
	} else if ok && s.now().Sub(last) < HardCooldown {
		skipHard = true
	}

	for _, cat := range catalog.Chain(mode) {
		for _, p := range cat.Problems {
			if solved[p.Number] || already[p.Number] {
				continue
			}
			if skipHard && p.Difficulty == catalog.Hard {
				continue
			}
			if err := s.repo.MarkRecommended(ctx, userID, p.Number); err != nil {
				return nil, err
			}
			s.log.Info("recommend: picked",
				"userID", userID,
				"mode", mode,
				"catalog", cat.Name,
				"taskNumber", p.Number,
				"title", p.Title,
				"difficulty", p.Difficulty,
			)
			pp := p
			return &pp, nil
		}
	}

	return nil, ErrCatalogExhausted
}
