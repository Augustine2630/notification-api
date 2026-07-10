package main

import (
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"notification-api/internal/config"
	"notification-api/internal/notification/controller"
	"notification-api/internal/notification/job"
	notifsender "notification-api/internal/notification/sender"
	notifservice "notification-api/internal/notification/service"
	"notification-api/internal/tg/bot"
	"notification-api/internal/vpnprofile"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	cfg := config.Load()

	httpClient := &http.Client{
		Timeout: 120 * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyURL(buildProxyURL(cfg)),
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   15 * time.Second,
			ExpectContinueTimeout: 30 * time.Second,
		},
	}
	botAPI, err := tgbotapi.NewBotAPIWithClient(cfg.BotToken, tgbotapi.APIEndpoint, httpClient)
	if err != nil {
		log.Panic(err)
	}

	log.Printf("Authorized on account %s", botAPI.Self.UserName)

	// Notification stack: sender talks to Telegram, service composes messages,
	// controller exposes it over HTTP. Every outbound message in the app flows
	// through notifier.
	sender := notifsender.NewTelegramSender(botAPI)
	notifier := notifservice.NewNotificationService(sender)

	ctx := &bot.HandlerContext{
		Notifier:   notifier,
		States:     make(map[int64]*bot.UserState),
		Keyboards:  bot.SetupKeyboards().Keyboards,
		Service:    &vpnprofile.ProfileService{HostUSA: cfg.HostUSA, HostFIN: cfg.HostFIN, Password: cfg.Password},
		MiniAppURL: cfg.MiniAppURL,
	}

	// Start battery monitoring service
	alertChatIDs := []int64{422714320, 1075418720}
	batteryService := vpnprofile.NewBatteryService(notifier, cfg.NodeExporterHost, alertChatIDs)
	batteryService.Start()
	defer batteryService.Stop()

	// Start scheduled announcement jobs, if a jobs file is present
	if _, statErr := os.Stat(cfg.JobsFilePath); statErr == nil {
		scheduler, schedErr := job.NewScheduler(notifier, cfg.JobsFilePath)
		if schedErr != nil {
			log.Printf("Scheduler: failed to load jobs from %s: %v", cfg.JobsFilePath, schedErr)
		} else {
			scheduler.Start()
			defer scheduler.Stop()
		}
	} else {
		log.Printf("Scheduler: no jobs file at %s, skipping", cfg.JobsFilePath)
	}

	// Start HTTP server in goroutine
	go startHTTPServer(notifier)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 55
	updates := botAPI.GetUpdatesChan(u)

	for update := range updates {
		bot.Route(ctx, update)
	}
}

func startHTTPServer(notifier *notifservice.NotificationService) {
	mux := http.NewServeMux()
	controller.NewController(notifier).RegisterRoutes(mux)

	log.Println("HTTP server listening on :80")
	if err := http.ListenAndServe(":80", mux); err != nil {
		log.Println("HTTP server error:", err)
	}
}

// buildProxyURL builds the outbound proxy for Telegram API requests from config.
// Returns nil (no proxy) if PROXY_HOST isn't set.
func buildProxyURL(cfg *config.Config) *url.URL {
	if cfg.ProxyHost == "" {
		return nil
	}

	proxyURL := &url.URL{
		Scheme: cfg.ProxyScheme,
		Host:   cfg.ProxyHost,
	}
	if cfg.ProxyUser != "" {
		proxyURL.User = url.UserPassword(cfg.ProxyUser, cfg.ProxyPassword)
	}
	return proxyURL
}
