package handlers

import (
	"ufygobot/handlers/readers"

	tgbotapi "github.com/Syfaro/telegram-bot-api"
)

// deafaultHandler will be called when there is no handler that matches first message from a user
var defaultHandlerCreator = func(conversation readers.BotConversation, firstMessage *tgbotapi.Message) convrsationHandler {
	return &baseHandler{
		conversation: conversation,
		steps: []conversationStep{
			{
				action: func(conversation readers.BotConversation) (stepActionResult, error) {
					msg := tgbotapi.NewMessage(conversation.ChatID(),
						"К сожалению, бот не может обработать ваш запрос. Убедитесь в правильности команды или обратитесь в поддержку.")
					err := conversation.SendBotMessage(msg)
					return endConversation(), err
				},
			},
		},
	}
}
