package handlers

import (
	"ufygobot/handlers/readers"

	tgbotapi "github.com/Syfaro/telegram-bot-api"
)

// OneStepCommandHandlerType is a type og handler that can be used as the only step of a conversation handler
type OneStepCommandHandlerType func(conversation readers.BotConversation, firstMessage *tgbotapi.Message) error

// OneStepHandlerCreator is a helper function to create handler with just one step and no user-data
func OneStepHandlerCreator(handler OneStepCommandHandlerType) HandlerCreatorType {
	return func(conversation readers.BotConversation, firstMessage *tgbotapi.Message) Handler {
		return &StandardHandler{
			Conversation: conversation,
			Steps: []ConversationStep{
				{
					Action: func(conversation readers.BotConversation) (StepResult, error) {
						return ActionResultWithError(EndConversation, handler(conversation, firstMessage))
					},
				},
			},
			GetUserData: func() interface{} { return nil },
			SetUserData: func(data interface{}) error { return nil },
		}
	}
}
