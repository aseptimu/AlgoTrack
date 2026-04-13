// Command poller-smoke runs the LeetCode submission poller end-to-end against
// a real database and the real LeetCode GraphQL API, but with a stub Telegram
// sender that prints messages to stdout. It is meant for local manual testing
// only and is never built or shipped to production.
//
// Usage:
//
//	DATABASE_URL=postgres://algo_app:algo_local_pw@localhost:6000/master?sslmode=disable \
//	POLLER_INTERVAL=15s POLLER_SUBMISSIONS_LIMIT=5 \
//	go run ./cmd/poller-smoke
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aseptimu/AlgoTrack/internal/client"
	"github.com/aseptimu/AlgoTrack/internal/db"
	taskrepo "github.com/aseptimu/AlgoTrack/internal/repo/task"
	userrepo "github.com/aseptimu/AlgoTrack/internal/repo/user"
	"github.com/aseptimu/AlgoTrack/internal/service/submission"
	"github.com/aseptimu/AlgoTrack/internal/service/task"
	"github.com/aseptimu/AlgoTrack/internal/service/user"

	tgbot "github.com/go-telegram/bot"
	tgmodels "github.com/go-telegram/bot/models"
)

type stdoutSender struct{}

func (stdoutSender) SendMessage(_ context.Context, p *tgbot.SendMessageParams) (*tgmodels.Message, error) {
	fmt.Printf("\n=== STUB SEND chat=%v ===\n%s\n=========================\n\n", p.ChatID, p.Text)
	return &tgmodels.Message{}, nil
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		logger.Error("DATABASE_URL is required")
		os.Exit(1)
	}

	interval := 15 * time.Second
	if v := os.Getenv("POLLER_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			interval = d
		}
	}
	limit := 5
	if v := os.Getenv("POLLER_SUBMISSIONS_LIMIT"); v != "" {
		if n, err := fmt.Sscanf(v, "%d", &limit); err != nil || n != 1 {
			limit = 5
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	database, err := db.NewDB(ctx, dbURL)
	if err != nil {
		logger.Error("connect db", "err", err)
		os.Exit(1)
	}
	defer database.Pool.Close()

	leetcodeClient := client.NewHTTPLeetCodeClient(logger)
	tgUserRepo := userrepo.NewTgUserRepo(database)
	tRepo := taskrepo.NewTaskRepo(database)
	tgUserService := user.NewUserService(tgUserRepo, tRepo, logger)
	taskService := task.NewTaskService(tgUserService, tRepo, leetcodeClient, logger)

	poller := submission.NewPollerWithOptions(
		leetcodeClient,
		tgUserService,
		taskService,
		stdoutSender{},
		logger,
		submission.Options{
			Enabled:          true,
			Interval:         interval,
			SubmissionsLimit: limit,
		},
	)

	logger.Info("smoke runner starting", "interval", interval, "limit", limit)

	if os.Getenv("SMOKE_MODE") == "oneshot" {
		// Two manual polls. First is "cold" (lastSeen empty) so every recent
		// accepted submission for every linked user fires a notification.
		// Second proves dedup: should be silent if no new submissions appeared
		// in between.
		logger.Info("oneshot: first poll (cold cache, all recent submissions are 'new')")
		poller.Poll(ctx)
		logger.Info("oneshot: second poll (warm cache, expecting silence)")
		poller.Poll(ctx)
		logger.Info("oneshot: done")
		return
	}

	poller.Start(ctx)
	logger.Info("smoke runner stopped")
}
