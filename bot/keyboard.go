package keyboard

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

type Keyboard struct {
	Keyboards map[string]tgbotapi.ReplyKeyboardMarkup
}

func SetupKeyboards() *Keyboard {
	countryKeyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("USA"),
			tgbotapi.NewKeyboardButton("FINLAND"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🔙 В начало"),
		),
	)
	profileKeyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Создать профиль"),
			tgbotapi.NewKeyboardButton("Удалить профиль"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🔙 В начало"),
		),
	)
	platformKeyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("phone"),
			tgbotapi.NewKeyboardButton("pc"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🔙 В начало"),
		),
	)
	nameKeyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🔙 В начало"),
		),
	)
	return &Keyboard{
		Keyboards: map[string]tgbotapi.ReplyKeyboardMarkup{
			"countryKeyboard":  countryKeyboard,
			"profileKeyboard":  profileKeyboard,
			"platformKeyboard": platformKeyboard,
			"nameKeyboard":     nameKeyboard,
		},
	}
}
