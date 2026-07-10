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
	ProxyScheme      string
	ProxyHost        string
	ProxyUser        string
	ProxyPassword    string
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
		ProxyScheme:      os.Getenv("PROXY_SCHEME"),
		ProxyHost:        os.Getenv("PROXY_HOST"),
		ProxyUser:        os.Getenv("PROXY_USER"),
		ProxyPassword:    os.Getenv("PROXY_PASSWORD"),
	}
	if cfg.BotToken == "" {
		log.Panic("BOT_TOKEN is empty")
	}
	if cfg.JobsFilePath == "" {
		cfg.JobsFilePath = "./jobs.json"
	}
	if cfg.ProxyScheme == "" {
		cfg.ProxyScheme = "http"
	}
	return cfg
}
