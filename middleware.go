package main

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"

// Middleware functions types
type CommandHandlerFunc func(bot *tgbotapi.BotAPI, db Repository, msg *tgbotapi.Message)

// Helper function to avoid circular imports
func sendAdminDeniedMessage(bot *tgbotapi.BotAPI, chatID int64) {
	message := tgbotapi.NewMessage(chatID, "У вас нет прав для выполнения этой команды. Только администраторы могут выполнять это действие.")
	bot.Send(message)
}

// AdminCheckMiddleware wraps a command handler with admin verification
func AdminCheckMiddleware(handler CommandHandlerFunc) CommandHandlerFunc {
	return func(bot *tgbotapi.BotAPI, db Repository, msg *tgbotapi.Message) {
		if !IsAdmin(msg.From.UserName) {
			sendAdminDeniedMessage(bot, msg.Chat.ID)
			return
		}
		handler(bot, db, msg)
	}
}
