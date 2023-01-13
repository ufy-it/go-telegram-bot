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
		return NewStatelessHandler(
			[]ConversationStep{
				{
					Action: func() (StepResult, error) {
						return ActionResultWithError(EndConversation, handler(ctx, conversation))
					},
				},
			})
	}
}

// EmptyHandlerCreator is a helper function to create handler with just one step and no user-data, the handler does nothing
func EmptyHandlerCreator() HandlerCreatorType {
	return OneStepHandlerCreator(func(ctx context.Context, conversation readers.BotConversation) error {
		return nil
	})
}
