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
	mu   sync.Mutex
	subs map[string][]model.LeetCodeSubmission // username -> submissions, newest-first
	err  error
}

func (f *fakeFetcher) GetRecentAcceptedSubmissions(_ context.Context, username string, limit int) ([]model.LeetCodeSubmission, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	all := f.subs[username]
	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}
	out := make([]model.LeetCodeSubmission, len(all))
	copy(out, all)
	return out, nil
}

func (f *fakeFetcher) set(username string, subs []model.LeetCodeSubmission) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.subs == nil {
		f.subs = make(map[string][]model.LeetCodeSubmission)
	}
	f.subs[username] = subs
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

type noopTasks struct{}

func (noopTasks) Add(_ context.Context, _ int64, _ *model.User) (*model.AddTaskResult, error) {
	return nil, nil
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

func newTestPoller(f *fakeFetcher, u *fakeUsers, s *fakeSender, opts Options) *Poller {
	return NewPollerWithOptions(f, u, noopTasks{}, s, newSilentLogger(), opts)
}

// --- tests ---

func TestNewPoller_DefaultsApplied(t *testing.T) {
	p := NewPoller(&fakeFetcher{}, &fakeUsers{}, noopTasks{}, &fakeSender{}, nil)
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

func TestNewPollerWithOptions_FallbackOnZero(t *testing.T) {
	p := NewPollerWithOptions(&fakeFetcher{}, &fakeUsers{}, noopTasks{}, &fakeSender{}, nil, Options{Enabled: true})
	if p.interval != DefaultPollInterval || p.submissionsLimit != DefaultSubmissionsLimit {
		t.Errorf("zero options should fall back to defaults; got interval=%v limit=%d", p.interval, p.submissionsLimit)
	}
}

func TestStart_DisabledReturnsImmediately(t *testing.T) {
	p := newTestPoller(&fakeFetcher{}, &fakeUsers{}, &fakeSender{}, Options{Enabled: false})
	done := make(chan struct{})
	go func() { p.Start(context.Background()); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Start with disabled poller did not return")
	}
}

func TestSeedLastSeen_DoesNotSend(t *testing.T) {
	f := &fakeFetcher{}
	f.set("alice", []model.LeetCodeSubmission{mkSub("100", "two-sum", 1700000000)})
	u := &fakeUsers{users: []model.User{mkUser(1, "alice")}}
	s := &fakeSender{}
	p := newTestPoller(f, u, s, Options{Enabled: true})

	p.seedLastSeen(context.Background())

	if got := s.snapshot(); len(got) != 0 {
		t.Errorf("seed should not send messages; got %d", len(got))
	}
	if p.lastSeen[1] != "100" {
		t.Errorf("seed lastSeen = %q, want %q", p.lastSeen[1], "100")
	}
}

func TestPoll_NewSubmissionTriggersSend(t *testing.T) {
	f := &fakeFetcher{}
	u := &fakeUsers{users: []model.User{mkUser(1, "alice")}}
	s := &fakeSender{}
	p := newTestPoller(f, u, s, Options{Enabled: true})

	// seed
	f.set("alice", []model.LeetCodeSubmission{mkSub("100", "two-sum", 1700000000)})
	p.seedLastSeen(context.Background())

	// new submission appears (newer first)
	f.set("alice", []model.LeetCodeSubmission{
		mkSub("101", "add-two-numbers", 1700000100),
		mkSub("100", "two-sum", 1700000000),
	})
	p.poll(context.Background())

	sent := s.snapshot()
	if len(sent) != 1 {
		t.Fatalf("want 1 send, got %d", len(sent))
	}
	if p.lastSeen[1] != "101" {
		t.Errorf("lastSeen = %q, want %q", p.lastSeen[1], "101")
	}
}

func TestPoll_DedupSameSubmission(t *testing.T) {
	f := &fakeFetcher{}
	u := &fakeUsers{users: []model.User{mkUser(1, "alice")}}
	s := &fakeSender{}
	p := newTestPoller(f, u, s, Options{Enabled: true})

	f.set("alice", []model.LeetCodeSubmission{mkSub("100", "two-sum", 1700000000)})
	p.seedLastSeen(context.Background())
	p.poll(context.Background())
	p.poll(context.Background())

	if got := s.snapshot(); len(got) != 0 {
		t.Errorf("dedup failed: %d messages sent for unchanged submissions", len(got))
	}
}

func TestPoll_MultipleNewSubmissionsOldestFirst(t *testing.T) {
	f := &fakeFetcher{}
	u := &fakeUsers{users: []model.User{mkUser(1, "alice")}}
	s := &fakeSender{}
	p := newTestPoller(f, u, s, Options{Enabled: true})

	f.set("alice", []model.LeetCodeSubmission{mkSub("100", "two-sum", 1700000000)})
	p.seedLastSeen(context.Background())

	// Three new submissions appear, newer first.
	f.set("alice", []model.LeetCodeSubmission{
		mkSub("103", "z-third", 1700000300),
		mkSub("102", "y-second", 1700000200),
		mkSub("101", "x-first", 1700000100),
		mkSub("100", "two-sum", 1700000000),
	})
	p.poll(context.Background())

	sent := s.snapshot()
	if len(sent) != 3 {
		t.Fatalf("want 3 sends, got %d", len(sent))
	}
	// Oldest first: x-first, y-second, z-third
	wantOrder := []string{"x-first", "y-second", "z-third"}
	for i, w := range wantOrder {
		if got := sent[i].Text; !contains(got, w) {
			t.Errorf("send[%d] = %q, want substring %q", i, got, w)
		}
	}
}

func TestPoll_EmptyUsernameSkipped(t *testing.T) {
	f := &fakeFetcher{}
	u := &fakeUsers{users: []model.User{{UserID: 1, ChatID: 10, LeetCodeUsername: ptr("")}}}
	s := &fakeSender{}
	p := newTestPoller(f, u, s, Options{Enabled: true})

	p.poll(context.Background())

	if got := s.snapshot(); len(got) != 0 {
		t.Errorf("empty username should be skipped, got %d sends", len(got))
	}
}

func TestPoll_FetcherErrorDoesNotPanic(t *testing.T) {
	f := &fakeFetcher{err: errors.New("network down")}
	u := &fakeUsers{users: []model.User{mkUser(1, "alice")}}
	s := &fakeSender{}
	p := newTestPoller(f, u, s, Options{Enabled: true})

	p.poll(context.Background()) // must not panic
	if got := s.snapshot(); len(got) != 0 {
		t.Errorf("fetcher error should not produce sends, got %d", len(got))
	}
}

func TestPoll_UserProviderErrorHandled(t *testing.T) {
	f := &fakeFetcher{}
	u := &fakeUsers{err: errors.New("db down")}
	s := &fakeSender{}
	p := newTestPoller(f, u, s, Options{Enabled: true})

	p.poll(context.Background())
	if got := s.snapshot(); len(got) != 0 {
		t.Errorf("user provider error should not produce sends, got %d", len(got))
	}
}

// daySecond returns a Unix timestamp at the given hour MSK on day=base+offset,
// where the base is 2026-04-13 in MSK. Used to keep test data calendar-aware.
// MSK is UTC+3 with no DST.
func daySecond(dayOffset int, hourMSK int) int64 {
	base := time.Date(2026, 4, 13, hourMSK-3, 0, 0, 0, time.UTC)
	return base.AddDate(0, 0, dayOffset).Unix()
}

func TestPoll_SameDayRepeatSuppressed(t *testing.T) {
	f := &fakeFetcher{}
	u := &fakeUsers{users: []model.User{mkUser(1, "alice")}}
	s := &fakeSender{}
	p := newTestPoller(f, u, s, Options{Enabled: true})

	f.set("alice", []model.LeetCodeSubmission{mkSub("100", "two-sum", daySecond(0, 12))})
	p.poll(context.Background())

	// Same problem, 3 hours later, same calendar day MSK.
	f.set("alice", []model.LeetCodeSubmission{
		mkSub("101", "two-sum", daySecond(0, 15)),
		mkSub("100", "two-sum", daySecond(0, 12)),
	})
	p.poll(context.Background())

	if got := s.snapshot(); len(got) != 1 {
		t.Fatalf("want exactly 1 send (same-day repeat suppressed), got %d", len(got))
	}
}

func TestPoll_NextDayRepeatNotifies(t *testing.T) {
	f := &fakeFetcher{}
	u := &fakeUsers{users: []model.User{mkUser(1, "alice")}}
	s := &fakeSender{}
	p := newTestPoller(f, u, s, Options{Enabled: true})

	f.set("alice", []model.LeetCodeSubmission{mkSub("100", "two-sum", daySecond(0, 12))})
	p.poll(context.Background())

	// Same problem on the next calendar day.
	f.set("alice", []model.LeetCodeSubmission{
		mkSub("200", "two-sum", daySecond(1, 12)),
		mkSub("100", "two-sum", daySecond(0, 12)),
	})
	p.poll(context.Background())

	if got := s.snapshot(); len(got) != 2 {
		t.Fatalf("want 2 sends across two calendar days, got %d", len(got))
	}
}

func TestPoll_DifferentProblemsSameDayBothNotify(t *testing.T) {
	f := &fakeFetcher{}
	u := &fakeUsers{users: []model.User{mkUser(1, "alice")}}
	s := &fakeSender{}
	p := newTestPoller(f, u, s, Options{Enabled: true})

	f.set("alice", []model.LeetCodeSubmission{
		mkSub("101", "add-two-numbers", daySecond(0, 13)),
		mkSub("100", "two-sum", daySecond(0, 12)),
	})
	p.poll(context.Background())

	if got := s.snapshot(); len(got) != 2 {
		t.Fatalf("two distinct problems same day should both notify, got %d", len(got))
	}
}

func TestPoll_SameDayRepeatDoesNotBlockOtherProblems(t *testing.T) {
	f := &fakeFetcher{}
	u := &fakeUsers{users: []model.User{mkUser(1, "alice")}}
	s := &fakeSender{}
	p := newTestPoller(f, u, s, Options{Enabled: true})

	// First poll: notify on two-sum.
	f.set("alice", []model.LeetCodeSubmission{mkSub("100", "two-sum", daySecond(0, 12))})
	p.poll(context.Background())

	// Second poll: same-day two-sum repeat AND a brand-new problem same day.
	// The new problem must still notify; the repeat must be suppressed.
	f.set("alice", []model.LeetCodeSubmission{
		mkSub("102", "valid-parentheses", daySecond(0, 18)),
		mkSub("101", "two-sum", daySecond(0, 15)),
		mkSub("100", "two-sum", daySecond(0, 12)),
	})
	p.poll(context.Background())

	sent := s.snapshot()
	if len(sent) != 2 {
		t.Fatalf("want 2 sends total (orig two-sum + valid-parentheses), got %d", len(sent))
	}
	if !contains(sent[0].Text, "two-sum") {
		t.Errorf("first send should be two-sum, got %q", sent[0].Text)
	}
	if !contains(sent[1].Text, "valid-parentheses") {
		t.Errorf("second send should be valid-parentheses, got %q", sent[1].Text)
	}
}

func TestPoll_FailedSendDoesNotPoisonCooldown(t *testing.T) {
	f := &fakeFetcher{}
	u := &fakeUsers{users: []model.User{mkUser(1, "alice")}}
	s := &fakeSender{err: errors.New("telegram down")}
	p := newTestPoller(f, u, s, Options{Enabled: true})

	f.set("alice", []model.LeetCodeSubmission{mkSub("100", "two-sum", daySecond(0, 12))})
	p.poll(context.Background())

	// Telegram came back. The same submission ID is no longer "new" (lastSeen advanced),
	// but a fresh submission of the same problem same day should now succeed and notify
	// because the previous attempt failed and the cooldown was not set.
	s.mu.Lock()
	s.err = nil
	s.mu.Unlock()
	f.set("alice", []model.LeetCodeSubmission{
		mkSub("101", "two-sum", daySecond(0, 15)),
		mkSub("100", "two-sum", daySecond(0, 12)),
	})
	p.poll(context.Background())

	if got := s.snapshot(); len(got) != 1 {
		t.Fatalf("after a failed send the cooldown must not be set; want 1 successful send, got %d", len(got))
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
