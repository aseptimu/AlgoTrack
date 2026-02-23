package config

import "github.com/caarlos0/env/v11"

type Config struct {
	BotToken string `env:"TELEGRAM_TOKEN"`
	DBURL    string `env:"DATABASE_URL"`
}

func NewConfig() (*Config, error) {
	var cfg Config
	err := env.Parse(&cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
