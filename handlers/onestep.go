package handlers

import (
	"context"

	"github.com/ufy-it/go-telegram-bot/handlers/readers"
)

// OneStepCommandHandlerType is a type og handler that can be used as the only step of a conversation handler
type OneStepCommandHandlerType func(ctx context.Context, conversation readers.BotConversation) error

// OneStepHandlerCreator is a helper function to create handler with just one step and no user-data
func OneStepHandlerCreator(handler OneStepCommandHandlerType) HandlerCreatorType {
	return func(ctx context.Context, conversation readers.BotConversation) Handler {
		return &StandardHandler{
			Conversation: conversation,
			Steps: []ConversationStep{
				{
					Action: func(conversation readers.BotConversation) (StepResult, error) {
						return ActionResultWithError(EndConversation, handler(ctx, conversation))
					},
				},
			},
			GetUserData: func() interface{} { return nil },
			SetUserData: func(data interface{}) error { return nil },
		}
	}
}
