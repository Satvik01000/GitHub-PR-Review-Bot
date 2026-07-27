package config

import (
	"os"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Server ServerConfig `toml:"server"`
	GitHub GitHubConfig `toml:"github"`
	Worker WorkerConfig `toml:"worker"`
}

type ServerConfig struct {
	Port          string `toml:"port"`
	WebhookSecret string `toml:"webhook_secret"`
}

type GitHubConfig struct {
	AppID          int64  `toml:"app_id"`
	PrivateKeyPath string `toml:"private_key_path"`
}

type WorkerConfig struct {
	MaxWorkers int `toml:"max_workers"`
	QueueSize  int `toml:"queue_size"`
}

func LoadConfig() (*Config, error) {
	data, err := os.ReadFile("config.toml")
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
