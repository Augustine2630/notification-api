package main

import (
	"log"
	"tg-vpn-bot/bot"
	"tg-vpn-bot/config"
	"tg-vpn-bot/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	cfg := config.Load()

	botAPI, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		log.Panic(err)
	}
	log.Printf("Authorized on account %s", botAPI.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := botAPI.GetUpdatesChan(u)

	ctx := &bot.HandlerContext{
		Bot:       botAPI,
		States:    make(map[int64]*bot.UserState),
		Keyboards: bot.SetupKeyboards().Keyboards,
		Service:   &service.ProfileService{HostUSA: cfg.HostUSA, HostFIN: cfg.HostFIN, Password: cfg.Password},
	}

	for update := range updates {
		bot.Route(ctx, update)
	}
}
