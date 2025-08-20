package bot

import (
	"log"
	"os"
	"regexp"
	"strings"
	"tg-vpn-bot/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type HandlerContext struct {
	Bot       *tgbotapi.BotAPI
	States    map[int64]*UserState
	Keyboards map[string]tgbotapi.ReplyKeyboardMarkup
	Service   *service.ProfileService
}

var nameRe = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,12}$`)

func handleStart(ctx *HandlerContext, chatID int64) {
	st := ctx.States[chatID]
	*st = UserState{Step: StepChooseRegion}
	send(ctx, chatID, "Выберите страну:", ctx.Keyboards["countryKeyboard"])
}

func handleRegion(ctx *HandlerContext, chatID int64, text string) {
	st := ctx.States[chatID]
	st.Region = text
	st.Step = StepChooseAction
	send(ctx, chatID, "Вы выбрали: "+text+"\nДальше выберите действие:", ctx.Keyboards["profileKeyboard"])
}

func handleCreate(ctx *HandlerContext, chatID int64) {
	st := ctx.States[chatID]
	if st.Region == "" {
		send(ctx, chatID, "Сначала выберите страну:", ctx.Keyboards["countryKeyboard"])
		return
	}
	st.Step = StepChoosePlatform
	send(ctx, chatID, "Выберите платформу:", ctx.Keyboards["platformKeyboard"])
}

func handlePlatform(ctx *HandlerContext, chatID int64, text string) {
	st := ctx.States[chatID]
	if st.Step != StepChoosePlatform {
		send(ctx, chatID, "Сначала нажмите «Создать профиль».", ctx.Keyboards["profileKeyboard"])
		return
	}
	st.Platform = strings.ToLower(text)
	st.Step = StepEnterName
	send(ctx, chatID, "Введите имя туннеля (до 12 символов, латиница/цифры/_-):", ctx.Keyboards["nameKeyboard"])
}

func handleName(ctx *HandlerContext, chatID int64, text string) {
	st := ctx.States[chatID]

	if !nameRe.MatchString(text) {
		send(ctx, chatID, "Неверный формат имени, попробуйте ещё раз:", ctx.Keyboards["nameKeyboard"])
		return
	}

	path, err := ctx.Service.Create(st.Region, st.Platform, text)
	if err != nil {
		log.Println("Create error:", err)
		send(ctx, chatID, "Не удалось создать профиль: "+err.Error(), ctx.Keyboards["countryKeyboard"])
		*st = UserState{Step: StepChooseRegion}
		return
	}

	// отправляем файл
	doc := tgbotapi.NewDocument(chatID, tgbotapi.FilePath(path))
	doc.Caption = "Ваш профиль готов ✅"
	if _, err := ctx.Bot.Send(doc); err != nil {
		log.Println("Ошибка при отправке файла:", err)
	}
	if err := os.Remove(path); err != nil {
		log.Println("Ошибка при удалении файла:", err)
	}

	*st = UserState{Step: StepChooseRegion}
	send(ctx, chatID, "Выберите страну:", ctx.Keyboards["countryKeyboard"])
}

func handleDelete(ctx *HandlerContext, chatID int64) {
	send(ctx, chatID, "Профиль удалён ❌", ctx.Keyboards["profileKeyboard"])
}

func handleDefault(ctx *HandlerContext, chatID int64, st *UserState) {
	kb := ctx.Keyboards["countryKeyboard"]
	switch st.Step {
	case StepChoosePlatform:
		kb = ctx.Keyboards["platformKeyboard"]
	case StepEnterName:
		kb = ctx.Keyboards["nameKeyboard"]
	case StepChooseAction:
		kb = ctx.Keyboards["profileKeyboard"]
	default:
		kb = ctx.Keyboards["countryKeyboard"]
	}
	send(ctx, chatID, "Не понял команду 😅", kb)
}

func send(ctx *HandlerContext, chatID int64, text string, kb tgbotapi.ReplyKeyboardMarkup) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = kb
	if _, err := ctx.Bot.Send(msg); err != nil {
		log.Println("send error:", err)
	}
}
