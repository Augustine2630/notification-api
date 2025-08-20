package config

import (
	"log"
	"os"
)

type Config struct {
	BotToken string
	HostUSA  string
	HostFIN  string
	Password string
}

func Load() *Config {
	cfg := &Config{
		BotToken: os.Getenv("BOT_TOKEN"),
		HostUSA:  os.Getenv("HOST_USA"),
		HostFIN:  os.Getenv("HOST_FIN"),
		Password: os.Getenv("PASSWORD"),
	}
	if cfg.BotToken == "" {
		log.Panic("BOT_TOKEN is empty")
	}
	return cfg
}
