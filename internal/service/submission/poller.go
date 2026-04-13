package submission

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/aseptimu/AlgoTrack/internal/model"
	tgbot "github.com/go-telegram/bot"
	tgmodels "github.com/go-telegram/bot/models"
)

const (
	DefaultPollInterval     = 5 * time.Minute
	DefaultSubmissionsLimit = 10
)

// Options configures the submission poller. Zero values fall back to defaults.
type Options struct {
	Enabled          bool
	Interval         time.Duration
	SubmissionsLimit int
}

// MessageSender is the minimal Telegram bot surface the poller needs.
// *github.com/go-telegram/bot.Bot satisfies this interface.
type MessageSender interface {
	SendMessage(ctx context.Context, params *tgbot.SendMessageParams) (*tgmodels.Message, error)
}

// LeetCodeFetcher fetches recent accepted submissions from LeetCode.
type LeetCodeFetcher interface {
	GetRecentAcceptedSubmissions(ctx context.Context, username string, limit int) ([]model.LeetCodeSubmission, error)
}

// UserProvider returns users who have linked their LeetCode accounts.
type UserProvider interface {
	GetUsersWithLeetCode(ctx context.Context) ([]model.User, error)
}

// TaskAdder adds a task for a user.
type TaskAdder interface {
	Add(ctx context.Context, taskNumber int64, user *model.User) (*model.AddTaskResult, error)
}

// Poller periodically checks for new LeetCode submissions.
type Poller struct {
	fetcher LeetCodeFetcher
	users   UserProvider
	tasks   TaskAdder
	bot     MessageSender
	logger  *slog.Logger

	enabled          bool
	interval         time.Duration
	submissionsLimit int

	// lastSeen tracks the latest submission ID per user to avoid duplicates.
	mu       sync.Mutex
	lastSeen map[int64]string // userID -> last submission ID
}

// NewPoller constructs a Poller with default options (enabled, 5m interval, limit 10).
// Use NewPollerWithOptions to override.
func NewPoller(
	fetcher LeetCodeFetcher,
	users UserProvider,
	tasks TaskAdder,
	bot MessageSender,
	logger *slog.Logger,
) *Poller {
	return NewPollerWithOptions(fetcher, users, tasks, bot, logger, Options{
		Enabled:          true,
		Interval:         DefaultPollInterval,
		SubmissionsLimit: DefaultSubmissionsLimit,
	})
}

// NewPollerWithOptions constructs a Poller and applies opts; zero fields fall back to defaults.
func NewPollerWithOptions(
	fetcher LeetCodeFetcher,
	users UserProvider,
	tasks TaskAdder,
	bot MessageSender,
	logger *slog.Logger,
	opts Options,
) *Poller {
	if logger == nil {
		logger = slog.Default()
	}
	if opts.Interval <= 0 {
		opts.Interval = DefaultPollInterval
	}
	if opts.SubmissionsLimit <= 0 {
		opts.SubmissionsLimit = DefaultSubmissionsLimit
	}
	return &Poller{
		fetcher:          fetcher,
		users:            users,
		tasks:            tasks,
		bot:              bot,
		logger:           logger,
		enabled:          opts.Enabled,
		interval:         opts.Interval,
		submissionsLimit: opts.SubmissionsLimit,
		lastSeen:         make(map[int64]string),
	}
}

// Start runs the polling loop. It blocks until ctx is cancelled.
// If the poller is disabled, Start logs and returns immediately.
func (p *Poller) Start(ctx context.Context) {
	if !p.enabled {
		p.logger.Info("submission poller: disabled, not starting")
		return
	}

	// Do an initial poll after a short delay to populate lastSeen without sending notifications.
	initTimer := time.NewTimer(10 * time.Second)
	select {
	case <-ctx.Done():
		initTimer.Stop()
		return
	case <-initTimer.C:
		p.seedLastSeen(ctx)
	}

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.poll(ctx)
		}
	}
}

// seedLastSeen populates the last seen submissions without triggering notifications.
func (p *Poller) seedLastSeen(ctx context.Context) {
	users, err := p.users.GetUsersWithLeetCode(ctx)
	if err != nil {
		p.logger.Error("submission poller: failed to get users for seed", "err", err)
		return
	}

	for _, user := range users {
		if user.LeetCodeUsername == nil || *user.LeetCodeUsername == "" {
			continue
		}

		submissions, err := p.fetcher.GetRecentAcceptedSubmissions(ctx, *user.LeetCodeUsername, 1)
		if err != nil {
			p.logger.Warn("submission poller: failed to seed submissions", "err", err, "userID", user.UserID, "leetcode", *user.LeetCodeUsername)
			continue
		}

		if len(submissions) > 0 {
			p.mu.Lock()
			p.lastSeen[user.UserID] = submissions[0].ID
			p.mu.Unlock()
		}
	}

	p.logger.Info("submission poller: seed complete", "users", len(users))
}

// Poll runs a single poll cycle against all linked users and notifies on new
// accepted submissions. Exposed for test runners and one-shot invocations;
// the long-running loop in Start uses it internally.
func (p *Poller) Poll(ctx context.Context) { p.poll(ctx) }

func (p *Poller) poll(ctx context.Context) {
	users, err := p.users.GetUsersWithLeetCode(ctx)
	if err != nil {
		p.logger.Error("submission poller: failed to get users", "err", err)
		return
	}

	for _, user := range users {
		if user.LeetCodeUsername == nil || *user.LeetCodeUsername == "" {
			continue
		}

		p.checkUser(ctx, user)
	}
}

func (p *Poller) checkUser(ctx context.Context, user model.User) {
	submissions, err := p.fetcher.GetRecentAcceptedSubmissions(ctx, *user.LeetCodeUsername, p.submissionsLimit)
	if err != nil {
		p.logger.Warn("submission poller: failed to fetch submissions", "err", err, "userID", user.UserID, "leetcode", *user.LeetCodeUsername)
		return
	}

	if len(submissions) == 0 {
		return
	}

	p.mu.Lock()
	lastSeenID := p.lastSeen[user.UserID]
	p.mu.Unlock()

	// Find new submissions (those we haven't seen yet).
	var newSubmissions []model.LeetCodeSubmission
	for _, s := range submissions {
		if s.ID == lastSeenID {
			break
		}
		newSubmissions = append(newSubmissions, s)
	}

	if len(newSubmissions) == 0 {
		return
	}

	// Update last seen to the most recent.
	p.mu.Lock()
	p.lastSeen[user.UserID] = submissions[0].ID
	p.mu.Unlock()

	// Process new submissions in reverse order (oldest first).
	for i := len(newSubmissions) - 1; i >= 0; i-- {
		s := newSubmissions[i]
		p.processSubmission(ctx, user, s)
	}
}

func (p *Poller) processSubmission(ctx context.Context, user model.User, sub model.LeetCodeSubmission) {
	// Try to extract problem number from the submission.
	// LeetCode titleSlug doesn't directly give us the number,
	// so we'll add the task via the task service which will look it up.
	// For now, we notify the user and let them add it manually if auto-add fails.

	p.logger.Info("submission poller: new accepted submission",
		"userID", user.UserID,
		"title", sub.Title,
		"titleSlug", sub.TitleSlug,
		"submissionID", sub.ID,
	)

	ts, _ := strconv.ParseInt(sub.Timestamp, 10, 64)
	submittedAt := time.Unix(ts, 0)
	timeStr := submittedAt.In(time.FixedZone("MSK", 3*60*60)).Format("02.01.2006 15:04 MSK")

	title := html.EscapeString(sub.Title)
	link := fmt.Sprintf("https://leetcode.com/problems/%s/", sub.TitleSlug)

	msg := fmt.Sprintf(
		"🎉 <b>Новое решение на LeetCode!</b>\n\n"+
			"<b><a href=\"%s\">%s</a></b>\n"+
			"Время: <i>%s</i>\n\n"+
			"Добавь задачу через <code>/add номер</code> для отслеживания повторений.",
		html.EscapeString(link),
		title,
		timeStr,
	)

	if _, err := p.bot.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:    user.ChatID,
		Text:      msg,
		ParseMode: tgmodels.ParseModeHTML,
	}); err != nil {
		p.logger.Error("submission poller: failed to send notification", "err", err, "userID", user.UserID)
	}
}
