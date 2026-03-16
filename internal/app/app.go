package app

import (
	"context"
	"github.com/aseptimu/AlgoTrack/internal/client"
	"github.com/aseptimu/AlgoTrack/internal/config"
	"github.com/aseptimu/AlgoTrack/internal/db"
	task2 "github.com/aseptimu/AlgoTrack/internal/repo/task"
	user2 "github.com/aseptimu/AlgoTrack/internal/repo/user"
	"github.com/aseptimu/AlgoTrack/internal/service/task"
	"github.com/aseptimu/AlgoTrack/internal/service/user"
	"github.com/aseptimu/AlgoTrack/internal/telegram"
	"github.com/aseptimu/AlgoTrack/internal/telegram/commands/add"
	"github.com/aseptimu/AlgoTrack/internal/telegram/commands/goal"
	"github.com/aseptimu/AlgoTrack/internal/telegram/commands/setgoal"
	"github.com/aseptimu/AlgoTrack/internal/telegram/commands/start"
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

	leetcodeClient := client.NewHTTPLeetCodeClient()

	tgUserRepo := user2.NewTgUserRepo(database)
	tgUserService := user.NewUserService(tgUserRepo, logger)

	taskRepo := task2.NewTaskRepo(database)
	taskService := task.NewTaskService(tgUserService, taskRepo, leetcodeClient, logger)

	myBot, err := telegram.New(cfg.BotToken, logger)
	if err != nil {
		logger.Error("Failed to init telegram bot", "Error", err.Error())
		return err
	}

	startHandler := start.New(tgUserService, logger)
	addHandler := add.New(taskService, logger)
	textHandler := fallback.New(logger)
	goalCallbackHandler := goal.New(tgUserService, logger)
	setGoalHandler := setgoal.New(tgUserService, logger)

	router.Register(myBot.Raw(), router.Handlers{
		Start:        startHandler,
		Add:          addHandler,
		Text:         textHandler,
		GoalCallback: goalCallbackHandler,
		SetGoal:      setGoalHandler,
	})

	myBot.Run(ctx)
	return nil
}
