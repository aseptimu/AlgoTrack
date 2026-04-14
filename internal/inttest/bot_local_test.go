//go:build integration

// Package inttest contains opt-in integration tests that exercise the real
// bot stack (handlers, services, repos) against a real Postgres started via
// docker-compose, with the Telegram Bot API mocked by a local httptest
// server. These tests are skipped by default; run them with:
//
//	cd ~/work/AlgoTrack && docker compose up -d db && \
//	   docker compose run --rm migrate && \
//	   ALGOTRACK_INTTEST_DB=postgres://algo_app:algo_local_pw@localhost:6000/master?sslmode=disable \
//	   go test -tags=integration -count=1 -v ./internal/inttest/...
package inttest

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aseptimu/AlgoTrack/internal/db"
	"github.com/aseptimu/AlgoTrack/internal/model"
	taskrepo "github.com/aseptimu/AlgoTrack/internal/repo/task"
	userrepo "github.com/aseptimu/AlgoTrack/internal/repo/user"
	"github.com/aseptimu/AlgoTrack/internal/service/submission"
	tasksvc "github.com/aseptimu/AlgoTrack/internal/service/task"
	usersvc "github.com/aseptimu/AlgoTrack/internal/service/user"
	"github.com/aseptimu/AlgoTrack/internal/telegram"
	"github.com/aseptimu/AlgoTrack/internal/telegram/commands/add"
	"github.com/aseptimu/AlgoTrack/internal/telegram/commands/goal"
	helpcmd "github.com/aseptimu/AlgoTrack/internal/telegram/commands/help"
	"github.com/aseptimu/AlgoTrack/internal/telegram/commands/link"
	"github.com/aseptimu/AlgoTrack/internal/telegram/commands/list"
	"github.com/aseptimu/AlgoTrack/internal/telegram/commands/setgoal"
	"github.com/aseptimu/AlgoTrack/internal/telegram/commands/start"
	"github.com/aseptimu/AlgoTrack/internal/telegram/commands/stats"
	"github.com/aseptimu/AlgoTrack/internal/telegram/messages/fallback"
	"github.com/aseptimu/AlgoTrack/internal/telegram/router"

	tgbot "github.com/go-telegram/bot"
	tgmodels "github.com/go-telegram/bot/models"
)

const (
	testUserID = int64(424242)
	testChatID = int64(424242)
)

// fakeTelegramServer captures sendMessage calls and answers Bot API requests
// well enough for *tgbot.Bot to operate without talking to telegram.org.
type fakeTelegramServer struct {
	server *httptest.Server

	mu          sync.Mutex
	sendCalls   []sendMessageCall
	rawRequests []string
}

type sendMessageCall struct {
	ChatID    int64  `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

func newFakeTelegramServer(t *testing.T) *fakeTelegramServer {
	t.Helper()
	f := &fakeTelegramServer{}
	mux := http.NewServeMux()

	// /bot<token>/<method> — the bot lib appends method names like sendMessage.
	// Bot API requests are sent as multipart/form-data, not JSON.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/sendMessage"):
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				_ = r.ParseForm()
			}
			var chatID int64
			if v := r.FormValue("chat_id"); v != "" {
				chatID, _ = strconv.ParseInt(v, 10, 64)
			}
			text := r.FormValue("text")
			parseMode := r.FormValue("parse_mode")
			f.mu.Lock()
			f.sendCalls = append(f.sendCalls, sendMessageCall{ChatID: chatID, Text: text, ParseMode: parseMode})
			f.mu.Unlock()
			fakeMessage(w, chatID, text)
			return
		case strings.HasSuffix(r.URL.Path, "/getMe"):
			_, _ = io.WriteString(w, `{"ok":true,"result":{"id":1,"is_bot":true,"first_name":"AlgoTrackTest","username":"algotrackbot"}}`)
			return
		default:
			_, _ = io.WriteString(w, `{"ok":true,"result":true}`)
		}
	})

	f.server = httptest.NewServer(mux)
	t.Cleanup(f.server.Close)
	return f
}

func fakeMessage(w http.ResponseWriter, chatID int64, text string) {
	resp := map[string]any{
		"ok": true,
		"result": map[string]any{
			"message_id": time.Now().UnixNano() % 1_000_000,
			"chat":       map[string]any{"id": chatID, "type": "private"},
			"date":       time.Now().Unix(),
			"text":       text,
		},
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (f *fakeTelegramServer) sends() []sendMessageCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]sendMessageCall, len(f.sendCalls))
	copy(out, f.sendCalls)
	return out
}

func (f *fakeTelegramServer) reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sendCalls = nil
	f.rawRequests = nil
}

// fakeFetcher is a controllable LeetCode fetcher for poller scenarios.
type fakeFetcher struct {
	mu       sync.Mutex
	subs     map[string][]model.LeetCodeSubmission
	problems map[string]*model.ProblemInfo
	err      error
}

// GetProblemBySlug satisfies submission.LeetCodeFetcher so the poller can
// resolve a slug into a full ProblemInfo. Tests register problems via
// setProblem; the default returns a Stub with number 9999.
func (f *fakeFetcher) GetProblemBySlug(_ context.Context, slug string) (*model.ProblemInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if p, ok := f.problems[slug]; ok {
		return p, nil
	}
	return &model.ProblemInfo{Number: 9999, Title: slug, TitleSlug: slug, Difficulty: "Easy", Link: "https://leetcode.com/problems/" + slug + "/"}, nil
}

func (f *fakeFetcher) setProblem(slug string, p *model.ProblemInfo) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.problems == nil {
		f.problems = make(map[string]*model.ProblemInfo)
	}
	f.problems[slug] = p
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

// GetProblemByNumber satisfies task.ProblemProvider so the same fake can be
// passed as both the LeetCode client (for poller) and the problem catalog
// (for /add). Integration tests don't exercise /add so this stub is enough.
func (f *fakeFetcher) GetProblemByNumber(_ context.Context, number int64) (*model.ProblemInfo, error) {
	return &model.ProblemInfo{
		Number:    int(number),
		Title:     "Stub",
		TitleSlug: "stub",
		Link:      "https://leetcode.com/problems/stub/",
		Platform:  "leetcode",
	}, nil
}

func (f *fakeFetcher) set(username string, subs []model.LeetCodeSubmission) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.subs == nil {
		f.subs = make(map[string][]model.LeetCodeSubmission)
	}
	f.subs[username] = subs
}

// fixedClock returns deterministic timestamps for "today" and "tomorrow" MSK.
func tsMSK(dayOffset int, hour int) string {
	base := time.Date(2026, 4, 13, hour-3, 0, 0, 0, time.UTC) // hour MSK -> hour-3 UTC
	t := base.AddDate(0, 0, dayOffset)
	return strconv.FormatInt(t.Unix(), 10)
}

// testEnv wires together the full app stack against a real DB and a fake bot API.
type testEnv struct {
	ctx     context.Context
	db      *db.DB
	bot     *telegram.Bot
	tgFake  *fakeTelegramServer
	fetcher *fakeFetcher
	poller  *submission.Poller

	userSvc *usersvc.TgUserService
	taskSvc *tasksvc.TaskService
}

func setupEnv(t *testing.T) *testEnv {
	t.Helper()

	dbURL := os.Getenv("ALGOTRACK_INTTEST_DB")
	if dbURL == "" {
		t.Skip("ALGOTRACK_INTTEST_DB not set; skipping local integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	database, err := db.NewDB(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	t.Cleanup(func() { database.Pool.Close() })

	// Wipe ALL data so each test run is independent. Safe because this DB is
	// a local docker-compose volume gated by ALGOTRACK_INTTEST_DB env var and
	// never points at production.
	if _, err := database.Pool.Exec(ctx, `TRUNCATE TABLE algo_tasks, notified_problem, tg_user RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	tgUserRepo := userrepo.NewTgUserRepo(database)
	tRepo := taskrepo.NewTaskRepo(database)
	userSvc := usersvc.NewUserService(tgUserRepo, tRepo, logger)

	fetcher := &fakeFetcher{}
	taskSvc := tasksvc.NewTaskService(userSvc, tRepo, fetcher, logger)

	tgFake := newFakeTelegramServer(t)

	bot, err := telegram.New("test:fake-token", logger,
		tgbot.WithSkipGetMe(),
		tgbot.WithServerURL(tgFake.server.URL),
	)
	if err != nil {
		t.Fatalf("init bot: %v", err)
	}

	startHandler := start.New(userSvc, logger)
	addHandler := add.New(taskSvc, logger)
	helpHandler := helpcmd.New(logger)
	textHandler := fallback.New(logger)
	goalCallbackHandler := goal.New(userSvc, logger)
	setGoalHandler := setgoal.New(userSvc, logger)
	linkHandler := link.New(userSvc, logger)
	listHandler := list.New(taskSvc, userSvc, logger)
	statsHandler := stats.New(taskSvc, userSvc, logger)

	router.Register(bot.Raw(), router.Handlers{
		Start:        startHandler,
		Add:          addHandler,
		Help:         helpHandler,
		Text:         textHandler,
		GoalCallback: goalCallbackHandler,
		SetGoal:      setGoalHandler,
		Link:         linkHandler,
		List:         listHandler,
		Stats:        statsHandler,
	})

	poller := submission.NewPollerWithOptions(
		fetcher,
		userSvc,
		taskSvc,
		tgUserRepo,
		bot.Raw(),
		logger,
		submission.Options{Enabled: true, Interval: 100 * time.Millisecond, SubmissionsLimit: 10},
	)

	return &testEnv{
		ctx:     ctx,
		db:      database,
		bot:     bot,
		tgFake:  tgFake,
		fetcher: fetcher,
		poller:  poller,
		userSvc: userSvc,
		taskSvc: taskSvc,
	}
}

func (e *testEnv) sendCommand(text string) {
	upd := &tgmodels.Update{
		ID: time.Now().UnixNano() % 1_000_000,
		Message: &tgmodels.Message{
			ID:   1,
			Date: int(time.Now().Unix()),
			From: &tgmodels.User{ID: testUserID, Username: "inttester", FirstName: "Int"},
			Chat: tgmodels.Chat{ID: testChatID, Type: tgmodels.ChatTypePrivate},
			Text: text,
		},
	}
	e.bot.Raw().ProcessUpdate(e.ctx, upd)
	// Tiny grace period for async handler dispatch inside the bot lib.
	time.Sleep(150 * time.Millisecond)
}

func (e *testEnv) leetcodeUsernameInDB(t *testing.T) string {
	t.Helper()
	var lc *string
	err := e.db.Pool.QueryRow(e.ctx, `SELECT leetcode_username FROM tg_user WHERE user_id = $1`, testUserID).Scan(&lc)
	if errors.Is(err, context.Canceled) {
		t.Fatal("ctx cancelled")
	}
	if err != nil {
		return ""
	}
	if lc == nil {
		return ""
	}
	return *lc
}

// --- scenarios ---

func TestStartCreatesUserAndReplies(t *testing.T) {
	e := setupEnv(t)

	e.sendCommand("/start")

	sends := e.tgFake.sends()
	if len(sends) == 0 {
		t.Fatal("/start should produce a reply")
	}
	if sends[0].ChatID != testChatID {
		t.Errorf("reply chat = %d, want %d", sends[0].ChatID, testChatID)
	}

	var count int
	if err := e.db.Pool.QueryRow(e.ctx, `SELECT COUNT(*) FROM tg_user WHERE user_id = $1`, testUserID).Scan(&count); err != nil {
		t.Fatalf("query user: %v", err)
	}
	if count != 1 {
		t.Fatalf("user row count = %d, want 1", count)
	}
}

func TestLinkPersistsLeetCodeUsername(t *testing.T) {
	e := setupEnv(t)

	e.sendCommand("/start")
	e.tgFake.reset()
	e.sendCommand("/link lee215")

	if got := e.leetcodeUsernameInDB(t); got != "lee215" {
		t.Errorf("DB leetcode_username = %q, want %q", got, "lee215")
	}
	sends := e.tgFake.sends()
	if len(sends) == 0 || !strings.Contains(sends[0].Text, "lee215") {
		t.Errorf("link reply should mention lee215, got %+v", sends)
	}
}

func TestLinkRejectsInvalidUsername(t *testing.T) {
	e := setupEnv(t)

	e.sendCommand("/start")
	e.tgFake.reset()
	e.sendCommand("/link bad name with spaces") // /link expects exactly one arg, this is rejected

	if got := e.leetcodeUsernameInDB(t); got != "" {
		t.Errorf("invalid /link should not persist; got %q", got)
	}
	sends := e.tgFake.sends()
	if len(sends) == 0 || !strings.Contains(sends[0].Text, "/link") {
		t.Errorf("invalid /link should explain usage, got %+v", sends)
	}
}

func TestStatsRespondsForFreshUser(t *testing.T) {
	e := setupEnv(t)
	e.sendCommand("/start")
	e.tgFake.reset()
	e.sendCommand("/stats")

	sends := e.tgFake.sends()
	if len(sends) == 0 {
		t.Fatal("/stats should reply")
	}
	if !strings.Contains(sends[0].Text, "Решено") {
		t.Errorf("stats reply should mention 'Solved', got %q", sends[0].Text)
	}
}

func TestListRespondsForFreshUser(t *testing.T) {
	e := setupEnv(t)
	e.sendCommand("/start")
	e.tgFake.reset()
	e.sendCommand("/list")

	if len(e.tgFake.sends()) == 0 {
		t.Fatal("/list should reply (even when empty)")
	}
}

func setSubs(f *fakeFetcher, username string, subs []model.LeetCodeSubmission) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.subs == nil {
		f.subs = make(map[string][]model.LeetCodeSubmission)
	}
	f.subs[username] = subs
}

func (e *testEnv) countTasks(t *testing.T, taskNumber int) int64 {
	t.Helper()
	var n int64
	if err := e.db.Pool.QueryRow(e.ctx,
		`SELECT COALESCE(review_count, 0) FROM algo_tasks WHERE user_id = $1 AND task_number = $2`,
		testUserID, taskNumber).Scan(&n); err != nil {
		return 0
	}
	return n
}

func TestPollerEndToEnd_AutoAddAndPersist(t *testing.T) {
	e := setupEnv(t)
	e.sendCommand("/start")
	e.sendCommand("/link lee215")
	e.tgFake.reset()

	// Register problem metadata so GetProblemBySlug returns the right numbers.
	e.fetcher.setProblem("two-sum", &model.ProblemInfo{Number: 1, Title: "Two Sum", TitleSlug: "two-sum", Difficulty: "Easy", Link: "https://leetcode.com/problems/two-sum/"})
	e.fetcher.setProblem("valid-parentheses", &model.ProblemInfo{Number: 20, Title: "Valid Parentheses", TitleSlug: "valid-parentheses", Difficulty: "Easy", Link: "https://leetcode.com/problems/valid-parentheses/"})

	// First poll: cold cache → silently absorb whatever lee215 has on top.
	setSubs(e.fetcher, "lee215", []model.LeetCodeSubmission{
		{ID: "100", Title: "warmup", TitleSlug: "warmup", Timestamp: tsMSK(-30, 0)},
	})
	e.poller.Poll(e.ctx)
	if got := e.tgFake.sends(); len(got) != 0 {
		t.Fatalf("first poll must be silent, got %d sends", len(got))
	}
	if e.countTasks(t, 9999) != 0 {
		t.Errorf("first poll must not auto-add, got rows for warmup")
	}

	// Day 0: two distinct problems plus a same-day repeat of two-sum.
	setSubs(e.fetcher, "lee215", []model.LeetCodeSubmission{
		{ID: "303", Title: "Valid Parentheses", TitleSlug: "valid-parentheses", Timestamp: tsMSK(0, 18)},
		{ID: "302", Title: "Two Sum", TitleSlug: "two-sum", Timestamp: tsMSK(0, 14)},
		{ID: "301", Title: "Two Sum", TitleSlug: "two-sum", Timestamp: tsMSK(0, 12)},
		{ID: "100", Title: "warmup", TitleSlug: "warmup", Timestamp: tsMSK(-30, 0)},
	})
	e.poller.Poll(e.ctx)
	time.Sleep(150 * time.Millisecond)

	sends := e.tgFake.sends()
	if len(sends) != 2 {
		t.Fatalf("expected 2 notifications (same-day two-sum dedup'd), got %d", len(sends))
	}
	if e.countTasks(t, 1) != 1 {
		t.Errorf("two-sum should be auto-added exactly once, review_count = %d", e.countTasks(t, 1))
	}
	if e.countTasks(t, 20) != 1 {
		t.Errorf("valid-parentheses should be auto-added exactly once, review_count = %d", e.countTasks(t, 20))
	}

	// Same-day repeat poll → silence + no extra rows.
	e.tgFake.reset()
	setSubs(e.fetcher, "lee215", []model.LeetCodeSubmission{
		{ID: "304", Title: "Two Sum", TitleSlug: "two-sum", Timestamp: tsMSK(0, 22)},
		{ID: "303", Title: "Valid Parentheses", TitleSlug: "valid-parentheses", Timestamp: tsMSK(0, 18)},
	})
	e.poller.Poll(e.ctx)
	if got := e.tgFake.sends(); len(got) != 0 {
		t.Errorf("same-day repeat must be silent, got %d sends", len(got))
	}
	if e.countTasks(t, 1) != 1 {
		t.Errorf("same-day repeat must not increment review_count, got %d", e.countTasks(t, 1))
	}

	// Next-day repeat of two-sum → fires 1 notification AND increments review_count to 2.
	e.tgFake.reset()
	setSubs(e.fetcher, "lee215", []model.LeetCodeSubmission{
		{ID: "401", Title: "Two Sum", TitleSlug: "two-sum", Timestamp: tsMSK(1, 12)},
		{ID: "304", Title: "Two Sum", TitleSlug: "two-sum", Timestamp: tsMSK(0, 22)},
	})
	e.poller.Poll(e.ctx)
	time.Sleep(150 * time.Millisecond)
	if got := e.tgFake.sends(); len(got) != 1 {
		t.Errorf("next-day repeat should fire 1 notification, got %d", len(got))
	}
	if e.countTasks(t, 1) != 2 {
		t.Errorf("next-day repeat must bump review_count to 2, got %d", e.countTasks(t, 1))
	}
}

func TestPoller_StatePersistsAcrossPollerInstance(t *testing.T) {
	e := setupEnv(t)
	e.sendCommand("/start")
	e.sendCommand("/link lee215")
	e.tgFake.reset()

	e.fetcher.setProblem("two-sum", &model.ProblemInfo{Number: 1, Title: "Two Sum", TitleSlug: "two-sum", Difficulty: "Easy", Link: "x"})
	setSubs(e.fetcher, "lee215", []model.LeetCodeSubmission{
		{ID: "100", Title: "warmup", TitleSlug: "warmup", Timestamp: tsMSK(-30, 0)},
	})
	e.poller.Poll(e.ctx) // absorb watermark "100"

	setSubs(e.fetcher, "lee215", []model.LeetCodeSubmission{
		{ID: "200", Title: "Two Sum", TitleSlug: "two-sum", Timestamp: tsMSK(0, 12)},
		{ID: "100", Title: "warmup", TitleSlug: "warmup", Timestamp: tsMSK(-30, 0)},
	})
	e.poller.Poll(e.ctx)
	time.Sleep(100 * time.Millisecond)
	if len(e.tgFake.sends()) != 1 {
		t.Fatalf("setup: want 1 notification, got %d", len(e.tgFake.sends()))
	}

	// Build a fresh Poller pointing at the SAME database (simulates restart).
	p2 := submission.NewPollerWithOptions(
		e.fetcher,
		e.userSvc,
		e.taskSvc,
		userrepo.NewTgUserRepo(e.db),
		e.bot.Raw(),
		newSilentLoggerInt(),
		submission.Options{Enabled: true, Interval: 100 * time.Millisecond, SubmissionsLimit: 10},
	)
	e.tgFake.reset()
	p2.Poll(e.ctx) // same submissions, same DB, watermark already at 200
	if got := e.tgFake.sends(); len(got) != 0 {
		t.Errorf("restart must not re-fire notifications; got %d sends", len(got))
	}
}

func newSilentLoggerInt() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
