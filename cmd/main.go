package main

import (
	"log"
	"os"
	"regexp"
	"strings"
	keyboard "tg-vpn-bot/bot"
	"tg-vpn-bot/client"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// простая память состояния на время жизни процесса
type userState struct {
	Region        string // "USA" или "FINLAND"
	AwaitPlatform bool   // ждём выбор платформы ("Создать профиль" -> phone/pc)
	Platform      string // выбранная платформа: "phone" или "pc"
	AwaitName     bool   // ждём ввод имени (tgID)
}

func main() {
	botToken := os.Getenv("BOT_TOKEN")
	if botToken == "" {
		log.Panic("BOT_TOKEN is empty")
	}

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err)
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	// память состояний пользователей по chatID
	states := make(map[int64]*userState)

	keyboards := keyboard.SetupKeyboards().Keyboards

	// валидатор имени: 1..12 символов, [a-zA-Z0-9_-]
	nameRe := regexp.MustCompile(`^[a-zA-Z0-9_-]{1,12}$`)

	for update := range updates {
		if update.Message == nil {
			continue
		}
		chatID := update.Message.Chat.ID
		text := strings.TrimSpace(update.Message.Text)

		// инициализируем состояние, если нужно
		if _, ok := states[chatID]; !ok {
			states[chatID] = &userState{}
		}
		st := states[chatID]

		if st.AwaitName && text != "🔙 В начало" {
			// проверим имя
			if !nameRe.MatchString(text) {
				msg := tgbotapi.NewMessage(chatID, "Введите имя туннеля — до 12 символов, латиница/цифры/_- (желательно своё имя или ник в дискорде) :")
				msg.ReplyMarkup = keyboards["nameKeyboard"]
				bot.Send(msg)
				continue
			}

			// собрать параметры для вызова клиента
			var host string
			if st.Region == "FINLAND" {
				host = os.Getenv("HOST_FIN")
			} else {
				host = os.Getenv("HOST_USA")
			}
			password := os.Getenv("PASSWORD")
			if host == "" || password == "" {
				log.Println("HOST or PASSWORD is empty")
				msg := tgbotapi.NewMessage(chatID, "Техническая ошибка конфигурации (HOST/PASSWORD). Обратитесь к администратору.")
				msg.ReplyMarkup = keyboards["countryKeyboard"]
				bot.Send(msg)
				// сброс в начало
				st.Region, st.Platform, st.AwaitPlatform, st.AwaitName = "", "", false, false
				continue
			}

			// region в нижнем регистре для API
			region := "u"
			if st.Region == "FINLAND" {
				region = "f"
			}
			tgID := text
			platform := "i" // "phone"
			if st.Platform == "pc" {
				platform = "p" // "pc"
			}

			path, err := client.DoCreateConfig(host, password, region, tgID, platform)
			if err != nil {
				log.Println("DoCreateConfig error:", err)
				msg := tgbotapi.NewMessage(chatID, "Не удалось создать профиль: "+err.Error())
				msg.ReplyMarkup = keyboards["countryKeyboard"]
				bot.Send(msg)
				st.Region, st.Platform, st.AwaitPlatform, st.AwaitName = "", "", false, false
				continue
			}

			// отправляем файл
			doc := tgbotapi.NewDocument(chatID, tgbotapi.FilePath(path))
			doc.Caption = "Ваш профиль готов ✅"
			if _, err := bot.Send(doc); err != nil {
				log.Println("Ошибка при отправке файла:", err)
			}
			// удаляем файл после отправки
			if err := os.Remove(path); err != nil {
				log.Println("Ошибка при удалении файла:", err)
			}

			// сброс состояния и возврат в начало
			st.Region, st.Platform, st.AwaitPlatform, st.AwaitName = "", "", false, false
			msg := tgbotapi.NewMessage(chatID, "Выберите страну:")
			msg.ReplyMarkup = keyboards["countryKeyboard"]
			bot.Send(msg)
			continue
		}

		switch text {

		case "/start", "🔙 В начало":
			// полный сброс
			st.Region, st.Platform, st.AwaitPlatform, st.AwaitName = "", "", false, false

			msg := tgbotapi.NewMessage(chatID, "Выберите страну:")
			msg.ReplyMarkup = keyboards["countryKeyboard"]
			bot.Send(msg)

		case "USA", "FINLAND":
			// сохраняем регион
			st.Region = text
			st.AwaitPlatform = false
			st.Platform = ""
			st.AwaitName = false

			msg := tgbotapi.NewMessage(chatID, "Вы выбрали: "+st.Region+"\nДальше выберите действие:")
			msg.ReplyMarkup = keyboards["profileKeyboard"]
			bot.Send(msg)

		case "Создать профиль":
			if st.Region == "" {
				msg := tgbotapi.NewMessage(chatID, "Сначала выберите страну:")
				msg.ReplyMarkup = keyboards["countryKeyboard"]
				bot.Send(msg)
				continue
			}
			// ждём выбор платформы
			st.AwaitPlatform = true
			st.Platform = ""
			st.AwaitName = false

			msg := tgbotapi.NewMessage(chatID, "Выберите платформу:")
			msg.ReplyMarkup = keyboards["platformKeyboard"]
			bot.Send(msg)

		case "Удалить профиль":
			// здесь можно вызвать ваш метод удаления, если он есть
			// пока просто сообщим
			msg := tgbotapi.NewMessage(chatID, "Профиль удалён ❌")
			msg.ReplyMarkup = keyboards["profileKeyboard"]
			bot.Send(msg)

		case "phone", "pc":
			// реагируем на выбор платформы только если мы действительно её ждём
			if !st.AwaitPlatform {
				msg := tgbotapi.NewMessage(chatID, "Нажмите «Создать профиль», а затем выберите платформу.")
				msg.ReplyMarkup = keyboards["profileKeyboard"]
				bot.Send(msg)
				continue
			}
			if st.Region == "" {
				msg := tgbotapi.NewMessage(chatID, "Сначала выберите страну:")
				msg.ReplyMarkup = keyboards["countryKeyboard"]
				bot.Send(msg)
				continue
			}

			// сохраняем платформу и переходим к вводу имени
			st.Platform = strings.ToLower(text) // "phone" / "pc"
			st.AwaitPlatform = false
			st.AwaitName = true

			msg := tgbotapi.NewMessage(chatID, "Введите имя туннеля — до 12 символов, латиница/цифры/_- (желательно своё имя или ник в дискорде) :")
			msg.ReplyMarkup = keyboards["nameKeyboard"]
			bot.Send(msg)

		default:
			msg := tgbotapi.NewMessage(chatID, "Не понял команду 😅")
			// показываем контекстное меню: если есть регион — меню профиля, иначе старт
			if st.Region == "" {
				msg.ReplyMarkup = keyboards["countryKeyboard"]
			} else if st.AwaitPlatform {
				msg.ReplyMarkup = keyboards["platformKeyboard"]
			} else if st.AwaitName {
				msg.ReplyMarkup = keyboards["nameKeyboard"]
			} else {
				msg.ReplyMarkup = keyboards["profileKeyboard"]
			}
			bot.Send(msg)
		}
	}
}
