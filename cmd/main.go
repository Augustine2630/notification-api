package main

import (
	"log"
	"net"
	"net/http"
	"net/url"
	"tg-vpn-bot/bot"
	"tg-vpn-bot/config"
	"tg-vpn-bot/service"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	cfg := config.Load()

	//Set up proxy for Telegram API requests
	proxyURL := &url.URL{
		Scheme: "http",
		User:   url.UserPassword("tg-vpn-bot", "9DBOt3nBGdI2a4cD"),
		Host:   "139.28.97.175:3128",
	}

	client := &http.Client{
		Timeout: 120 * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   15 * time.Second,
			ExpectContinueTimeout: 30 * time.Second,
		},
	}
	botAPI, err := tgbotapi.NewBotAPIWithClient(cfg.BotToken, tgbotapi.APIEndpoint, client)
	if err != nil {
		log.Panic(err)
	}

	log.Printf("Authorized on account %s", botAPI.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 55
	updates := botAPI.GetUpdatesChan(u)

	ctx := &bot.HandlerContext{
		Bot:        botAPI,
		States:     make(map[int64]*bot.UserState),
		Keyboards:  bot.SetupKeyboards().Keyboards,
		Service:    &service.ProfileService{HostUSA: cfg.HostUSA, HostFIN: cfg.HostFIN, Password: cfg.Password},
		MiniAppURL: cfg.MiniAppURL,
	}

	// Start battery monitoring service
	alertChatIDs := []int64{422714320, 1075418720}
	batteryService := service.NewBatteryService(botAPI, cfg.NodeExporterHost, alertChatIDs)
	batteryService.Start()
	defer batteryService.Stop()

	// Start HTTP server in goroutine
	go startHTTPServer(ctx)

	for update := range updates {
		bot.Route(ctx, update)
	}
}

func startHTTPServer(ctx *bot.HandlerContext) {
	http.HandleFunc("/approve", bot.HandleApprove(ctx))
	http.HandleFunc("/api/v1/send/announce", bot.HandleAnnounce(ctx))
	log.Println("HTTP server listening on :80")
	if err := http.ListenAndServe(":80", nil); err != nil {
		log.Println("HTTP server error:", err)
	}
}
