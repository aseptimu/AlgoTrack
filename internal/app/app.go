package app

import (
	"context"
	"github.com/aseptimu/AlgoTrack/internal/client"
	"github.com/aseptimu/AlgoTrack/internal/config"
	"github.com/aseptimu/AlgoTrack/internal/db"
	"github.com/aseptimu/AlgoTrack/internal/repo"
	"github.com/aseptimu/AlgoTrack/internal/service"
	"github.com/aseptimu/AlgoTrack/internal/telegram"
	"github.com/aseptimu/AlgoTrack/internal/telegram/commands/add"
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

	tgUserRepo := repo.NewTgUserRepo(database)
	tgUserService := service.NewUserService(tgUserRepo, logger)

	taskRepo := repo.NewTaskRepo(database)
	taskService := service.NewTaskService(tgUserService, taskRepo, leetcodeClient, logger)

	myBot, err := telegram.New(cfg.BotToken, logger)
	if err != nil {
		logger.Error("Failed to init telegram bot", "Error", err.Error())
		return err
	}

	startHandler := start.New()
	addHandler := add.New(taskService, logger)
	textHandler := fallback.New(logger)

	router.Register(myBot.Raw(), router.Handlers{
		Start: startHandler,
		Add:   addHandler,
		Text:  textHandler,
	})

	myBot.Run(ctx)
	return nil
}
