package bot

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

type Keyboard struct {
	Keyboards map[string]tgbotapi.ReplyKeyboardMarkup
}

func SetupKeyboards() *Keyboard {
	startKeyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("➕ Создать профиль"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("❓ Инструкция по xray"),
		),
	)
	countryKeyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🇺🇸 USA"),
			tgbotapi.NewKeyboardButton("🇫🇮 FINLAND"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("◀️ Назад"),
		),
	)
	platformKeyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📱 Phone"),
			tgbotapi.NewKeyboardButton("💻 PC"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("◀️ Назад"),
		),
	)
	nameKeyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("◀️ Назад"),
		),
	)
	return &Keyboard{
		Keyboards: map[string]tgbotapi.ReplyKeyboardMarkup{
			"startKeyboard":    startKeyboard,
			"countryKeyboard":  countryKeyboard,
			"platformKeyboard": platformKeyboard,
			"nameKeyboard":     nameKeyboard,
		},
	}
}
