package app

import (
	"context"

	"github.com/aseptimu/AlgoTrack/internal/client"
	"github.com/aseptimu/AlgoTrack/internal/config"
	"github.com/aseptimu/AlgoTrack/internal/db"
	task2 "github.com/aseptimu/AlgoTrack/internal/repo/task"
	user2 "github.com/aseptimu/AlgoTrack/internal/repo/user"
	"github.com/aseptimu/AlgoTrack/internal/service/review"
	"github.com/aseptimu/AlgoTrack/internal/service/submission"
	"github.com/aseptimu/AlgoTrack/internal/service/task"
	"github.com/aseptimu/AlgoTrack/internal/service/user"
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
	reminderService := review.NewReminderService(taskRepo, myBot.Raw(), logger)
	submissionPoller := submission.NewPoller(leetcodeClient, tgUserService, taskService, myBot.Raw(), logger)

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
	})

	go reminderService.Start(ctx)
	go submissionPoller.Start(ctx)
	myBot.Run(ctx)
	logger.Info("bot shutting down")
	return nil
}
