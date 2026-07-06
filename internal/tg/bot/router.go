package bot

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func Route(ctx *HandlerContext, upd tgbotapi.Update) {
	if upd.Message == nil {
		return
	}
	chatID := upd.Message.Chat.ID
	text := upd.Message.Text

	// init state
	if _, ok := ctx.States[chatID]; !ok {
		ctx.States[chatID] = &UserState{Step: StepChooseRegion}
	}
	switch {
	case text == "/start":
		handleXrayInstruction(ctx, chatID)
	case text == "❓ Инструкция по xray":
		handleXrayInstruction(ctx, chatID)
	default:
		handleXrayInstruction(ctx, chatID)
	}
}
