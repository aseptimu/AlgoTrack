package review

import (
	"context"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"strings"
	"time"

	"github.com/aseptimu/AlgoTrack/internal/catalog"
	"github.com/aseptimu/AlgoTrack/internal/model"
	"github.com/aseptimu/AlgoTrack/internal/service/recommend"
	"github.com/aseptimu/AlgoTrack/internal/timezone"
	tgbot "github.com/go-telegram/bot"
	tgbotmodels "github.com/go-telegram/bot/models"
)

// UserSource yields all tg_user rows for the daily reminder iteration.
type UserSource interface {
	AllUsers(ctx context.Context) ([]model.User, error)
}

// DueReviewSource returns due reviews for a single user.
type DueReviewSource interface {
	GetDueReviewsForUser(ctx context.Context, userID int64, nowTime time.Time) ([]model.DueReviewTask, error)
}

// Recommender picks fresh problem(s) for a user. The review service uses it
// to compose the morning bundle. *recommend.Service satisfies this.
//   - Next returns a single problem (used by /next).
//   - NextDailyBundle returns 1 problem normally, 2 if the first pick is Easy.
type Recommender interface {
	Next(ctx context.Context, userID int64, mode string) (*catalog.Problem, error)
	NextDailyBundle(ctx context.Context, userID int64, mode string) ([]catalog.Problem, error)
}

// MessageSender is the minimal Telegram bot surface needed to send the bundle.
// *github.com/go-telegram/bot.Bot satisfies this.
type MessageSender interface {
	SendMessage(ctx context.Context, params *tgbot.SendMessageParams) (*tgbotmodels.Message, error)
}

// MessageBundle is the data the review service hands to the formatter.
// Exposed for tests so they can inspect what would be sent without going
// through Telegram.
type MessageBundle struct {
	NewProblems []catalog.Problem
	Reviews     []model.DueReviewTask
}

// Empty reports whether the bundle has neither new problems nor reviews.
// Empty bundles still get a message — the "all clear" celebratory note —
// they just take a different formatter path.
func (b MessageBundle) Empty() bool {
	return len(b.NewProblems) == 0 && len(b.Reviews) == 0
}

type ReminderService struct {
	users       UserSource
	reviews     DueReviewSource
	recommender Recommender
	bot         MessageSender
	logger      *slog.Logger
	location    *time.Location
	nowFn       func() time.Time
}

func NewReminderService(
	users UserSource,
	reviews DueReviewSource,
	recommender Recommender,
	bot MessageSender,
	logger *slog.Logger,
) *ReminderService {
	if logger == nil {
		logger = slog.Default()
	}
	return &ReminderService{
		users:       users,
		reviews:     reviews,
		recommender: recommender,
		bot:         bot,
		logger:      logger,
		location:    timezone.MoscowLocation,
		nowFn:       time.Now,
	}
}

// Start runs the daily 9:00 MSK loop. Blocks until ctx is cancelled.
func (r *ReminderService) Start(ctx context.Context) {
	for {
		runAt := r.nextRun(r.nowFn())
		timer := time.NewTimer(time.Until(runAt))

		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			r.SendDailyBundles(ctx, runAt)
		}
	}
}

func (r *ReminderService) nextRun(now time.Time) time.Time {
	localNow := now.In(r.location)
	runAt := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 9, 0, 0, 0, r.location)
	if !localNow.Before(runAt) {
		runAt = runAt.AddDate(0, 0, 1)
	}
	return runAt
}

// SendDailyBundles iterates every user and sends a single laconic message:
// 1 (or 2 Easy) fresh problem(s) + the capped due-review list, or an
// "all clear" celebration when the user has nothing pending.
func (r *ReminderService) SendDailyBundles(ctx context.Context, runAt time.Time) {
	users, err := r.users.AllUsers(ctx)
	if err != nil {
		r.logger.Error("daily bundle: failed to load users", "err", err)
		return
	}

	for _, u := range users {
		bundle, err := r.BuildBundle(ctx, u, runAt.UTC())
		if err != nil {
			r.logger.Error("daily bundle: failed to build", "err", err, "userID", u.UserID)
			continue
		}

		text := FormatBundle(bundle, r.location)
		if _, err := r.bot.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID:             u.ChatID,
			Text:               text,
			ParseMode:          tgbotmodels.ParseModeHTML,
			LinkPreviewOptions: noPreview,
		}); err != nil {
			r.logger.Error("daily bundle: send failed", "err", err, "userID", u.UserID)
		}
	}
}

// BuildBundle composes one user's morning bundle: the capped due-review
// list and 1-or-2 fresh recommendations. Either or both may be empty —
// the formatter renders an all-clear note in that case.
func (r *ReminderService) BuildBundle(ctx context.Context, user model.User, asOf time.Time) (MessageBundle, error) {
	var b MessageBundle

	dues, err := r.reviews.GetDueReviewsForUser(ctx, user.UserID, asOf)
	if err != nil {
		return b, err
	}
	b.Reviews = CapByDifficulty(dues)

	mode := user.RecommendMode
	if mode == "" {
		mode = "default"
	}
	picks, err := r.recommender.NextDailyBundle(ctx, user.UserID, mode)
	if err != nil && !errors.Is(err, errCatalogExhausted) {
		// Don't fail the whole bundle just because the recommender is unhappy.
		r.logger.Warn("daily bundle: recommendation skipped", "err", err, "userID", user.UserID)
	} else {
		b.NewProblems = picks
	}

	return b, nil
}

// noPreview is the link-preview-disabled options pointer reused by every
// outgoing daily-bundle message. We never want big preview cards because
// every link is leetcode.com.
var noPreview = func() *tgbotmodels.LinkPreviewOptions {
	disabled := true
	return &tgbotmodels.LinkPreviewOptions{IsDisabled: &disabled}
}()

// errCatalogExhausted is an alias for the recommend package sentinel, kept
// here so the daily reminder treats exhaustion as a non-error skip.
var errCatalogExhausted = recommend.ErrCatalogExhausted

// FormatBundle renders the morning message. Kept laconic per product req.
// Three modes:
//   - all clear: empty bundle → celebratory note, nothing else
//   - new problems only: header + new-problems block
//   - new problems + reviews: header + both blocks + /add hint
func FormatBundle(b MessageBundle, location *time.Location) string {
	if b.Empty() {
		return "☀️ <b>Доброе утро!</b>\n\n🎉 На сегодня всё закрыто — повторений нет, новых задач не осталось. Можешь отдохнуть или попросить ещё через <code>/next</code>."
	}

	var sb strings.Builder
	sb.WriteString("☀️ <b>Доброе утро!</b>")

	if len(b.NewProblems) > 0 {
		header := "📌 <b>Новая задача</b>"
		if len(b.NewProblems) > 1 {
			header = "📌 <b>Новые задачи на сегодня</b>"
		}
		sb.WriteString("\n\n" + header)
		for _, p := range b.NewProblems {
			fmt.Fprintf(&sb,
				"\n<a href=\"%s\">#%d %s</a> [%s] · %s",
				html.EscapeString(p.Link()),
				p.Number,
				html.EscapeString(p.Title),
				html.EscapeString(p.Difficulty),
				html.EscapeString(p.Topic),
			)
		}
	}

	if len(b.Reviews) > 0 {
		sb.WriteString("\n\n🔁 <b>На повторение</b>")
		for i, t := range b.Reviews {
			title := t.Title
			if title == "" {
				title = fmt.Sprintf("Задача %d", t.TaskNumber)
			}
			line := fmt.Sprintf("<a href=\"%s\">#%d %s</a>", html.EscapeString(t.Link), t.TaskNumber, html.EscapeString(title))
			if t.Link == "" {
				line = fmt.Sprintf("#%d %s", t.TaskNumber, html.EscapeString(title))
			}
			diff := t.Difficulty
			if diff == "" {
				diff = "Easy"
			}
			fmt.Fprintf(&sb, "\n%d) %s [%s] · последнее %s",
				i+1, line, html.EscapeString(diff),
				t.LastReviewedAt.In(location).Format("02.01"),
			)
		}
		sb.WriteString("\n\n<i>Отметь повторение через</i> <code>/add номер</code>")
	}

	return sb.String()
}
