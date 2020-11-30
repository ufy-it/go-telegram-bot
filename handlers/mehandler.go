package handlers

import (
	"fmt"
	"ufygobot/handlers/readers"

	tgbotapi "github.com/Syfaro/telegram-bot-api"
)

// meHandler handles "/me" command
var meHandlerCreator = func(conversation readers.BotConversation, firstMessage *tgbotapi.Message) convrsationHandler {
	name := firstMessage.From.FirstName
	lastName := firstMessage.From.LastName
	userName := firstMessage.From.UserName
	userId := firstMessage.From.ID
	chatId := conversation.ChatID()
	return &baseHandler{
		conversation: conversation,
		steps: []conversationStep{
			{
				action: func(conversation readers.BotConversation) (stepActionResult, error) {
					msg := tgbotapi.NewMessage(chatId,
						fmt.Sprintf(`Имя: %s %s
						Ник: %s
						ID: %d`, name, lastName, userName, userId))
					err := conversation.SendBotMessage(msg)
					return nextStep(), err
				},
			},
			{
				action: func(conversation readers.BotConversation) (stepActionResult, error) {
					msg := tgbotapi.NewMessage(chatId, "Бот повторит ваше следующее сообщение")
					err := conversation.SendBotMessage(msg)
					if err != nil {
						return endConversation(), err
					}
					message, exit := conversation.ReadMessage()
					if exit {
						return closeCommand(), nil
					}
					msg = tgbotapi.NewMessage(chatId, message.Text)
					err = conversation.SendBotMessage(msg)
					return endConversation(), err
				},
			},
		},
	}
}
