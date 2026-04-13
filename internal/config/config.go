package config

import (
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	BotToken string `env:"TELEGRAM_TOKEN,required"`
	DBURL    string `env:"DATABASE_URL,required"`

	PollerEnabled          bool          `env:"POLLER_ENABLED" envDefault:"true"`
	PollerInterval         time.Duration `env:"POLLER_INTERVAL" envDefault:"5m"`
	PollerSubmissionsLimit int           `env:"POLLER_SUBMISSIONS_LIMIT" envDefault:"10"`
}

func NewConfig() (*Config, error) {
	var cfg Config
	err := env.Parse(&cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
