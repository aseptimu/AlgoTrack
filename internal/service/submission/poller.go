package submission

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"strconv"
	"time"

	"github.com/aseptimu/AlgoTrack/internal/model"
	"github.com/aseptimu/AlgoTrack/internal/timezone"
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

// LeetCodeFetcher fetches recent accepted submissions and resolves slugs
// into full problem info. Both methods are served by *client.HTTPLeetCodeClient.
type LeetCodeFetcher interface {
	GetRecentAcceptedSubmissions(ctx context.Context, username string, limit int) ([]model.LeetCodeSubmission, error)
	GetProblemBySlug(ctx context.Context, slug string) (*model.ProblemInfo, error)
}

// UserProvider returns users who have linked their LeetCode accounts.
type UserProvider interface {
	GetUsersWithLeetCode(ctx context.Context) ([]model.User, error)
}

// TaskUpserter adds (or increments review_count for) a task by a pre-resolved
// problem info. Implemented by *task.TaskService.AddByProblem.
type TaskUpserter interface {
	AddByProblem(ctx context.Context, user *model.User, problem *model.ProblemInfo) (*model.AddTaskResult, error)
}

// StateStore persists per-user poller state across restarts:
//   - last accepted submission id we have inspected (the watermark)
//   - per-(user, problem, day) marker so same-day repeats are suppressed
type StateStore interface {
	GetLastPolledSubmissionID(ctx context.Context, userID int64) (string, bool, error)
	SetLastPolledSubmissionID(ctx context.Context, userID int64, submissionID string) error
	WasNotifiedToday(ctx context.Context, userID int64, titleSlug, day string) (bool, error)
	MarkNotified(ctx context.Context, userID int64, titleSlug, day string) error
}

// Poller periodically checks for new LeetCode submissions, auto-adds them as
// tasks, and notifies the user. State is persisted in the StateStore so
// behavior is deterministic across restarts and deploys.
type Poller struct {
	fetcher LeetCodeFetcher
	users   UserProvider
	tasks   TaskUpserter
	state   StateStore
	bot     MessageSender
	logger  *slog.Logger

	enabled          bool
	interval         time.Duration
	submissionsLimit int
}

// NewPoller constructs a Poller with default options (enabled, 5m interval, limit 10).
func NewPoller(
	fetcher LeetCodeFetcher,
	users UserProvider,
	tasks TaskUpserter,
	state StateStore,
	bot MessageSender,
	logger *slog.Logger,
) *Poller {
	return NewPollerWithOptions(fetcher, users, tasks, state, bot, logger, Options{
		Enabled:          true,
		Interval:         DefaultPollInterval,
		SubmissionsLimit: DefaultSubmissionsLimit,
	})
}

// NewPollerWithOptions constructs a Poller and applies opts; zero fields fall back to defaults.
func NewPollerWithOptions(
	fetcher LeetCodeFetcher,
	users UserProvider,
	tasks TaskUpserter,
	state StateStore,
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
		state:            state,
		bot:              bot,
		logger:           logger,
		enabled:          opts.Enabled,
		interval:         opts.Interval,
		submissionsLimit: opts.SubmissionsLimit,
	}
}

// Start runs the polling loop until ctx is cancelled. If the poller is
// disabled, Start logs and returns immediately.
func (p *Poller) Start(ctx context.Context) {
	if !p.enabled {
		p.logger.Info("submission poller: disabled, not starting")
		return
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

// Poll runs a single poll cycle against all linked users. Exposed for test
// runners; the long-running loop in Start uses it internally.
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

	lastSeenID, hasWatermark, err := p.state.GetLastPolledSubmissionID(ctx, user.UserID)
	if err != nil {
		p.logger.Error("submission poller: failed to get watermark", "err", err, "userID", user.UserID)
		return
	}

	// First poll for this user (e.g. linked after the bot started, or fresh
	// install). Silently absorb the current top as the watermark; do not flood
	// the user with notifications about old submissions.
	if !hasWatermark {
		if err := p.state.SetLastPolledSubmissionID(ctx, user.UserID, submissions[0].ID); err != nil {
			p.logger.Error("submission poller: failed to absorb watermark", "err", err, "userID", user.UserID)
			return
		}
		p.logger.Info("submission poller: first-poll absorb",
			"userID", user.UserID,
			"leetcode", *user.LeetCodeUsername,
			"watermark", submissions[0].ID,
		)
		return
	}

	// Find new submissions (those above the watermark, newest-first).
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

	// Advance watermark to the freshest seen id BEFORE processing, so a
	// failure to process one submission does not cause it to be re-emitted
	// on the next poll.
	if err := p.state.SetLastPolledSubmissionID(ctx, user.UserID, submissions[0].ID); err != nil {
		p.logger.Error("submission poller: failed to advance watermark", "err", err, "userID", user.UserID)
		return
	}

	// Process oldest-first so notifications arrive in chronological order.
	for i := len(newSubmissions) - 1; i >= 0; i-- {
		p.processSubmission(ctx, user, newSubmissions[i])
	}
}

func (p *Poller) processSubmission(ctx context.Context, user model.User, sub model.LeetCodeSubmission) {
	ts, _ := strconv.ParseInt(sub.Timestamp, 10, 64)
	submittedAt := time.Unix(ts, 0).In(timezone.MoscowLocation)
	day := submittedAt.Format("2006-01-02")

	already, err := p.state.WasNotifiedToday(ctx, user.UserID, sub.TitleSlug, day)
	if err != nil {
		p.logger.Error("submission poller: failed to check cooldown", "err", err, "userID", user.UserID)
		return
	}
	if already {
		p.logger.Info("submission poller: same-day repeat, suppressed",
			"userID", user.UserID,
			"titleSlug", sub.TitleSlug,
			"day", day,
		)
		return
	}

	// Resolve slug → full problem info (number, difficulty, link).
	problem, err := p.fetcher.GetProblemBySlug(ctx, sub.TitleSlug)
	if err != nil {
		p.logger.Warn("submission poller: failed to resolve problem by slug",
			"err", err,
			"titleSlug", sub.TitleSlug,
			"userID", user.UserID,
		)
		return
	}

	// Auto-upsert: creates the task on first solve, increments review_count
	// on subsequent days. The same-day cooldown above prevents accidental
	// double-counting if LeetCode reports the same problem twice in one day.
	result, err := p.tasks.AddByProblem(ctx, &user, problem)
	if err != nil {
		p.logger.Error("submission poller: failed to auto-add task",
			"err", err,
			"taskNumber", problem.Number,
			"userID", user.UserID,
		)
		return
	}

	p.logger.Info("submission poller: auto-added task",
		"userID", user.UserID,
		"taskNumber", problem.Number,
		"title", problem.Title,
		"isReview", result.IsReview,
		"reviewCount", result.Task.ReviewCount,
	)

	if err := p.sendNotification(ctx, user, submittedAt, problem, result); err != nil {
		p.logger.Error("submission poller: failed to send notification", "err", err, "userID", user.UserID)
		return
	}

	// Mark processed only after a successful send so a failed Telegram delivery
	// does not silently swallow the day.
	if err := p.state.MarkNotified(ctx, user.UserID, sub.TitleSlug, day); err != nil {
		p.logger.Error("submission poller: failed to mark notified", "err", err, "userID", user.UserID)
	}
}

func (p *Poller) sendNotification(
	ctx context.Context,
	user model.User,
	submittedAt time.Time,
	problem *model.ProblemInfo,
	result *model.AddTaskResult,
) error {
	timeStr := submittedAt.Format("02.01.2006 15:04 MSK")
	title := html.EscapeString(problem.Title)
	link := html.EscapeString(problem.Link)
	difficulty := html.EscapeString(problem.Difficulty)

	header := "🎉 <b>Новое решение на LeetCode!</b>"
	statusLine := fmt.Sprintf("✅ <b>Задача #%d добавлена</b>", problem.Number)
	if result.IsReview {
		statusLine = fmt.Sprintf("🔁 <b>Повторение #%d</b> (раз №%d)", problem.Number, result.Task.ReviewCount)
	}

	nextReview := ""
	if result.Task.NextReviewAt != nil {
		nextReview = "\nСледующее повторение: <i>" + html.EscapeString(
			result.Task.NextReviewAt.In(timezone.MoscowLocation).Format("02.01.2006 15:04 MSK"),
		) + "</i>"
	}

	msg := fmt.Sprintf(
		"%s\n\n<b><a href=\"%s\">%s</a></b> [%s]\nВремя: <i>%s</i>\n\n%s%s",
		header, link, title, difficulty, timeStr, statusLine, nextReview,
	)

	_, err := p.bot.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:    user.ChatID,
		Text:      msg,
		ParseMode: tgmodels.ParseModeHTML,
	})
	return err
}
