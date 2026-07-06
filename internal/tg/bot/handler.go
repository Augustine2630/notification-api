package bot

import (
	"log"
	"os"
	"regexp"
	"strings"

	"notification-api/internal/notification/service"
	"notification-api/internal/vpnprofile"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// HandlerContext holds everything the bot flow handlers need: conversation
// state, keyboards, the VPN profile service, and the notification service that
// every outbound message is sent through.
type HandlerContext struct {
	Notifier   *service.NotificationService
	States     map[int64]*UserState
	Keyboards  map[string]tgbotapi.ReplyKeyboardMarkup
	Service    *vpnprofile.ProfileService
	MiniAppURL string
}

var nameRe = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,12}$`)

func handleXrayInstruction(ctx *HandlerContext, chatID int64) {
	st := ctx.States[chatID]
	*st = UserState{Step: StepChooseRegion}
	if ctx.MiniAppURL == "" {
		ctx.MiniAppURL = "https://auth-api.uuu-uuu.tech/#app=vpn_bot"
	}

	log.Printf("User %d action", chatID)

	startKb := ctx.Keyboards["startKeyboard"]
	if err := ctx.Notifier.SendInstruction(chatID, ctx.MiniAppURL, &startKb); err != nil {
		log.Println("send instruction error:", err)
	}
}

func handleRegion(ctx *HandlerContext, chatID int64, text string) {
	st := ctx.States[chatID]
	region := text
	if text == "🇺🇸 USA" {
		region = "USA"
	} else if text == "🇫🇮 FINLAND" {
		region = "FINLAND"
	}
	st.Region = region
	st.Step = StepChoosePlatform
	send(ctx, chatID, "Выбран: "+text+"\nТеперь выбери платформу:", ctx.Keyboards["platformKeyboard"])
}

func handleCreate(ctx *HandlerContext, chatID int64) {
	st := ctx.States[chatID]
	st.Step = StepChooseRegion
	send(ctx, chatID, "Какой сервер выбираешь?", ctx.Keyboards["countryKeyboard"])
}

func handlePlatform(ctx *HandlerContext, chatID int64, text string) {
	st := ctx.States[chatID]
	if st.Region == "" {
		send(ctx, chatID, "Сначала выбери сервер", ctx.Keyboards["countryKeyboard"])
		return
	}
	platform := text
	if text == "📱 Phone" {
		platform = "phone"
	} else if text == "💻 PC" {
		platform = "pc"
	}
	st.Platform = strings.ToLower(platform)
	st.Step = StepEnterName
	send(ctx, chatID, "Придумай имя туннеля (макс 12 символов):", ctx.Keyboards["nameKeyboard"])
}

func handleName(ctx *HandlerContext, chatID int64, text string) {
	st := ctx.States[chatID]

	if !nameRe.MatchString(text) {
		send(ctx, chatID, "Неверный формат имени, попробуйте ещё раз:", ctx.Keyboards["nameKeyboard"])
		return
	}

	path, qrPng, err := ctx.Service.Create(st.Region, st.Platform, text)
	if err != nil {
		log.Println("Create error:", err)
		send(ctx, chatID, "Не удалось создать профиль: "+err.Error(), ctx.Keyboards["countryKeyboard"])
		*st = UserState{Step: StepChooseRegion}
		return
	}

	if err := ctx.Notifier.SendProfileReady(chatID, qrPng, path); err != nil {
		log.Println("send profile ready error:", err)
	}
	if err := os.Remove(path); err != nil {
		log.Println("Ошибка при удалении файла:", err)
	}

	*st = UserState{Step: StepChooseRegion}
	send(ctx, chatID, "✅ Профиль готов! Что дальше?", ctx.Keyboards["startKeyboard"])
}

func handleDefault(ctx *HandlerContext, chatID int64, st *UserState) {
	kb := ctx.Keyboards["startKeyboard"]
	switch st.Step {
	case StepChooseRegion:
		kb = ctx.Keyboards["countryKeyboard"]
	case StepChoosePlatform:
		kb = ctx.Keyboards["platformKeyboard"]
	case StepEnterName:
		kb = ctx.Keyboards["nameKeyboard"]
	default:
		kb = ctx.Keyboards["startKeyboard"]
	}
	send(ctx, chatID, "❓ Непонятно, попробуй ещё раз", kb)
}

func send(ctx *HandlerContext, chatID int64, text string, kb tgbotapi.ReplyKeyboardMarkup) {
	if err := ctx.Notifier.SendText(chatID, text, &kb); err != nil {
		log.Println("send error:", err)
	}
}
