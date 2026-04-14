package app

import (
	"context"
	"time"

	"github.com/aseptimu/AlgoTrack/internal/client"
	"github.com/aseptimu/AlgoTrack/internal/config"
	"github.com/aseptimu/AlgoTrack/internal/db"
	task2 "github.com/aseptimu/AlgoTrack/internal/repo/task"
	user2 "github.com/aseptimu/AlgoTrack/internal/repo/user"
	"github.com/aseptimu/AlgoTrack/internal/service/recommend"
	reviewsvc "github.com/aseptimu/AlgoTrack/internal/service/review"
	"github.com/aseptimu/AlgoTrack/internal/service/submission"
	"github.com/aseptimu/AlgoTrack/internal/service/task"
	"github.com/aseptimu/AlgoTrack/internal/service/user"
	"github.com/aseptimu/AlgoTrack/internal/telegram"
	"github.com/aseptimu/AlgoTrack/internal/telegram/commands/add"
	"github.com/aseptimu/AlgoTrack/internal/telegram/commands/goal"
	helpcmd "github.com/aseptimu/AlgoTrack/internal/telegram/commands/help"
	"github.com/aseptimu/AlgoTrack/internal/telegram/commands/link"
	"github.com/aseptimu/AlgoTrack/internal/telegram/commands/list"
	modecmd "github.com/aseptimu/AlgoTrack/internal/telegram/commands/mode"
	nextcmd "github.com/aseptimu/AlgoTrack/internal/telegram/commands/next"
	reviewcmd "github.com/aseptimu/AlgoTrack/internal/telegram/commands/review"
	"github.com/aseptimu/AlgoTrack/internal/telegram/commands/setgoal"
	"github.com/aseptimu/AlgoTrack/internal/telegram/commands/start"
	"github.com/aseptimu/AlgoTrack/internal/telegram/commands/stats"
	"github.com/aseptimu/AlgoTrack/internal/telegram/messages/fallback"
	"github.com/aseptimu/AlgoTrack/internal/telegram/router"
)

func Run(ctx context.Context) error {
	logger := config.NewLogger()

	cfg, err := config.NewConfig()
	if err != nil {
		logger.Error("Failed to init config", "Error", err.Error())
		return err
	}

	database, err := db.NewDB(ctx, cfg.DBURL)
	if err != nil {
		logger.Error("Failed to connect to DB", "Error", err.Error())
		return err
	}
	defer database.Pool.Close()

	leetcodeClient := client.NewHTTPLeetCodeClient(logger)

	tgUserRepo := user2.NewTgUserRepo(database)
	taskRepo := task2.NewTaskRepo(database)
	tgUserService := user.NewUserService(tgUserRepo, taskRepo, logger)
	taskService := task.NewTaskService(tgUserService, taskRepo, leetcodeClient, logger)
	recommendService := recommend.New(&recommendRepo{users: tgUserRepo, tasks: taskRepo}, logger)

	myBot, err := telegram.New(cfg.BotToken, logger)
	if err != nil {
		logger.Error("Failed to init telegram bot", "Error", err.Error())
		return err
	}

	startHandler := start.New(tgUserService, logger)
	addHandler := add.New(taskService, logger)
	helpHandler := helpcmd.New(logger)
	textHandler := fallback.New(logger)
	goalCallbackHandler := goal.New(tgUserService, logger)
	setGoalHandler := setgoal.New(tgUserService, logger)
	linkHandler := link.New(tgUserService, logger)
	listHandler := list.New(taskService, tgUserService, logger)
	statsHandler := stats.New(taskService, tgUserService, logger)
	nextHandler := nextcmd.New(tgUserService, recommendService, logger)
	reviewHandler := reviewcmd.New(tgUserService, taskRepo, logger)
	modeHandler := modecmd.New(tgUserService, tgUserRepo, logger)

	reminderService := reviewsvc.NewReminderService(tgUserRepo, taskRepo, recommendService, myBot.Raw(), logger)
	submissionPoller := submission.NewPollerWithOptions(
		leetcodeClient,
		tgUserService,
		taskService,
		tgUserRepo,
		myBot.Raw(),
		logger,
		submission.Options{
			Enabled:          cfg.PollerEnabled,
			Interval:         cfg.PollerInterval,
			SubmissionsLimit: cfg.PollerSubmissionsLimit,
		},
	)

	router.Register(myBot.Raw(), router.Handlers{
		Start:        startHandler,
		Add:          addHandler,
		Help:         helpHandler,
		Text:         textHandler,
		GoalCallback: goalCallbackHandler,
		SetGoal:      setGoalHandler,
		Link:         linkHandler,
		List:         listHandler,
		Stats:        statsHandler,
		Next:         nextHandler,
		Review:       reviewHandler,
		Mode:         modeHandler,
	})

	go reminderService.Start(ctx)
	go submissionPoller.Start(ctx)
	myBot.Run(ctx)
	logger.Info("bot shutting down")
	return nil
}

// recommendRepo bridges *user.TgUserRepo + *task.TaskRepo to the recommend.Repo interface.
type recommendRepo struct {
	users *user2.TgUserRepo
	tasks *task2.TaskRepo
}

func (r *recommendRepo) GetSolvedNumbers(ctx context.Context, userID int64) (map[int]bool, error) {
	return r.tasks.GetSolvedNumbers(ctx, userID)
}
func (r *recommendRepo) RecommendedNumbers(ctx context.Context, userID int64) (map[int]bool, error) {
	return r.users.RecommendedNumbers(ctx, userID)
}
func (r *recommendRepo) MarkRecommended(ctx context.Context, userID int64, n int) error {
	return r.users.MarkRecommended(ctx, userID, n)
}
func (r *recommendRepo) LastHardCreatedAt(ctx context.Context, userID int64) (time.Time, bool, error) {
	return r.tasks.LastHardCreatedAt(ctx, userID)
}
