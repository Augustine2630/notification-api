package sender

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// TelegramSender is the single place that talks to the Telegram Bot API.
// Every outbound message in the service — bot replies, approval webhooks,
// announcements, battery alerts — goes through here.
type TelegramSender struct {
	bot *tgbotapi.BotAPI
}

func NewTelegramSender(bot *tgbotapi.BotAPI) *TelegramSender {
	return &TelegramSender{bot: bot}
}

// SendText sends a plain (optionally keyboard-attached) text message.
func (s *TelegramSender) SendText(chatID int64, text string, kb *tgbotapi.ReplyKeyboardMarkup) error {
	msg := tgbotapi.NewMessage(chatID, text)
	if kb != nil {
		msg.ReplyMarkup = kb
	}
	_, err := s.bot.Send(msg)
	return err
}

// SendHTML sends an HTML-formatted text message, optionally with an inline keyboard payload.
func (s *TelegramSender) SendHTML(chatID int64, html string, replyMarkup interface{}) error {
	msg := tgbotapi.MessageConfig{
		BaseChat: tgbotapi.BaseChat{
			ChatID:      chatID,
			ReplyMarkup: replyMarkup,
		},
		Text:      html,
		ParseMode: "HTML",
	}
	_, err := s.bot.Send(msg)
	return err
}

// SendPhotoBytes sends an in-memory image (e.g. a generated QR code).
func (s *TelegramSender) SendPhotoBytes(chatID int64, filename string, data []byte, caption string) error {
	photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileBytes{Name: filename, Bytes: data})
	if caption != "" {
		photo.Caption = caption
	}
	_, err := s.bot.Send(photo)
	return err
}

// SendPhotoURL sends a photo referenced by URL (e.g. an announcement image).
func (s *TelegramSender) SendPhotoURL(chatID int64, url, caption string) error {
	photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileURL(url))
	photo.Caption = caption
	photo.ParseMode = "HTML"
	_, err := s.bot.Send(photo)
	return err
}

// SendDocument sends a local file as a document attachment (e.g. a VPN config file).
func (s *TelegramSender) SendDocument(chatID int64, path, caption string) error {
	doc := tgbotapi.NewDocument(chatID, tgbotapi.FilePath(path))
	if caption != "" {
		doc.Caption = caption
	}
	_, err := s.bot.Send(doc)
	return err
}
