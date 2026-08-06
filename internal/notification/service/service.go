package service

import (
	"encoding/json"
	"fmt"

	"notification-api/internal/notification/sender"
	"notification-api/internal/notification/template"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// NotificationService is the single entry point for every outbound message the
// service produces — bot replies, approval webhooks, announcements, and battery
// alerts all funnel through here so message composition (template) and delivery
// (sender) stay in one place.
type NotificationService struct {
	sender *sender.TelegramSender
}

func NewNotificationService(sender *sender.TelegramSender) *NotificationService {
	return &NotificationService{sender: sender}
}

// AnnounceResult reports per-recipient delivery outcome for a fan-out send.
type AnnounceResult struct {
	Sent      int
	Failed    int
	FailedIDs []int64
	Errors    []string
}

// SendInstruction sends the bot onboarding/instruction message with an inline
// mini-app button, then a plain follow-up text with the given keyboard.
func (s *NotificationService) SendInstruction(chatID int64, miniAppURL string, followUpKb *tgbotapi.ReplyKeyboardMarkup) error {
	text := template.Instruction()

	if miniAppURL != "" {
		if err := s.sender.SendHTML(chatID, text, newWebAppMarkup("🚀 Открыть приложение", miniAppURL)); err != nil {
			return fmt.Errorf("send instruction: %w", err)
		}
	} else {
		if err := s.sender.SendText(chatID, text, followUpKb); err != nil {
			return fmt.Errorf("send instruction: %w", err)
		}
		return nil
	}

	return s.sender.SendText(chatID, "Что дальше?", followUpKb)
}

// SendText is a passthrough used by bot flow handlers for plain prompts
// (region/platform/name steps, validation errors, etc).
func (s *NotificationService) SendText(chatID int64, text string, kb *tgbotapi.ReplyKeyboardMarkup) error {
	return s.sender.SendText(chatID, text, kb)
}

// SendProfileReady delivers the freshly generated VPN profile: a QR code photo
// followed by the config file document.
func (s *NotificationService) SendProfileReady(chatID int64, qrPng []byte, configPath string) error {
	if err := s.sender.SendPhotoBytes(chatID, "qr.png", qrPng, ""); err != nil {
		return fmt.Errorf("send qr photo: %w", err)
	}
	if err := s.sender.SendDocument(chatID, configPath, template.ProfileReadyCaption()); err != nil {
		return fmt.Errorf("send config document: %w", err)
	}
	return nil
}

// SendApprove notifies a user that their access request was approved or rejected.
func (s *NotificationService) SendApprove(telegramID int64, approved bool) (string, error) {
	text, err := template.RenderApprove(template.ApproveData{Approved: approved})
	if err != nil {
		return "", err
	}
	if err := s.sender.SendText(telegramID, text, nil); err != nil {
		return "", fmt.Errorf("send approve: %w", err)
	}
	return text, nil
}

// SendAnnounce fans a text (optionally with an image) out to a list of recipients,
// tracking per-recipient failures instead of aborting on the first error.
// Duplicate recipient IDs are sent to only once.
func (s *NotificationService) SendAnnounce(recipients []int64, text, imageLink string) AnnounceResult {
	result := AnnounceResult{}

	seen := make(map[int64]struct{}, len(recipients))
	uniqueRecipients := make([]int64, 0, len(recipients))
	for _, userID := range recipients {
		if _, ok := seen[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}
		uniqueRecipients = append(uniqueRecipients, userID)
	}

	for _, userID := range uniqueRecipients {
		var err error
		if imageLink != "" {
			err = s.sender.SendPhotoURL(userID, imageLink, text)
		} else {
			err = s.sender.SendHTML(userID, text, nil)
		}

		if err != nil {
			result.Failed++
			result.FailedIDs = append(result.FailedIDs, userID)
			result.Errors = append(result.Errors, fmt.Sprintf("User %d: %v", userID, err))
			continue
		}
		result.Sent++
	}

	return result
}

// SendBatteryAlert notifies the given chat IDs that battery capacity dropped below threshold.
func (s *NotificationService) SendBatteryAlert(chatIDs []int64, capacity int) []error {
	text, err := template.RenderBatteryAlert(template.BatteryAlertData{Capacity: capacity})
	if err != nil {
		return []error{err}
	}

	var errs []error
	for _, chatID := range chatIDs {
		if sendErr := s.sender.SendHTML(chatID, text, nil); sendErr != nil {
			errs = append(errs, fmt.Errorf("chat %d: %w", chatID, sendErr))
		}
	}
	return errs
}

// webAppButton is a custom inline button that opens a Telegram Mini App.
// The go-telegram-bot-api v5 InlineKeyboardButton type doesn't expose a
// web_app field, so the markup is built and marshalled by hand, same as the
// original bot code did.
type webAppButton struct {
	Text   string `json:"text"`
	WebApp struct {
		URL string `json:"url"`
	} `json:"web_app"`
}

func newWebAppMarkup(text, url string) json.RawMessage {
	btn := webAppButton{Text: text}
	btn.WebApp.URL = url
	data, _ := json.Marshal(map[string]interface{}{
		"inline_keyboard": [][]webAppButton{{btn}},
	})
	return data
}
