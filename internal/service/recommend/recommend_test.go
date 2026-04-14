package recommend

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/aseptimu/AlgoTrack/internal/catalog"
)

type fakeRepo struct {
	mu       sync.Mutex
	solved   map[int]bool
	rec      map[int]bool
	lastHard time.Time
	hasHard  bool
	err      error
}

func (f *fakeRepo) GetSolvedNumbers(_ context.Context, _ int64) (map[int]bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	out := make(map[int]bool, len(f.solved))
	for k, v := range f.solved {
		out[k] = v
	}
	return out, nil
}
func (f *fakeRepo) RecommendedNumbers(_ context.Context, _ int64) (map[int]bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make(map[int]bool, len(f.rec))
	for k, v := range f.rec {
		out[k] = v
	}
	return out, nil
}
func (f *fakeRepo) MarkRecommended(_ context.Context, _ int64, n int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.rec == nil {
		f.rec = map[int]bool{}
	}
	f.rec[n] = true
	return nil
}
func (f *fakeRepo) LastHardCreatedAt(_ context.Context, _ int64) (time.Time, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.lastHard, f.hasHard, nil
}

func newSvc(repo *fakeRepo, now time.Time) *Service {
	s := New(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	s.now = func() time.Time { return now }
	return s
}

func TestNext_PicksFirstUnseenInDefault(t *testing.T) {
	repo := &fakeRepo{}
	s := newSvc(repo, time.Now())
	p, err := s.Next(context.Background(), 1, "default")
	if err != nil {
		t.Fatal(err)
	}
	if p == nil || p.Number != catalog.NeetCode150.Problems[0].Number {
		t.Errorf("want first NeetCode 150 entry, got %+v", p)
	}
	if !repo.rec[p.Number] {
		t.Errorf("MarkRecommended not called for #%d", p.Number)
	}
}

func TestNext_SkipsSolvedAndRecommended(t *testing.T) {
	first := catalog.NeetCode150.Problems[0]
	second := catalog.NeetCode150.Problems[1]
	repo := &fakeRepo{
		solved: map[int]bool{first.Number: true},
		rec:    map[int]bool{second.Number: true},
	}
	s := newSvc(repo, time.Now())
	p, err := s.Next(context.Background(), 1, "default")
	if err != nil {
		t.Fatal(err)
	}
	if p.Number == first.Number || p.Number == second.Number {
		t.Errorf("returned a filtered entry: %+v", p)
	}
}

func TestNext_HardCooldownSkipsHardWithin14Days(t *testing.T) {
	repo := &fakeRepo{
		hasHard:  true,
		lastHard: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
	}
	s := newSvc(repo, time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)) // 9 days later

	for range catalog.NeetCode150.Problems {
		p, err := s.Next(context.Background(), 1, "default")
		if err != nil {
			break
		}
		if p.Difficulty == catalog.Hard {
			t.Fatalf("hard surfaced inside 14-day cooldown: #%d %s", p.Number, p.Title)
		}
	}
}

func TestNext_HardAvailableAfterCooldown(t *testing.T) {
	// Simulate: every easy/medium already solved or recommended so that the
	// only candidate left is a Hard. Cooldown is over.
	solved := map[int]bool{}
	for _, p := range catalog.NeetCode150.Problems {
		if p.Difficulty != catalog.Hard {
			solved[p.Number] = true
		}
	}
	for _, p := range catalog.Popular.Problems {
		solved[p.Number] = true
	}
	repo := &fakeRepo{
		solved:   solved,
		hasHard:  true,
		lastHard: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
	}
	s := newSvc(repo, time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)) // 30 days later
	p, err := s.Next(context.Background(), 1, "default")
	if err != nil {
		t.Fatal(err)
	}
	if p.Difficulty != catalog.Hard {
		t.Errorf("expected hard after cooldown elapsed; got %s #%d", p.Difficulty, p.Number)
	}
}

func TestNext_JsModeUsesJsCatalogFirst(t *testing.T) {
	s := newSvc(&fakeRepo{}, time.Now())
	p, err := s.Next(context.Background(), 1, "js")
	if err != nil {
		t.Fatal(err)
	}
	if p.Number != catalog.LeetCodeJS30.Problems[0].Number {
		t.Errorf("js mode should start with LeetCodeJS30, got #%d", p.Number)
	}
}

func TestNext_FallsThroughCatalogChain(t *testing.T) {
	// Mark every NeetCode 150 problem as recommended so the engine has to
	// walk into the Popular fallback.
	rec := map[int]bool{}
	for _, p := range catalog.NeetCode150.Problems {
		rec[p.Number] = true
	}
	repo := &fakeRepo{rec: rec}
	s := newSvc(repo, time.Now())
	p, err := s.Next(context.Background(), 1, "default")
	if err != nil {
		t.Fatal(err)
	}
	// Must come from Popular.
	found := false
	for _, pp := range catalog.Popular.Problems {
		if pp.Number == p.Number {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected Popular fallback, got #%d", p.Number)
	}
}

func TestNextDailyBundle_EasyGivesTwo(t *testing.T) {
	repo := &fakeRepo{}
	s := newSvc(repo, time.Now())
	out, err := s.NextDailyBundle(context.Background(), 1, "default")
	if err != nil {
		t.Fatal(err)
	}
	// First problem in NeetCode 150 is Contains Duplicate (Easy), second
	// available Easy is Valid Anagram. Both should be returned.
	if len(out) != 2 {
		t.Fatalf("expected 2 picks for Easy day, got %d", len(out))
	}
	for _, p := range out {
		if p.Difficulty != "Easy" {
			t.Errorf("expected only Easy in daily bundle when first is Easy; got %s #%d", p.Difficulty, p.Number)
		}
	}
}

func TestNextDailyBundle_MediumGivesOne(t *testing.T) {
	// Knock out every Easy so the first pick is Medium.
	solved := map[int]bool{}
	for _, p := range catalog.NeetCode150.Problems {
		if p.Difficulty == "Easy" {
			solved[p.Number] = true
		}
	}
	repo := &fakeRepo{solved: solved}
	s := newSvc(repo, time.Now())
	out, err := s.NextDailyBundle(context.Background(), 1, "default")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("Medium-day bundle should be exactly 1, got %d", len(out))
	}
	if out[0].Difficulty != "Medium" {
		t.Errorf("first pick should be Medium, got %s", out[0].Difficulty)
	}
}

func TestNextDailyBundle_ExhaustedReturnsEmpty(t *testing.T) {
	rec := map[int]bool{}
	for _, p := range catalog.NeetCode150.Problems {
		rec[p.Number] = true
	}
	for _, p := range catalog.Popular.Problems {
		rec[p.Number] = true
	}
	repo := &fakeRepo{rec: rec}
	s := newSvc(repo, time.Now())
	out, err := s.NextDailyBundle(context.Background(), 1, "default")
	if err != nil {
		t.Fatalf("exhausted catalog should return empty without error, got %v", err)
	}
	if len(out) != 0 {
		t.Errorf("exhausted catalog should yield empty bundle, got %d picks", len(out))
	}
}

func TestNext_ExhaustedReturnsSentinel(t *testing.T) {
	rec := map[int]bool{}
	for _, p := range catalog.NeetCode150.Problems {
		rec[p.Number] = true
	}
	for _, p := range catalog.Popular.Problems {
		rec[p.Number] = true
	}
	repo := &fakeRepo{rec: rec}
	s := newSvc(repo, time.Now())
	_, err := s.Next(context.Background(), 1, "default")
	if !errors.Is(err, ErrCatalogExhausted) {
		t.Errorf("expected ErrCatalogExhausted, got %v", err)
	}
}
