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

// Recommender picks a fresh problem for a user. The review service uses it
// to compose the morning bundle. *recommend.Service satisfies this.
type Recommender interface {
	Next(ctx context.Context, userID int64, mode string) (*catalog.Problem, error)
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
	Recommendation *catalog.Problem
	Reviews        []model.DueReviewTask
}

// Empty reports whether the bundle has nothing worth sending.
func (b MessageBundle) Empty() bool {
	return b.Recommendation == nil && len(b.Reviews) == 0
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

// SendDailyBundles iterates every user and sends a single laconic message
// containing one fresh recommendation plus the capped due-review list.
// Users with both empty are skipped silently.
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
		if bundle.Empty() {
			continue
		}

		text := FormatBundle(bundle, r.location)
		if _, err := r.bot.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID:    u.ChatID,
			Text:      text,
			ParseMode: tgbotmodels.ParseModeHTML,
		}); err != nil {
			r.logger.Error("daily bundle: send failed", "err", err, "userID", u.UserID)
		}
	}
}

// BuildBundle composes one user's morning bundle: the capped due-review
// list and one fresh recommendation. Either side may be empty.
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
	rec, err := r.recommender.Next(ctx, user.UserID, mode)
	if err != nil && !errors.Is(err, errCatalogExhausted) {
		// Don't fail the whole bundle just because the recommender is unhappy.
		r.logger.Warn("daily bundle: recommendation skipped", "err", err, "userID", user.UserID)
	} else if rec != nil {
		b.Recommendation = rec
	}

	return b, nil
}

// errCatalogExhausted is an alias for the recommend package sentinel, kept
// here so the daily reminder treats exhaustion as a non-error skip.
var errCatalogExhausted = recommend.ErrCatalogExhausted

// FormatBundle renders the morning message. Kept laconic per product req:
// header + recommendation block (if any) + capped review list (if any) +
// short hint about /add for marking repetitions.
func FormatBundle(b MessageBundle, location *time.Location) string {
	var sb strings.Builder
	sb.WriteString("☀️ <b>Доброе утро!</b>")

	if b.Recommendation != nil {
		p := b.Recommendation
		fmt.Fprintf(&sb,
			"\n\n📌 <b>Новая задача</b>\n<a href=\"%s\">#%d %s</a> [%s] · %s",
			html.EscapeString(p.Link()),
			p.Number,
			html.EscapeString(p.Title),
			html.EscapeString(p.Difficulty),
			html.EscapeString(p.Topic),
		)
	}

	if len(b.Reviews) > 0 {
		sb.WriteString("\n\n🔁 <b>На повторение</b>")
		for i, t := range b.Reviews {
			title := t.Title
			if title == "" {
				title = fmt.Sprintf("Task %d", t.TaskNumber)
			}
			line := fmt.Sprintf("<a href=\"%s\">#%d %s</a>", html.EscapeString(t.Link), t.TaskNumber, html.EscapeString(title))
			if t.Link == "" {
				line = fmt.Sprintf("#%d %s", t.TaskNumber, html.EscapeString(title))
			}
			diff := t.Difficulty
			if diff == "" {
				diff = "Easy"
			}
			fmt.Fprintf(&sb, "\n%d) %s [%s] · last %s",
				i+1, line, html.EscapeString(diff),
				t.LastReviewedAt.In(location).Format("02.01"),
			)
		}
		sb.WriteString("\n\n<i>Отметь повторение через</i> <code>/add номер</code>")
	}

	return sb.String()
}
