package bot

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"tg-vpn-bot/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type HandlerContext struct {
	Bot        *tgbotapi.BotAPI
	States     map[int64]*UserState
	Keyboards  map[string]tgbotapi.ReplyKeyboardMarkup
	Service    *service.ProfileService
	MiniAppURL string
}

var nameRe = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,12}$`)

// WebAppButton is a custom inline button that opens a Telegram Mini App
type WebAppButton struct {
	Text   string `json:"text"`
	WebApp struct {
		URL string `json:"url"`
	} `json:"web_app"`
}

// newWebAppMarkup creates an inline keyboard with a web_app button
func newWebAppMarkup(text, url string) json.RawMessage {
	btn := WebAppButton{
		Text: text,
	}
	btn.WebApp.URL = url
	data, _ := json.Marshal(map[string]interface{}{
		"inline_keyboard": [][]WebAppButton{{btn}},
	})
	return data
}

func handleXrayInstruction(ctx *HandlerContext, chatID int64) {
	instruction := `📖 Инструкция по использованию
<a href="https://sub.uuu-uuu.tech/docs/user-guide">PDF документ с руководством</a>

1. Откройте приложение по кнопке под этим сообщением ↓
2. Нажмите "Login" в приложении и запросите доступ
3. Я подтверждаю доступ (<a href="tg://user?id=5477149309">@Augustine_kptz</a>)
4. После получения подтверждения доступа нажать еще раз "Login" и создайте профиль
5. Скачайте Happ для нужной платформы
   <a href="https://play.google.com/store/apps/details?id=com.happproxy">Android</a> | <a href="https://github.com/Happ-proxy/happ-android/releases/latest/download/Happ.apk">Android APK</a>
   <a href="https://apps.apple.com/ru/app/happ-proxy-utility-plus/id6746188973">iOS</a>
   <a href="https://github.com/Happ-proxy/happ-desktop/releases/latest/download/setup-Happ.x64.exe">Windows</a>
   <a href="https://github.com/Happ-proxy/happ-desktop/releases/latest/download/Happ.macOS.universal.dmg">macOS</a>
6. Отсканируйте QR-код в <a href="https://www.happ.su/main/ru">Happ приложении</a> или скопируйте ссылку

Готово! VPN активирован 🎉`

	st := ctx.States[chatID]
	*st = UserState{Step: StepChooseRegion}
	if ctx.MiniAppURL == "" {
		ctx.MiniAppURL = "https://auth-api.uuu-uuu.tech/#app=vpn_bot"
	}
	log.Default().Println(fmt.Sprintf("User %d action", chatID))
	if ctx.MiniAppURL != "" {
		msgConfig := tgbotapi.MessageConfig{
			BaseChat: tgbotapi.BaseChat{
				ChatID:      chatID,
				ReplyMarkup: newWebAppMarkup("🚀 Открыть приложение", ctx.MiniAppURL),
			},
			Text:      instruction,
			ParseMode: "HTML",
		}
		if _, err := ctx.Bot.Send(msgConfig); err != nil {
			log.Println("send error:", err)
		}
	} else {
		send(ctx, chatID, instruction, ctx.Keyboards["startKeyboard"])
		return
	}

	// Follow-up message
	send(ctx, chatID, "Что дальше?", ctx.Keyboards["startKeyboard"])
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

	if _, err := ctx.Bot.Send(tgbotapi.NewPhoto(chatID, tgbotapi.FileBytes{Name: "qr.png", Bytes: qrPng})); err != nil {
		log.Println("Ошибка при отправке файла:", err)
	}
	doc := tgbotapi.NewDocument(chatID, tgbotapi.FilePath(path))
	doc.Caption = "Ваш профиль готов ✅. Отсканируйте QR или импортируйте файл конфигурации"
	if _, err := ctx.Bot.Send(doc); err != nil {
		log.Println("Ошибка при отправке файла:", err)
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
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = kb
	if _, err := ctx.Bot.Send(msg); err != nil {
		log.Println("send error:", err)
	}
}

// HandleApprove returns an HTTP handler that sends approval/rejection message to user
func HandleApprove(ctx *HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Get telegramId from query parameters
		telegramIDStr := r.URL.Query().Get("telegramId")
		if telegramIDStr == "" {
			http.Error(w, "telegramId parameter is required", http.StatusBadRequest)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"error": "telegramId is required"})
			return
		}

		// Convert string to int64
		telegramID, err := strconv.ParseInt(telegramIDStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid telegramId format", http.StatusBadRequest)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid telegramId"})
			return
		}

		// Get approved parameter (default: true)
		approvedStr := r.URL.Query().Get("approved")
		approved := true
		if approvedStr != "" {
			approved = approvedStr != "false" && approvedStr != "0"
		}

		var messageText string
		if approved {
			messageText = "✅ Доступ к xray подтвержден"
		} else {
			messageText = "❌ Доступ к xray отклонен"
		}

		// Send message to user
		msg := tgbotapi.NewMessage(telegramID, messageText)
		if _, err := ctx.Bot.Send(msg); err != nil {
			log.Println("Failed to send message:", err)
			http.Error(w, "Failed to send message", http.StatusInternalServerError)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to send message"})
			return
		}

		// Success response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":   "success",
			"message":  messageText,
			"approved": approved,
		})
	}
}
