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

// NextDailyBundle returns the set of new problems to surface in the
// 09:00 MSK morning message. It always picks one fresh problem; if that
// pick happens to be Easy, it tries to add a SECOND Easy on top so easy
// days get two warm-ups instead of one. Medium / Hard / catalog-exhausted
// stop at exactly one problem. The returned slice is empty only when even
// the first pick fails (catalog exhausted for this user).
func (s *Service) NextDailyBundle(ctx context.Context, userID int64, mode string) ([]catalog.Problem, error) {
	first, err := s.Next(ctx, userID, mode)
	if err != nil {
		if errors.Is(err, ErrCatalogExhausted) {
			return nil, nil
		}
		return nil, err
	}
	bundle := []catalog.Problem{*first}
	if first.Difficulty != catalog.Easy {
		return bundle, nil
	}
	second, err := s.nextOfDifficulty(ctx, userID, mode, catalog.Easy)
	if err != nil || second == nil {
		// No second Easy available, that's fine — keep the first pick.
		return bundle, nil
	}
	return append(bundle, *second), nil
}

// nextOfDifficulty is the constrained sibling of Next: it only returns a
// problem whose difficulty matches `want`. Used by NextDailyBundle to
// guarantee the second pick on Easy days is itself Easy.
func (s *Service) nextOfDifficulty(ctx context.Context, userID int64, mode, want string) (*catalog.Problem, error) {
	solved, err := s.repo.GetSolvedNumbers(ctx, userID)
	if err != nil {
		return nil, err
	}
	already, err := s.repo.RecommendedNumbers(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, cat := range catalog.Chain(mode) {
		for _, p := range cat.Problems {
			if solved[p.Number] || already[p.Number] {
				continue
			}
			if p.Difficulty != want {
				continue
			}
			if err := s.repo.MarkRecommended(ctx, userID, p.Number); err != nil {
				return nil, err
			}
			pp := p
			return &pp, nil
		}
	}
	return nil, nil
}

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
