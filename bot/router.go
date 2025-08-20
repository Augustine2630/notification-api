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
	st := ctx.States[chatID]

	switch {
	case text == "/start" || text == "🔙 В начало":
		handleStart(ctx, chatID)
	case text == "USA" || text == "FINLAND":
		handleRegion(ctx, chatID, text)
	case text == "Создать профиль":
		handleCreate(ctx, chatID)
	case text == "Удалить профиль":
		handleDelete(ctx, chatID)
	case text == "phone" || text == "pc":
		handlePlatform(ctx, chatID, text)
	case st.Step == StepEnterName:
		handleName(ctx, chatID, text)
	default:
		handleDefault(ctx, chatID, st)
	}
}
