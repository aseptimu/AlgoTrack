package submission

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/aseptimu/AlgoTrack/internal/model"
	tgbot "github.com/go-telegram/bot"
	tgmodels "github.com/go-telegram/bot/models"
)

// --- mocks ---

type fakeFetcher struct {
	mu       sync.Mutex
	subs     map[string][]model.LeetCodeSubmission // username -> submissions, newest-first
	problems map[string]*model.ProblemInfo         // titleSlug -> problem
	subErr   error
	probErr  error
}

func (f *fakeFetcher) GetRecentAcceptedSubmissions(_ context.Context, username string, limit int) ([]model.LeetCodeSubmission, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.subErr != nil {
		return nil, f.subErr
	}
	all := f.subs[username]
	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}
	out := make([]model.LeetCodeSubmission, len(all))
	copy(out, all)
	return out, nil
}

func (f *fakeFetcher) GetProblemBySlug(_ context.Context, slug string) (*model.ProblemInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.probErr != nil {
		return nil, f.probErr
	}
	if p, ok := f.problems[slug]; ok {
		return p, nil
	}
	// Default stub so tests can omit explicit problem maps.
	return &model.ProblemInfo{Number: 1, Title: slug, TitleSlug: slug, Difficulty: "Easy", Link: "https://leetcode.com/problems/" + slug + "/"}, nil
}

func (f *fakeFetcher) setSubs(username string, subs []model.LeetCodeSubmission) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.subs == nil {
		f.subs = make(map[string][]model.LeetCodeSubmission)
	}
	f.subs[username] = subs
}

func (f *fakeFetcher) setProblem(slug string, p *model.ProblemInfo) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.problems == nil {
		f.problems = make(map[string]*model.ProblemInfo)
	}
	f.problems[slug] = p
}

type fakeUsers struct {
	users []model.User
	err   error
}

func (u *fakeUsers) GetUsersWithLeetCode(_ context.Context) ([]model.User, error) {
	if u.err != nil {
		return nil, u.err
	}
	return u.users, nil
}

type fakeTasks struct {
	mu        sync.Mutex
	calls     []model.ProblemInfo
	reviewMap map[int]int // problemNumber -> current review count
	err       error
}

func (t *fakeTasks) AddByProblem(_ context.Context, _ *model.User, p *model.ProblemInfo) (*model.AddTaskResult, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.err != nil {
		return nil, t.err
	}
	if t.reviewMap == nil {
		t.reviewMap = make(map[int]int)
	}
	t.calls = append(t.calls, *p)
	prev, exists := t.reviewMap[p.Number]
	t.reviewMap[p.Number] = prev + 1
	next := time.Now().Add(24 * time.Hour)
	return &model.AddTaskResult{
		Task: model.Task{
			TaskNumber:   int64(p.Number),
			ReviewCount:  int64(prev + 1),
			NextReviewAt: &next,
		},
		Stats:    model.TaskStats{},
		IsReview: exists,
	}, nil
}

func (t *fakeTasks) snapshotCalls() []model.ProblemInfo {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]model.ProblemInfo, len(t.calls))
	copy(out, t.calls)
	return out
}

type fakeSender struct {
	mu   sync.Mutex
	sent []*tgbot.SendMessageParams
	err  error
}

func (s *fakeSender) SendMessage(_ context.Context, p *tgbot.SendMessageParams) (*tgmodels.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return nil, s.err
	}
	s.sent = append(s.sent, p)
	return &tgmodels.Message{}, nil
}

func (s *fakeSender) snapshot() []*tgbot.SendMessageParams {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*tgbot.SendMessageParams, len(s.sent))
	copy(out, s.sent)
	return out
}

type memoryStore struct {
	mu        sync.Mutex
	watermark map[int64]string
	notified  map[string]bool // key = "userID|slug|day"
}

func newMemoryStore() *memoryStore {
	return &memoryStore{watermark: make(map[int64]string), notified: make(map[string]bool)}
}

func (m *memoryStore) GetLastPolledSubmissionID(_ context.Context, userID int64) (string, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.watermark[userID]
	return id, ok, nil
}

func (m *memoryStore) SetLastPolledSubmissionID(_ context.Context, userID int64, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.watermark[userID] = id
	return nil
}

func (m *memoryStore) WasNotifiedToday(_ context.Context, userID int64, slug, day string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.notified[notifKey(userID, slug, day)], nil
}

func (m *memoryStore) MarkNotified(_ context.Context, userID int64, slug, day string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notified[notifKey(userID, slug, day)] = true
	return nil
}

func notifKey(userID int64, slug, day string) string {
	return strconv.FormatInt(userID, 10) + "|" + slug + "|" + day
}

// --- helpers ---

func newSilentLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func ptr(s string) *string { return &s }

func mkUser(id int64, lc string) model.User {
	return model.User{UserID: id, ChatID: id * 10, Username: "u" + strconv.FormatInt(id, 10), LeetCodeUsername: ptr(lc)}
}

func mkSub(id, slug string, ts int64) model.LeetCodeSubmission {
	return model.LeetCodeSubmission{ID: id, Title: slug, TitleSlug: slug, Timestamp: strconv.FormatInt(ts, 10)}
}

func newTestPoller(f *fakeFetcher, u *fakeUsers, tk *fakeTasks, s *fakeSender, st *memoryStore, opts Options) *Poller {
	return NewPollerWithOptions(f, u, tk, st, s, newSilentLogger(), opts)
}

func daySecond(dayOffset int, hourMSK int) int64 {
	base := time.Date(2026, 4, 13, hourMSK-3, 0, 0, 0, time.UTC)
	return base.AddDate(0, 0, dayOffset).Unix()
}

// --- tests ---

func TestNewPoller_DefaultsApplied(t *testing.T) {
	p := NewPoller(&fakeFetcher{}, &fakeUsers{}, &fakeTasks{}, newMemoryStore(), &fakeSender{}, nil)
	if p.interval != DefaultPollInterval {
		t.Errorf("default interval = %v, want %v", p.interval, DefaultPollInterval)
	}
	if p.submissionsLimit != DefaultSubmissionsLimit {
		t.Errorf("default limit = %d, want %d", p.submissionsLimit, DefaultSubmissionsLimit)
	}
	if !p.enabled {
		t.Error("default poller should be enabled")
	}
}

func TestStart_DisabledReturnsImmediately(t *testing.T) {
	p := newTestPoller(&fakeFetcher{}, &fakeUsers{}, &fakeTasks{}, &fakeSender{}, newMemoryStore(), Options{Enabled: false})
	done := make(chan struct{})
	go func() { p.Start(context.Background()); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Start with disabled poller did not return")
	}
}

func TestPoll_FirstPollAbsorbsSilently(t *testing.T) {
	f := &fakeFetcher{}
	tk := &fakeTasks{}
	s := &fakeSender{}
	st := newMemoryStore()
	p := newTestPoller(f, &fakeUsers{users: []model.User{mkUser(1, "alice")}}, tk, s, st, Options{Enabled: true})

	f.setSubs("alice", []model.LeetCodeSubmission{
		mkSub("103", "z", daySecond(0, 18)),
		mkSub("102", "y", daySecond(0, 14)),
		mkSub("101", "x", daySecond(0, 12)),
	})
	p.poll(context.Background())

	if got := s.snapshot(); len(got) != 0 {
		t.Errorf("first poll should be silent, got %d sends", len(got))
	}
	if got := tk.snapshotCalls(); len(got) != 0 {
		t.Errorf("first poll should not auto-add, got %d AddByProblem calls", len(got))
	}
	id, ok, _ := st.GetLastPolledSubmissionID(context.Background(), 1)
	if !ok || id != "103" {
		t.Errorf("watermark should be 103, got %q (found=%v)", id, ok)
	}
}

func TestPoll_NewSubmissionAutoAddsAndNotifies(t *testing.T) {
	f := &fakeFetcher{}
	tk := &fakeTasks{}
	s := &fakeSender{}
	st := newMemoryStore()
	users := &fakeUsers{users: []model.User{mkUser(1, "alice")}}
	p := newTestPoller(f, users, tk, s, st, Options{Enabled: true})

	// Prime watermark so we are past the first-poll absorb.
	f.setSubs("alice", []model.LeetCodeSubmission{mkSub("100", "warmup", daySecond(-30, 0))})
	p.poll(context.Background())
	tk.calls = nil

	// Now a brand new accepted submission appears.
	f.setProblem("two-sum", &model.ProblemInfo{Number: 1, Title: "Two Sum", TitleSlug: "two-sum", Difficulty: "Easy", Link: "https://leetcode.com/problems/two-sum/"})
	f.setSubs("alice", []model.LeetCodeSubmission{
		mkSub("101", "two-sum", daySecond(0, 12)),
		mkSub("100", "warmup", daySecond(-30, 0)),
	})
	p.poll(context.Background())

	calls := tk.snapshotCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 auto-add, got %d", len(calls))
	}
	if calls[0].Number != 1 || calls[0].TitleSlug != "two-sum" {
		t.Errorf("auto-add wrong problem: %+v", calls[0])
	}
	sends := s.snapshot()
	if len(sends) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(sends))
	}
	if !contains(sends[0].Text, "Two Sum") || !contains(sends[0].Text, "Задача") {
		t.Errorf("notification missing expected fields, got %q", sends[0].Text)
	}
}

func TestPoll_SameDayRepeatSuppressedFromAddAndNotify(t *testing.T) {
	f := &fakeFetcher{}
	tk := &fakeTasks{}
	s := &fakeSender{}
	st := newMemoryStore()
	p := newTestPoller(f, &fakeUsers{users: []model.User{mkUser(1, "alice")}}, tk, s, st, Options{Enabled: true})

	// First, fire one notification + auto-add to seed the cooldown.
	f.setSubs("alice", []model.LeetCodeSubmission{mkSub("100", "warmup", daySecond(-30, 0))})
	p.poll(context.Background())
	f.setSubs("alice", []model.LeetCodeSubmission{
		mkSub("200", "two-sum", daySecond(0, 12)),
		mkSub("100", "warmup", daySecond(-30, 0)),
	})
	p.poll(context.Background())
	if len(tk.snapshotCalls()) != 1 {
		t.Fatalf("setup: want 1 call, got %d", len(tk.snapshotCalls()))
	}
	if len(s.snapshot()) != 1 {
		t.Fatalf("setup: want 1 send, got %d", len(s.snapshot()))
	}

	// Now the user solves the same problem again 6 hours later, same day MSK.
	f.setSubs("alice", []model.LeetCodeSubmission{
		mkSub("201", "two-sum", daySecond(0, 18)),
		mkSub("200", "two-sum", daySecond(0, 12)),
		mkSub("100", "warmup", daySecond(-30, 0)),
	})
	p.poll(context.Background())

	if got := tk.snapshotCalls(); len(got) != 1 {
		t.Errorf("same-day repeat must NOT call auto-add again; total calls = %d", len(got))
	}
	if got := s.snapshot(); len(got) != 1 {
		t.Errorf("same-day repeat must NOT send another notification; total sends = %d", len(got))
	}
}

func TestPoll_NextDayRepeatNotifiesAgain(t *testing.T) {
	f := &fakeFetcher{}
	tk := &fakeTasks{}
	s := &fakeSender{}
	st := newMemoryStore()
	p := newTestPoller(f, &fakeUsers{users: []model.User{mkUser(1, "alice")}}, tk, s, st, Options{Enabled: true})

	f.setSubs("alice", []model.LeetCodeSubmission{mkSub("100", "warmup", daySecond(-30, 0))})
	p.poll(context.Background())

	f.setSubs("alice", []model.LeetCodeSubmission{
		mkSub("200", "two-sum", daySecond(0, 12)),
		mkSub("100", "warmup", daySecond(-30, 0)),
	})
	p.poll(context.Background())

	// Next calendar day, same problem.
	f.setSubs("alice", []model.LeetCodeSubmission{
		mkSub("300", "two-sum", daySecond(1, 12)),
		mkSub("200", "two-sum", daySecond(0, 12)),
		mkSub("100", "warmup", daySecond(-30, 0)),
	})
	p.poll(context.Background())

	calls := tk.snapshotCalls()
	if len(calls) != 2 {
		t.Errorf("expected 2 auto-adds across two days, got %d", len(calls))
	}
	if len(s.snapshot()) != 2 {
		t.Errorf("expected 2 notifications across two days, got %d", len(s.snapshot()))
	}
}

func TestPoll_DifferentProblemsSameDayBothFire(t *testing.T) {
	f := &fakeFetcher{}
	tk := &fakeTasks{}
	s := &fakeSender{}
	st := newMemoryStore()
	p := newTestPoller(f, &fakeUsers{users: []model.User{mkUser(1, "alice")}}, tk, s, st, Options{Enabled: true})

	f.setSubs("alice", []model.LeetCodeSubmission{mkSub("100", "warmup", daySecond(-30, 0))})
	p.poll(context.Background())

	f.setProblem("two-sum", &model.ProblemInfo{Number: 1, Title: "Two Sum", TitleSlug: "two-sum", Difficulty: "Easy", Link: "x"})
	f.setProblem("valid-parentheses", &model.ProblemInfo{Number: 20, Title: "Valid Parentheses", TitleSlug: "valid-parentheses", Difficulty: "Easy", Link: "x"})
	f.setSubs("alice", []model.LeetCodeSubmission{
		mkSub("301", "valid-parentheses", daySecond(0, 14)),
		mkSub("300", "two-sum", daySecond(0, 12)),
		mkSub("100", "warmup", daySecond(-30, 0)),
	})
	p.poll(context.Background())

	if got := tk.snapshotCalls(); len(got) != 2 {
		t.Errorf("two distinct problems should both auto-add, got %d", len(got))
	}
	if got := s.snapshot(); len(got) != 2 {
		t.Errorf("two distinct problems should both notify, got %d", len(got))
	}
}

func TestPoll_FetcherErrorDoesNotPanic(t *testing.T) {
	f := &fakeFetcher{subErr: errors.New("network down")}
	tk := &fakeTasks{}
	s := &fakeSender{}
	p := newTestPoller(f, &fakeUsers{users: []model.User{mkUser(1, "alice")}}, tk, s, newMemoryStore(), Options{Enabled: true})
	p.poll(context.Background()) // must not panic
	if got := s.snapshot(); len(got) != 0 {
		t.Errorf("fetcher error should not produce sends, got %d", len(got))
	}
}

func TestPoll_FailedSendDoesNotMarkCooldown(t *testing.T) {
	f := &fakeFetcher{}
	tk := &fakeTasks{}
	s := &fakeSender{err: errors.New("telegram down")}
	st := newMemoryStore()
	p := newTestPoller(f, &fakeUsers{users: []model.User{mkUser(1, "alice")}}, tk, s, st, Options{Enabled: true})

	f.setSubs("alice", []model.LeetCodeSubmission{mkSub("100", "warmup", daySecond(-30, 0))})
	p.poll(context.Background())
	f.setSubs("alice", []model.LeetCodeSubmission{
		mkSub("200", "two-sum", daySecond(0, 12)),
		mkSub("100", "warmup", daySecond(-30, 0)),
	})
	p.poll(context.Background())

	// telegram comes back, new submission of the same problem on the same day
	s.mu.Lock()
	s.err = nil
	s.mu.Unlock()
	f.setSubs("alice", []model.LeetCodeSubmission{
		mkSub("201", "two-sum", daySecond(0, 18)),
		mkSub("200", "two-sum", daySecond(0, 12)),
		mkSub("100", "warmup", daySecond(-30, 0)),
	})
	p.poll(context.Background())

	if got := s.snapshot(); len(got) != 1 {
		t.Errorf("after a failed send the cooldown must not have been set; want 1 successful send, got %d", len(got))
	}
}

func TestPoll_StatePersistsAcrossPollerInstance(t *testing.T) {
	f := &fakeFetcher{}
	tk := &fakeTasks{}
	s := &fakeSender{}
	st := newMemoryStore()
	users := &fakeUsers{users: []model.User{mkUser(1, "alice")}}

	// Simulate a "first run".
	p1 := newTestPoller(f, users, tk, s, st, Options{Enabled: true})
	f.setSubs("alice", []model.LeetCodeSubmission{mkSub("100", "warmup", daySecond(-30, 0))})
	p1.poll(context.Background())
	f.setSubs("alice", []model.LeetCodeSubmission{
		mkSub("200", "two-sum", daySecond(0, 12)),
		mkSub("100", "warmup", daySecond(-30, 0)),
	})
	p1.poll(context.Background())
	if len(tk.snapshotCalls()) != 1 {
		t.Fatalf("setup: expected 1 add, got %d", len(tk.snapshotCalls()))
	}

	// Simulate a process restart: brand new Poller instance, SAME store.
	p2 := newTestPoller(f, users, &fakeTasks{}, &fakeSender{}, st, Options{Enabled: true})
	p2.poll(context.Background()) // should be silent (watermark in DB)

	// Expect: nothing new because DB still has watermark "200".
	id, ok, _ := st.GetLastPolledSubmissionID(context.Background(), 1)
	if !ok || id != "200" {
		t.Errorf("watermark should survive restart, got id=%q ok=%v", id, ok)
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
