package config

import (
	"log"
	"os"
)

type Config struct {
	BotToken         string
	HostUSA          string
	HostFIN          string
	Password         string
	MiniAppURL       string
	NodeExporterHost string
	JobsFilePath     string
}

func Load() *Config {
	cfg := &Config{
		BotToken:         os.Getenv("BOT_TOKEN"),
		HostUSA:          os.Getenv("HOST_USA"),
		HostFIN:          os.Getenv("HOST_FIN"),
		Password:         os.Getenv("PASSWORD"),
		MiniAppURL:       os.Getenv("MINI_APP_URL"),
		NodeExporterHost: os.Getenv("NODE_EXPORTER_HOST"),
		JobsFilePath:     os.Getenv("JOBS_FILE_PATH"),
	}
	if cfg.BotToken == "" {
		log.Panic("BOT_TOKEN is empty")
	}
	if cfg.JobsFilePath == "" {
		cfg.JobsFilePath = "./jobs.json"
	}
	return cfg
}
